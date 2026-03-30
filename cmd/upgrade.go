// upgrade.go implements the upgrade subcommand.
// It checks for newer versions of mdpress from GitHub releases and optionally installs them.
package cmd

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yeasy/mdpress/pkg/utils"
)

var (
	errChecksumMismatch = errors.New("checksum mismatch")
	errNoChecksumEntry  = errors.New("no matching checksum entry")

	// mdpressUserAgent is the User-Agent header sent with upgrade HTTP requests.
	mdpressUserAgent = "mdpress/" + Version
)

// upgradeHTTPClient is a shared HTTP client with sensible timeouts for upgrade operations.
// Using http.DefaultClient has no timeout and could hang indefinitely.
// CheckRedirect validates redirect targets to prevent SSRF via DNS poisoning.
var upgradeHTTPClient = &http.Client{
	Timeout: upgradeClientTimeout,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= utils.MaxHTTPRedirects {
			return errors.New("too many redirects")
		}
		host := req.URL.Hostname()
		if !strings.HasSuffix(host, ".github.com") &&
			!strings.HasSuffix(host, ".githubusercontent.com") &&
			!strings.HasSuffix(host, ".githubassets.com") &&
			host != "github.com" {
			return fmt.Errorf("redirect to unexpected host: %s", host)
		}
		return nil
	},
}

const (
	maxBinarySize        = 500 << 20 // 500 MB
	upgradeClientTimeout = 5 * time.Minute
)

var upgradeCheckOnly bool

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Check for and install a newer version of mdpress",
	Long: `Check for newer versions of mdpress from GitHub releases and optionally install them.

By default, checks for a newer version and installs if found:
  mdpress upgrade

To only check without installing:
  mdpress upgrade --check

The upgrade command:
  - Fetches the latest release from the GitHub API
  - Compares with the current version
  - Downloads the appropriate binary for your OS/arch
  - Replaces the current mdpress binary
  - Shows progress during the download`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeUpgrade(cmd.Context(), upgradeCheckOnly)
	},
}

func init() {
	upgradeCmd.Flags().BoolVar(&upgradeCheckOnly, "check", false, "Only check for updates, do not install")
}

// gitHubRelease represents a GitHub release from the API.
type gitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func executeUpgrade(ctx context.Context, checkOnly bool) error {
	utils.Header("mdpress Upgrade Check")
	fmt.Println()

	slog.Info("checking for newer version", slog.String("current_version", Version))
	utils.Success("Current version: %s", Version)

	// Fetch the latest release from GitHub.
	latestRelease, err := fetchLatestRelease(ctx)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	latestVersion := strings.TrimPrefix(latestRelease.TagName, "v")

	utils.Success("Latest version: %s", latestVersion)

	// Compare versions.
	isNewer := isVersionNewer(latestVersion, Version)
	if !isNewer {
		fmt.Println()
		utils.Success("You are already running the latest version")
		return nil
	}

	fmt.Println()
	utils.Warning("Newer version available: %s -> %s", Version, latestVersion)

	if checkOnly {
		fmt.Println()
		fmt.Println("Run 'mdpress upgrade' to install the latest version")
		return nil
	}

	// Download and install the new version.
	fmt.Println()
	if err := installNewVersion(ctx, latestRelease, latestVersion); err != nil {
		return fmt.Errorf("failed to install upgrade: %w", err)
	}

	return nil
}

// fetchLatestRelease fetches the latest release from GitHub.
func fetchLatestRelease(ctx context.Context) (*gitHubRelease, error) {
	url := "https://api.github.com/repos/yeasy/mdpress/releases/latest"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", mdpressUserAgent)

	resp, err := upgradeHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB max for error body
		if err != nil {
			return nil, fmt.Errorf("github API returned %d and failed to read error body: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("github API returned %d: %s", resp.StatusCode, string(body))
	}

	var release gitHubRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&release); err != nil { // 10 MB max
		return nil, fmt.Errorf("failed to parse release JSON: %w", err)
	}

	if release.TagName == "" {
		return nil, errors.New("invalid release: missing tag_name")
	}

	if len(release.Assets) == 0 {
		return nil, errors.New("release has no assets")
	}

	return &release, nil
}

// isVersionNewer compares two semantic versions.
// Returns true if newVersion > currentVersion.
func isVersionNewer(newVersion, currentVersion string) bool {
	newParts := parseVersion(newVersion)
	currentParts := parseVersion(currentVersion)

	// Pad with zeros to have equal length.
	for len(newParts) < len(currentParts) {
		newParts = append(newParts, 0)
	}
	for len(currentParts) < len(newParts) {
		currentParts = append(currentParts, 0)
	}

	for i := 0; i < len(newParts); i++ {
		if newParts[i] > currentParts[i] {
			return true
		}
		if newParts[i] < currentParts[i] {
			return false
		}
	}

	return false
}

// parseVersion extracts numeric parts from a semantic version string.
// For example: "1.2.3" -> [1, 2, 3]
func parseVersion(version string) []int {
	// Remove 'v' prefix if present.
	version = strings.TrimPrefix(version, "v")

	// Split on '.' and parse each part.
	parts := []int{}
	for _, part := range strings.Split(version, ".") {
		// Extract leading digits using strings.Builder.
		var sb strings.Builder
		for _, ch := range part {
			if ch >= '0' && ch <= '9' {
				sb.WriteRune(ch)
			} else {
				break
			}
		}
		numStr := sb.String()
		if numStr != "" {
			num, err := strconv.Atoi(numStr)
			if err != nil {
				continue
			}
			parts = append(parts, num)
		}
	}

	return parts
}

// installNewVersion downloads and installs the new binary.
// Uses atomic file replacement to prevent corruption if the process crashes mid-write.
func installNewVersion(ctx context.Context, release *gitHubRelease, newVersion string) error {
	// Find the appropriate asset for this OS/arch.
	assetURL, assetName := findAssetForPlatform(release)
	if assetURL == "" {
		return fmt.Errorf("no binary found for %s/%s in release assets", runtime.GOOS, runtime.GOARCH)
	}

	slog.Info("found asset", slog.String("asset", assetName))
	utils.Success("Downloading %s", assetName)

	// Validate the download URL points to a known GitHub domain.
	if !isGitHubDownloadURL(assetURL) {
		return errors.New("asset URL has unexpected host (expected github.com or *.githubusercontent.com)")
	}

	// Download the binary.
	assetData, err := downloadBinary(ctx, assetURL)
	if err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}

	utils.Success("Downloaded %d bytes", len(assetData))

	// Verify checksum if available.
	if err := verifyChecksum(ctx, release, assetName, assetData); err != nil {
		// Abort on integrity failures (mismatch or missing entry in existing checksum file).
		// Only warn for non-critical issues (no checksum file in release, download failure).
		if errors.Is(err, errChecksumMismatch) || errors.Is(err, errNoChecksumEntry) {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		slog.Warn("checksum verification", slog.String("warning", err.Error()))
	}

	binaryData, err := extractBinaryData(assetName, assetData)
	if err != nil {
		return fmt.Errorf("failed to unpack %s: %w", assetName, err)
	}

	// Find the current mdpress executable.
	currentPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine current executable path: %w", err)
	}

	// Use atomic rename: write to temp file, then rename to target.
	// This prevents corruption if the process crashes mid-write.
	tempFile, err := os.CreateTemp(filepath.Dir(currentPath), "mdpress-upgrade-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempPath := tempFile.Name()
	tempFile.Close() //nolint:errcheck

	// Write the new binary to a temporary file.
	if err := writeBinaryFile(tempPath, binaryData); err != nil {
		// Clean up temp file if write failed.
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to write temporary binary: %w", err)
	}

	slog.Info("wrote temporary binary", slog.String("path", tempPath))

	// Create a backup of the current binary.
	backupPath := currentPath + ".backup"
	if err := os.Rename(currentPath, backupPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to create backup: %w", err)
	}

	slog.Info("created backup", slog.String("path", backupPath))

	// Atomically replace the old binary with the new one.
	if err := os.Rename(tempPath, currentPath); err != nil {
		// Try to restore from backup on error.
		if restoreErr := os.Rename(backupPath, currentPath); restoreErr != nil {
			return fmt.Errorf("failed to install binary (%w), and restore failed (%w); backup at %s", err, restoreErr, backupPath)
		}
		return fmt.Errorf("failed to install binary (backup restored): %w", err)
	}

	// Verify the new binary is functional before removing the backup.
	verifyCtx, verifyCancel := context.WithTimeout(ctx, 10*time.Second)
	defer verifyCancel()
	if out, err := exec.CommandContext(verifyCtx, currentPath, "version").CombinedOutput(); err != nil {
		slog.Error("new binary verification failed", slog.String("output", string(out)), slog.String("error", err.Error()))
		// Restore the backup since the new binary is broken.
		if restoreErr := os.Rename(backupPath, currentPath); restoreErr != nil {
			return fmt.Errorf("new binary verification failed (%w), and restore failed (%w); backup at %s", err, restoreErr, backupPath)
		}
		return fmt.Errorf("new binary verification failed (backup restored): %w", err)
	}

	slog.Info("new binary verified successfully")
	utils.Success("Installed to %s", currentPath)

	// Try to clean up the backup (non-fatal if it fails).
	if err := os.Remove(backupPath); err != nil {
		slog.Debug("failed to remove backup", slog.String("path", backupPath), slog.String("error", err.Error()))
	}

	fmt.Println()
	utils.Success("Upgrade complete! Version: %s", newVersion)
	fmt.Println()
	fmt.Println("Verify the upgrade with: mdpress --version")

	return nil
}

// platformOS and platformArch can be overridden in tests to simulate different platforms.
var platformOS = runtime.GOOS
var platformArch = runtime.GOARCH

// findAssetForPlatform finds the appropriate release asset for the current OS/arch.
// Returns empty strings if no suitable asset is found.
func findAssetForPlatform(release *gitHubRelease) (url string, name string) {
	goos := strings.ToLower(platformOS)
	archAliases := platformArchAliases(platformArch)

	for _, archivesOnly := range []bool{false, true} {
		for _, asset := range release.Assets {
			assetName := strings.ToLower(asset.Name)
			if shouldSkipReleaseAsset(assetName) || isArchiveAsset(assetName) != archivesOnly {
				continue
			}
			if strings.Contains(assetName, goos) && containsAny(assetName, archAliases) {
				return asset.URL, asset.Name
			}
		}
	}

	for _, archivesOnly := range []bool{false, true} {
		for _, asset := range release.Assets {
			assetName := strings.ToLower(asset.Name)
			if shouldSkipReleaseAsset(assetName) || isArchiveAsset(assetName) != archivesOnly {
				continue
			}
			if strings.Contains(assetName, goos) {
				return asset.URL, asset.Name
			}
		}
	}

	return "", ""
}

func platformArchAliases(goarch string) []string {
	switch strings.ToLower(goarch) {
	case "amd64":
		return []string{"amd64", "x86_64"}
	case "arm64":
		return []string{"arm64", "aarch64"}
	default:
		return []string{strings.ToLower(goarch)}
	}
}

func shouldSkipReleaseAsset(assetName string) bool {
	return strings.Contains(assetName, ".sha256") ||
		strings.Contains(assetName, ".sig") ||
		strings.Contains(assetName, "checksum") ||
		strings.Contains(assetName, "source")
}

func isArchiveAsset(assetName string) bool {
	return strings.HasSuffix(assetName, ".tar.gz") || strings.HasSuffix(assetName, ".zip")
}

func containsAny(assetName string, candidates []string) bool {
	for _, candidate := range candidates {
		if strings.Contains(assetName, candidate) {
			return true
		}
	}
	return false
}

// isGitHubDownloadURL returns true if rawURL points to a known GitHub domain.
func isGitHubDownloadURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return u.Scheme == "https" &&
		(u.Host == "github.com" || strings.HasSuffix(u.Host, ".githubusercontent.com"))
}

func downloadBinary(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", mdpressUserAgent)

	resp, err := upgradeHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBinarySize+1))
	if err != nil {
		return nil, fmt.Errorf("read download response: %w", err)
	}
	if int64(len(data)) > maxBinarySize {
		return nil, fmt.Errorf("binary exceeds maximum size of %d bytes", maxBinarySize)
	}
	return data, nil
}

func extractBinaryData(assetName string, data []byte) ([]byte, error) {
	assetName = strings.ToLower(assetName)
	switch {
	case strings.HasSuffix(assetName, ".tar.gz"):
		return extractBinaryFromTarGz(data)
	case strings.HasSuffix(assetName, ".zip"):
		return extractBinaryFromZip(data)
	default:
		return data, nil
	}
}

func expectedBinaryNames() []string {
	if strings.EqualFold(platformOS, "windows") {
		return []string{"mdpress.exe", "mdpress"}
	}
	return []string{"mdpress"}
}

func extractBinaryFromTarGz(data []byte) ([]byte, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("invalid gzip archive: %w", err)
	}
	defer gzr.Close() //nolint:errcheck

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("invalid tar archive: %w", err)
		}
		if header.FileInfo().IsDir() {
			continue
		}
		// Defense-in-depth: skip entries with path traversal components.
		cleaned := filepath.Clean(header.Name)
		if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
			continue
		}
		if isExpectedBinaryEntry(header.Name) {
			data, err := io.ReadAll(io.LimitReader(tr, maxBinarySize+1))
			if err != nil {
				return nil, fmt.Errorf("failed to read tar entry: %w", err)
			}
			if int64(len(data)) > maxBinarySize {
				return nil, fmt.Errorf("binary in archive exceeds maximum size (%d bytes)", maxBinarySize)
			}
			return data, nil
		}
	}

	return nil, fmt.Errorf("archive does not contain %s", strings.Join(expectedBinaryNames(), " or "))
}

func extractBinaryFromZip(data []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid zip archive: %w", err)
	}

	for _, file := range zr.File {
		if file.FileInfo().IsDir() || !isExpectedBinaryEntry(file.Name) {
			continue
		}
		// Defense-in-depth: skip entries with path traversal components.
		cleaned := filepath.Clean(file.Name)
		if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open zip entry: %w", err)
		}
		content, readErr := io.ReadAll(io.LimitReader(reader, maxBinarySize+1))
		closeErr := reader.Close()
		if readErr != nil {
			return nil, fmt.Errorf("failed to read zip entry: %w", readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("failed to close zip entry: %w", closeErr)
		}
		if int64(len(content)) > maxBinarySize {
			return nil, fmt.Errorf("binary in archive exceeds maximum size (%d bytes)", maxBinarySize)
		}
		return content, nil
	}

	return nil, fmt.Errorf("archive does not contain %s", strings.Join(expectedBinaryNames(), " or "))
}

func isExpectedBinaryEntry(name string) bool {
	base := strings.ToLower(path.Base(name))
	for _, candidate := range expectedBinaryNames() {
		if base == strings.ToLower(candidate) {
			return true
		}
	}
	return false
}

// verifyChecksum verifies the downloaded binary against a checksum file in the release assets.
// It looks for a file named "checksums.txt" or similar in the release.
// Returns nil if verification passes, or a warning error if checksum file not found.
func verifyChecksum(ctx context.Context, release *gitHubRelease, assetName string, binaryData []byte) error {
	// Look for a checksum file in the release assets.
	var checksumURL string
	var checksumFileName string
	for _, asset := range release.Assets {
		assetNameLower := strings.ToLower(asset.Name)
		if strings.Contains(assetNameLower, "checksum") && (strings.HasSuffix(assetNameLower, ".txt") || strings.HasSuffix(assetNameLower, ".sha256")) {
			checksumURL = asset.URL
			checksumFileName = asset.Name
			break
		}
	}

	if checksumURL == "" {
		return errors.New("no checksum file found in release, skipping verification")
	}

	slog.Info("found checksum file", slog.String("file", checksumFileName))

	// Validate the checksum URL points to a known GitHub domain.
	if !isGitHubDownloadURL(checksumURL) {
		return errors.New("checksum URL has unexpected host (expected github.com or *.githubusercontent.com)")
	}

	// Download the checksum file.
	checksumData, err := downloadBinary(ctx, checksumURL)
	if err != nil {
		return fmt.Errorf("failed to download checksum file: %w", err)
	}

	// Parse the checksum file and look for a matching entry.
	checksumContent := string(checksumData)
	for _, line := range strings.Split(checksumContent, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse checksum line (format: "hash  filename" or "hash filename").
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		expectedHash := parts[0]
		checksumAssetName := parts[1]

		// Match by exact basename to avoid substring false positives
		// (e.g., "mdpress-linux-amd64" matching "mdpress-linux-amd64-musl").
		checksumBase := filepath.Base(checksumAssetName)
		assetBase := filepath.Base(assetName)
		if checksumBase == assetBase {
			// Compute SHA256 of the downloaded binary.
			actualHash := sha256.Sum256(binaryData)
			actualHashStr := hex.EncodeToString(actualHash[:])

			if strings.EqualFold(actualHashStr, expectedHash) {
				slog.Info("checksum verified", slog.String("hash", actualHashStr))
				utils.Success("Checksum verified")
				return nil
			}

			return fmt.Errorf("expected %s, got %s: %w", expectedHash, actualHashStr, errChecksumMismatch)
		}
	}

	return fmt.Errorf("%s: %w", assetName, errNoChecksumEntry)
}

// writeBinaryFile writes binary data to a file with executable permissions.
func writeBinaryFile(path string, data []byte) error {
	// Ensure parent directory exists.
	if err := utils.EnsureDir(filepath.Dir(path)); err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}

	// Write with executable permissions.
	if err := os.WriteFile(path, data, 0755); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
