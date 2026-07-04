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
	"runtime/debug"
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
	Timeout:   upgradeClientTimeout,
	Transport: utils.SSRFSafeTransport(),
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
	// binaryVerifyTimeout is the timeout for verifying the new binary after upgrade.
	binaryVerifyTimeout = 10 * time.Second
)

var upgradeCheckOnly bool

// upgradeForce bypasses the package-manager install-method guard and forces the
// binary-replacement path even for Homebrew/go-install managed installs.
var upgradeForce bool

// upgradeSkipChecksum bypasses checksum verification. Verification fails hard by
// default (including when the checksum asset is missing or its download fails);
// this flag exists only as an explicit escape hatch.
var upgradeSkipChecksum bool

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
  - Detects package-manager installs (Homebrew, go install) and defers to them
  - Downloads the appropriate binary for your OS/arch
  - Verifies the download against the published checksums
  - Replaces the current mdpress binary
  - Shows progress during the download`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeUpgrade(cmd.Context(), upgradeCheckOnly)
	},
}

func init() {
	upgradeCmd.Flags().BoolVar(&upgradeCheckOnly, "check", false, "Only check for updates, do not install")
	upgradeCmd.Flags().BoolVar(&upgradeForce, "force", false, "Force binary replacement even for Homebrew/go-install managed installs")
	upgradeCmd.Flags().BoolVar(&upgradeSkipChecksum, "skip-checksum", false, "Skip checksum verification of the downloaded binary (not recommended)")
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
	// Classify how mdpress was installed. Overwriting a Homebrew keg or a
	// go-install binary desyncs from the package manager, so defer to it unless
	// the user explicitly forces a binary replacement with --force.
	if !upgradeForce {
		if method, advice := upgradeDetectInstallMethod(); method != upgradeInstallBinary {
			fmt.Println()
			utils.Warning("%s", advice)
			fmt.Println()
			fmt.Println("Use 'mdpress upgrade --force' to overwrite the binary in place anyway (not recommended).")
			// Abort cleanly: this is not an error, the user just needs a different command.
			return nil
		}
	}

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

	// Verify the download against the published checksums. The release always
	// ships a checksums.txt (see .goreleaser.yml), so any verification error --
	// mismatch, missing entry, missing checksum asset, or a failed checksum
	// download -- is treated as a hard failure to avoid installing an unverified
	// binary. The only bypass is the explicit --skip-checksum flag.
	if upgradeSkipChecksum {
		slog.Warn("skipping checksum verification (--skip-checksum)")
		utils.Warning("Skipping checksum verification (--skip-checksum)")
	} else if err := verifyChecksum(ctx, release, assetName, assetData); err != nil {
		return fmt.Errorf("checksum verification failed: %w", err)
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
	verifyCtx, verifyCancel := context.WithTimeout(ctx, binaryVerifyTimeout)
	defer verifyCancel()
	if out, err := exec.CommandContext(verifyCtx, currentPath, "version").CombinedOutput(); err != nil {
		slog.Error("new binary verification failed", slog.String("output", string(out)), slog.Any("error", err))
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
		slog.Debug("failed to remove backup", slog.String("path", backupPath), slog.Any("error", err))
	}

	fmt.Println()
	utils.Success("Upgrade complete! Version: %s", newVersion)
	fmt.Println()
	fmt.Println("Verify the upgrade with: mdpress --version")

	return nil
}

// upgradeInstallMethod identifies how the running mdpress binary was installed,
// which determines whether the self-updater may replace the binary in place.
type upgradeInstallMethod int

const (
	// upgradeInstallBinary means mdpress was installed as a standalone binary
	// (curl/download, manual copy); the self-updater may replace it in place.
	upgradeInstallBinary upgradeInstallMethod = iota
	// upgradeInstallBrew means mdpress lives inside a Homebrew keg; brew owns it.
	upgradeInstallBrew
	// upgradeInstallGoInstall means mdpress was placed by `go install`; the Go
	// tooling and module cache own it.
	upgradeInstallGoInstall
)

// upgradeReadBuildInfo is a hook allowing tests to inject build info. It returns
// the main module version, main module path, and whether build info was present.
var upgradeReadBuildInfo = func() (version string, mainPath string, ok bool) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "", "", false
	}
	return bi.Main.Version, bi.Main.Path, true
}

// upgradeDetectInstallMethod classifies the current executable's install method
// and returns human-readable advice when it is package-manager managed.
func upgradeDetectInstallMethod() (method upgradeInstallMethod, advice string) {
	exePath, err := os.Executable()
	if err != nil {
		// Cannot determine the executable path; fall through to binary replacement.
		return upgradeInstallBinary, ""
	}

	// Resolve symlinks so Homebrew's bin/ shims (which point into the Cellar)
	// are classified by their real location.
	resolvedPath := exePath
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		resolvedPath = resolved
	}

	goBinDirs := upgradeGoBinDirs()
	version, mainPath, hasBuildInfo := upgradeReadBuildInfo()

	return upgradeClassifyInstallPath(exePath, resolvedPath, upgradeHomebrewPrefixes(), goBinDirs, version, mainPath, hasBuildInfo)
}

// upgradeClassifyInstallPath is the path-based core of install-method detection,
// split out so it can be unit-tested without touching the real filesystem.
//   - rawPath/resolvedPath: the executable path before/after symlink resolution.
//   - brewPrefixes: Homebrew prefixes to check (e.g. /opt/homebrew, /usr/local).
//   - goBinDirs: directories that `go install` writes to (GOBIN, GOPATH/bin).
//   - biVersion/biMainPath/hasBuildInfo: runtime/debug build info about the binary.
func upgradeClassifyInstallPath(
	rawPath, resolvedPath string,
	brewPrefixes, goBinDirs []string,
	biVersion, biMainPath string,
	hasBuildInfo bool,
) (upgradeInstallMethod, string) {
	// Homebrew: the raw or resolved path is inside a Caskroom/Cellar or under a
	// Homebrew prefix.
	for _, p := range []string{rawPath, resolvedPath} {
		if upgradeIsBrewManaged(p, brewPrefixes) {
			return upgradeInstallBrew,
				"mdpress was installed via Homebrew -- run: brew update && brew upgrade --cask mdpress"
		}
	}

	// go install: build info reports a real module version/path AND the binary
	// lives under a Go bin directory (GOBIN or GOPATH/bin).
	if hasBuildInfo && biMainPath != "" && biVersion != "" && biVersion != "(devel)" {
		for _, p := range []string{rawPath, resolvedPath} {
			if upgradePathUnderAny(p, goBinDirs) {
				return upgradeInstallGoInstall,
					"mdpress was installed via go install -- run: go install github.com/yeasy/mdpress@latest"
			}
		}
	}

	return upgradeInstallBinary, ""
}

// upgradeIsBrewManaged reports whether path is Homebrew-managed: it contains a
// Caskroom/Cellar component or lives under one of the given Homebrew prefixes.
func upgradeIsBrewManaged(path string, brewPrefixes []string) bool {
	if path == "" {
		return false
	}
	normalized := filepath.ToSlash(path)
	if strings.Contains(normalized, "/Caskroom/") || strings.Contains(normalized, "/Cellar/") {
		return true
	}
	for _, prefix := range brewPrefixes {
		if prefix == "" {
			continue
		}
		if upgradePathUnderAny(path, []string{filepath.Join(prefix, "Caskroom"), filepath.Join(prefix, "Cellar")}) {
			return true
		}
	}
	return false
}

// upgradeHomebrewPrefixes returns candidate Homebrew prefixes, honoring
// HOMEBREW_PREFIX and falling back to the common defaults.
func upgradeHomebrewPrefixes() []string {
	if prefix := strings.TrimSpace(os.Getenv("HOMEBREW_PREFIX")); prefix != "" {
		return []string{prefix}
	}
	return []string{"/opt/homebrew", "/usr/local"}
}

// upgradeGoBinDirs returns the directories `go install` may write binaries to:
// $GOBIN if set, otherwise $GOPATH/bin (or the default GOPATH's bin).
func upgradeGoBinDirs() []string {
	if gobin := strings.TrimSpace(os.Getenv("GOBIN")); gobin != "" {
		return []string{gobin}
	}
	var dirs []string
	if gopath := strings.TrimSpace(os.Getenv("GOPATH")); gopath != "" {
		for _, p := range filepath.SplitList(gopath) {
			if p != "" {
				dirs = append(dirs, filepath.Join(p, "bin"))
			}
		}
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		dirs = append(dirs, filepath.Join(home, "go", "bin"))
	}
	return dirs
}

// upgradePathUnderAny reports whether path is inside one of dirs (or equal to it),
// using cleaned, separator-boundary-aware comparison to avoid prefix false positives.
func upgradePathUnderAny(path string, dirs []string) bool {
	if path == "" {
		return false
	}
	cleanPath := filepath.Clean(path)
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		cleanDir := filepath.Clean(dir)
		if cleanPath == cleanDir {
			return true
		}
		if strings.HasPrefix(cleanPath, cleanDir+string(filepath.Separator)) {
			return true
		}
	}
	return false
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
		if errors.Is(err, io.EOF) {
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
// Returns nil only if verification passes. Any failure -- a missing checksum
// asset, a failed checksum download, a missing entry, or a hash mismatch -- is
// returned as a non-nil error so callers never install an unverified binary.
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
	if err := os.WriteFile(path, data, 0o755); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Explicitly set permissions: os.WriteFile only applies the mode when
	// creating a new file. If the file was pre-created (e.g. by os.CreateTemp
	// with 0o600), the execute bits would be missing.
	if err := os.Chmod(path, 0o755); err != nil { //nolint:gosec // G302: binary must be executable
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	return nil
}
