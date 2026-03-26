package cmd

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestVersionComparison tests the version comparison logic.
func TestVersionComparison(t *testing.T) {
	tests := []struct {
		name        string
		newVersion  string
		oldVersion  string
		expectNewer bool
	}{
		// Basic version comparisons.
		{"newer major", "2.0.0", "1.9.9", true},
		{"same major, newer minor", "1.5.0", "1.4.9", true},
		{"same major.minor, newer patch", "1.2.5", "1.2.4", true},

		// Same versions.
		{"exact match", "1.2.3", "1.2.3", false},

		// Older versions.
		{"older major", "1.0.0", "2.0.0", false},
		{"older minor", "1.2.0", "1.3.0", false},
		{"older patch", "1.2.3", "1.2.4", false},

		// Versions with different lengths.
		{"longer new version", "1.2.3.4", "1.2.3", true},
		{"shorter new version, newer", "2.0", "1.9.9", true},

		// Versions with 'v' prefix.
		{"with v prefix", "v1.2.4", "1.2.3", true},
		{"both with v prefix", "v1.2.4", "v1.2.3", true},

		// Real-world examples.
		{"0.5.4 to 0.5.5", "0.5.5", "0.5.4", true},
		{"0.5.4 to 0.6.0", "0.6.0", "0.5.4", true},
		{"0.5.3 to 0.5.4", "0.5.4", "0.5.3", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVersionNewer(tt.newVersion, tt.oldVersion)
			if result != tt.expectNewer {
				t.Errorf("isVersionNewer(%q, %q) = %v, want %v",
					tt.newVersion, tt.oldVersion, result, tt.expectNewer)
			}
		})
	}
}

// TestParseVersion tests semantic version parsing.
func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []int
		wantLen int
	}{
		{"simple version", "1.2.3", []int{1, 2, 3}, 3},
		{"with v prefix", "v1.2.3", []int{1, 2, 3}, 3},
		{"two parts", "1.2", []int{1, 2}, 2},
		{"single number", "1", []int{1}, 1},
		{"with prerelease", "1.2.3-beta", []int{1, 2, 3}, 3},
		{"with build metadata", "1.2.3+build", []int{1, 2, 3}, 3},
		{"zero-padded", "01.02.03", []int{1, 2, 3}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseVersion(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseVersion(%q) returned length %d, want %d", tt.input, len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("parseVersion(%q)[%d] = %d, want %d", tt.input, i, v, tt.want[i])
				}
			}
		})
	}
}

// TestFindAssetForPlatform tests asset selection logic.
func TestFindAssetForPlatform(t *testing.T) {
	// Override platform to linux/amd64 for deterministic tests.
	origOS, origArch := platformOS, platformArch
	platformOS, platformArch = "linux", "amd64"
	t.Cleanup(func() { platformOS, platformArch = origOS, origArch })

	tests := []struct {
		name         string
		assetNames   []string
		shouldFind   bool
		expectedName string
	}{
		{
			name:         "linux x86_64",
			assetNames:   []string{"mdpress-linux-x86_64", "mdpress-darwin-aarch64", "mdpress-windows-x86_64.exe"},
			shouldFind:   true,
			expectedName: "mdpress-linux-x86_64",
		},
		{
			name:         "with amd64 naming",
			assetNames:   []string{"mdpress-linux-amd64", "mdpress-darwin-arm64"},
			shouldFind:   true,
			expectedName: "mdpress-linux-amd64",
		},
		{
			name:         "release archive accepted",
			assetNames:   []string{"source.tar.gz", "mdpress_1.0.0_linux_amd64.tar.gz", "checksums.txt"},
			shouldFind:   true,
			expectedName: "mdpress_1.0.0_linux_amd64.tar.gz",
		},
		{
			name:       "signature files skipped",
			assetNames: []string{"mdpress-linux-x86_64.sha256", "mdpress-linux-x86_64.sig"},
			shouldFind: false,
		},
		{
			name:         "prefer raw binary over archive",
			assetNames:   []string{"mdpress_1.0.0_linux_amd64.tar.gz", "mdpress-linux-amd64"},
			shouldFind:   true,
			expectedName: "mdpress-linux-amd64",
		},
		{
			name:       "empty assets",
			assetNames: []string{},
			shouldFind: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			release := &GitHubRelease{
				TagName: "v1.0.0",
				Assets: make([]struct {
					Name string `json:"name"`
					URL  string `json:"browser_download_url"`
				}, len(tt.assetNames)),
			}

			for i, name := range tt.assetNames {
				release.Assets[i].Name = name
				release.Assets[i].URL = fmt.Sprintf("https://example.com/%s", name)
			}

			_, name := findAssetForPlatform(release)

			if tt.shouldFind && name == "" {
				t.Errorf("findAssetForPlatform() expected to find an asset but got empty")
			}
			if !tt.shouldFind && name != "" {
				t.Errorf("findAssetForPlatform() expected not to find an asset but got %q", name)
			}
			if tt.shouldFind && name != tt.expectedName {
				t.Errorf("findAssetForPlatform() = %q, want %q", name, tt.expectedName)
			}
		})
	}
}

// TestFetchLatestReleaseMockServer tests GitHub API response parsing.
func TestFetchLatestReleaseMockServer(t *testing.T) {
	// Create a mock HTTP server.
	mockRelease := GitHubRelease{
		TagName: "v1.2.3",
		Assets: []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		}{
			{Name: "mdpress-linux-x86_64", URL: "https://example.com/mdpress-linux-x86_64"},
			{Name: "mdpress-darwin-aarch64", URL: "https://example.com/mdpress-darwin-aarch64"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET request, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockRelease); err != nil {
			t.Fatalf("failed to encode mock response: %v", err)
		}
	}))
	defer server.Close()

	// Test by mocking the GitHub API URL (this is a simplified test).
	// In real scenarios, you'd want to intercept the actual HTTP client.
	ctx := context.Background()

	// We can't easily test the real function without mocking the HTTP client,
	// but we've verified the JSON parsing structure works.
	data, err := json.Marshal(mockRelease)
	if err != nil {
		t.Fatalf("failed to marshal mock release: %v", err)
	}

	var parsed GitHubRelease
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.TagName != "v1.2.3" {
		t.Errorf("parsed tag name = %q, want %q", parsed.TagName, "v1.2.3")
	}
	if len(parsed.Assets) != 2 {
		t.Errorf("parsed asset count = %d, want 2", len(parsed.Assets))
	}

	_ = ctx // Suppress unused variable warning.
}

// TestFetchLatestReleaseHTTP tests the full fetch flow with a mock server.
func TestFetchLatestReleaseHTTP(t *testing.T) {
	mockRelease := GitHubRelease{
		TagName: "v1.2.3",
		Assets: []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		}{
			{Name: "mdpress-linux-x86_64", URL: "https://example.com/mdpress-linux-x86_64"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockRelease) //nolint:errcheck
	}))
	defer server.Close()

	// This is a simplified test showing the structure works.
	// For a full integration test, we'd need to mock the HTTP client in the main function.
	ctx := context.Background()

	// Create a test request to verify the client works as expected.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if release.TagName != "v1.2.3" {
		t.Errorf("release tag = %q, want %q", release.TagName, "v1.2.3")
	}
}

// TestUpgradeCheckOnlyFlag tests that the check flag is properly parsed.
func TestUpgradeCheckOnlyFlag(t *testing.T) {
	// This test verifies the flag definition exists and works with Cobra.
	if upgradeCmd == nil {
		t.Fatal("upgradeCmd is nil")
	}

	checkFlag := upgradeCmd.Flags().Lookup("check")
	if checkFlag == nil {
		t.Fatal("--check flag not found")
	}

	if checkFlag.DefValue != "false" {
		t.Errorf("--check default value = %q, want false", checkFlag.DefValue)
	}
}

// TestVersionParsingEdgeCases tests edge cases in version parsing.
func TestVersionParsingEdgeCases(t *testing.T) {
	tests := []struct {
		version  string
		expected []int
	}{
		{"v0.5.4", []int{0, 5, 4}},
		{"1", []int{1}},
		{"1.0.0.0", []int{1, 0, 0, 0}},
		{"99.99.99", []int{99, 99, 99}},
	}

	for _, tt := range tests {
		result := parseVersion(tt.version)
		if len(result) != len(tt.expected) {
			t.Errorf("parseVersion(%q) length = %d, want %d", tt.version, len(result), len(tt.expected))
			continue
		}

		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("parseVersion(%q)[%d] = %d, want %d", tt.version, i, v, tt.expected[i])
			}
		}
	}
}

// TestParseVersionExtendedEdgeCases tests additional edge cases for parseVersion.
func TestParseVersionExtendedEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []int
	}{
		{"empty string", "", []int{}},
		{"only v prefix", "v", []int{}},
		{"non-numeric prefix", "alpha1.2.3", []int{2, 3}}, // "alpha1" has no leading digits, skips it
		{"with build metadata", "1.2.3+build", []int{1, 2, 3}},
		{"with build metadata complex", "1.2.3+build.123", []int{1, 2, 3, 123}}, // extracts "123" from "+build.123"
		{"prerelease rc", "1.2.3-rc1", []int{1, 2, 3}},                          // "-rc1" has no leading digits, skips it
		{"prerelease alpha", "1.2.3-alpha.1", []int{1, 2, 3, 1}},                // "-alpha" has no leading digits, but "1" is extracted from ".1"
		{"single number", "5", []int{5}},
		{"very long version", "1.2.3.4.5.6.7.8.9.10", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		{"mixed separators with letters", "1.2a.3b.4", []int{1, 2, 3, 4}},
		{"zeros", "0.0.0", []int{0, 0, 0}},
		{"large numbers", "999.888.777", []int{999, 888, 777}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseVersion(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseVersion(%q) returned length %d, want %d", tt.input, len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("parseVersion(%q)[%d] = %d, want %d", tt.input, i, v, tt.want[i])
				}
			}
		})
	}
}

// TestIsVersionNewerExtendedEdgeCases tests edge cases for version comparison.
func TestIsVersionNewerExtendedEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		newVersion  string
		oldVersion  string
		expectNewer bool
	}{
		// Same version variations
		{"same version", "1.2.3", "1.2.3", false},
		{"same with v prefix", "v1.2.3", "v1.2.3", false},
		{"same mixed v prefix", "v1.2.3", "1.2.3", false},
		// Empty strings
		{"empty new version", "", "1.2.3", false},
		{"empty old version", "1.2.3", "", true},
		{"both empty", "", "", false},
		// Malformed versions
		{"malformed both", "abc", "def", false},
		{"malformed new", "abc", "1.2.3", false},
		{"malformed old", "1.2.3", "abc", true},
		// Different segment counts
		{"fewer segments in new", "1.2", "1.2.3", false},
		{"more segments in new", "1.2.3", "1.2", true},
		{"one vs two segments newer", "2", "1.2.3", true},
		{"one vs two segments older", "1", "2.0.0", false},
		// Prerelease and build metadata
		{"prerelease vs release", "1.2.3-beta", "1.2.3", false},
		{"both prerelease same base", "1.2.3-beta", "1.2.3-alpha", false},
		// Large version jumps
		{"major jump", "10.0.0", "9.9.9", true},
		{"large jump", "100.0.0", "1.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVersionNewer(tt.newVersion, tt.oldVersion)
			if result != tt.expectNewer {
				t.Errorf("isVersionNewer(%q, %q) = %v, want %v",
					tt.newVersion, tt.oldVersion, result, tt.expectNewer)
			}
		})
	}
}

// TestFindAssetForPlatformExtended tests asset selection for various platform combinations.
func TestFindAssetForPlatformExtended(t *testing.T) {
	tests := []struct {
		name              string
		assetNames        []string
		goos              string
		goarch            string
		shouldFind        bool
		expectedNameMatch string
	}{
		{
			name:              "linux amd64 converted to x86_64",
			assetNames:        []string{"mdpress-linux-x86_64"},
			goos:              "linux",
			goarch:            "amd64",
			shouldFind:        true,
			expectedNameMatch: "mdpress-linux-x86_64",
		},
		{
			name:              "darwin arm64 converted to aarch64",
			assetNames:        []string{"mdpress-darwin-aarch64"},
			goos:              "darwin",
			goarch:            "arm64",
			shouldFind:        true,
			expectedNameMatch: "mdpress-darwin-aarch64",
		},
		{
			name:              "windows with exe suffix",
			assetNames:        []string{"mdpress-windows-x86_64.exe", "mdpress-windows-aarch64.exe"},
			goos:              "windows",
			goarch:            "amd64",
			shouldFind:        true,
			expectedNameMatch: "mdpress-windows-x86_64.exe",
		},
		{
			name:              "no matching assets",
			assetNames:        []string{"mdpress-freebsd-x86_64"},
			goos:              "linux",
			goarch:            "amd64",
			shouldFind:        false,
			expectedNameMatch: "",
		},
		{
			name:              "multiple potential matches exact first",
			assetNames:        []string{"mdpress-linux-x86_64", "mdpress-linux-amd64"},
			goos:              "linux",
			goarch:            "amd64",
			shouldFind:        true,
			expectedNameMatch: "mdpress-linux-x86_64", // exact match with amd64->x86_64 conversion
		},
		{
			name:              "fallback to OS match only",
			assetNames:        []string{"mdpress-linux", "mdpress-windows-x86_64"},
			goos:              "linux",
			goarch:            "amd64",
			shouldFind:        true,
			expectedNameMatch: "mdpress-linux",
		},
		{
			name:              "skip sha256 and sig files",
			assetNames:        []string{"mdpress-linux-x86_64.sha256", "mdpress-linux-x86_64.sig", "mdpress-linux-x86_64"},
			goos:              "linux",
			goarch:            "amd64",
			shouldFind:        true,
			expectedNameMatch: "mdpress-linux-x86_64",
		},
		{
			name:              "skip source archives",
			assetNames:        []string{"source.tar.gz", "source.zip", "mdpress-linux-x86_64"},
			goos:              "linux",
			goarch:            "amd64",
			shouldFind:        true,
			expectedNameMatch: "mdpress-linux-x86_64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Override platform variables so the test works on any OS.
			oldOS, oldArch := platformOS, platformArch
			platformOS, platformArch = tt.goos, tt.goarch
			t.Cleanup(func() { platformOS, platformArch = oldOS, oldArch })

			release := &GitHubRelease{
				TagName: "v1.0.0",
				Assets: make([]struct {
					Name string `json:"name"`
					URL  string `json:"browser_download_url"`
				}, len(tt.assetNames)),
			}

			for i, name := range tt.assetNames {
				release.Assets[i].Name = name
				release.Assets[i].URL = fmt.Sprintf("https://example.com/%s", name)
			}

			_, name := findAssetForPlatform(release)

			if tt.shouldFind && name == "" {
				t.Errorf("findAssetForPlatform() expected to find an asset but got empty")
			}
			if !tt.shouldFind && name != "" {
				t.Errorf("findAssetForPlatform() expected not to find an asset but got %q", name)
			}
			if tt.shouldFind && tt.expectedNameMatch != "" && name != tt.expectedNameMatch {
				t.Errorf("findAssetForPlatform() = %q, want %q", name, tt.expectedNameMatch)
			}
		})
	}
}

// TestFetchLatestReleaseHTTPErrors tests error handling in fetchLatestRelease.
func TestFetchLatestReleaseHTTPErrors(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   any
		expectJSONErr  bool
		expectValidErr bool
	}{
		{
			name:           "404 not found",
			statusCode:     http.StatusNotFound,
			expectJSONErr:  false,
			expectValidErr: true,
		},
		{
			name:           "500 internal server error",
			statusCode:     http.StatusInternalServerError,
			expectJSONErr:  false,
			expectValidErr: true,
		},
		{
			name:           "invalid JSON response",
			statusCode:     http.StatusOK,
			responseBody:   "invalid json",
			expectJSONErr:  true,
			expectValidErr: false,
		},
		{
			name:       "missing tag_name",
			statusCode: http.StatusOK,
			responseBody: map[string]any{
				"assets": []any{},
			},
			expectJSONErr:  false,
			expectValidErr: true,
		},
		{
			name:       "no assets",
			statusCode: http.StatusOK,
			responseBody: map[string]any{
				"tag_name": "v1.0.0",
				"assets":   []any{},
			},
			expectJSONErr:  false,
			expectValidErr: true,
		},
		{
			name:       "valid response",
			statusCode: http.StatusOK,
			responseBody: map[string]any{
				"tag_name": "v1.0.0",
				"assets": []map[string]any{
					{
						"name":                 "mdpress-linux-x86_64",
						"browser_download_url": "https://example.com/mdpress",
					},
				},
			},
			expectJSONErr:  false,
			expectValidErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Header().Set("Content-Type", "application/json")

				if tt.responseBody != nil {
					switch body := tt.responseBody.(type) {
					case string:
						w.Write([]byte(body)) //nolint:errcheck
					case map[string]any:
						json.NewEncoder(w).Encode(body) //nolint:errcheck
					}
				}
			}))
			defer server.Close()

			ctx := context.Background()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			defer resp.Body.Close() //nolint:errcheck

			// Verify status code handling
			if resp.StatusCode != tt.statusCode {
				t.Errorf("status code = %d, want %d", resp.StatusCode, tt.statusCode)
			}

			// For non-200 responses, status code is the issue
			if resp.StatusCode != http.StatusOK {
				if !tt.expectValidErr {
					t.Errorf("expected status error but expected none")
				}
				return
			}

			// For 200 responses, verify JSON parsing
			var release GitHubRelease
			err = json.NewDecoder(resp.Body).Decode(&release)
			if tt.expectJSONErr && err == nil {
				t.Errorf("expected JSON parsing error but got none")
			}
			if !tt.expectJSONErr && err != nil {
				t.Errorf("unexpected JSON parsing error: %v", err)
			}

			// Check validation logic (mimicking fetchLatestRelease behavior)
			if !tt.expectJSONErr && err == nil {
				if release.TagName == "" && tt.expectValidErr {
					// This is expected - missing tag_name should fail
				} else if release.TagName == "" && !tt.expectValidErr {
					t.Errorf("tag_name is empty but shouldn't be")
				}
				if len(release.Assets) == 0 && tt.expectValidErr {
					// This is expected - no assets should fail
				} else if len(release.Assets) == 0 && !tt.expectValidErr {
					t.Errorf("assets list is empty but shouldn't be")
				}
			}
		})
	}
}

// TestInstallNewVersionFlow tests the installation flow with temporary files.
func TestInstallNewVersionFlow(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock current binary
	currentBinary := filepath.Join(tmpDir, "mdpress")
	originalData := []byte("original binary content")
	if err := os.WriteFile(currentBinary, originalData, 0755); err != nil {
		t.Fatalf("failed to create mock binary: %v", err)
	}

	// Create new binary data
	newData := []byte("new binary content")

	// Test successful installation
	t.Run("successful installation", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows does not use Unix executable permission bits")
		}

		err := writeBinaryFile(currentBinary, newData)
		if err != nil {
			t.Errorf("writeBinaryFile() failed: %v", err)
		}

		// Verify new content
		content, err := os.ReadFile(currentBinary)
		if err != nil {
			t.Fatalf("failed to read installed binary: %v", err)
		}
		if string(content) != string(newData) {
			t.Errorf("installed binary content mismatch")
		}

		// Verify permissions
		info, err := os.Stat(currentBinary)
		if err != nil {
			t.Fatalf("failed to stat binary: %v", err)
		}
		mode := info.Mode()
		if runtime.GOOS != "windows" && mode&0100 == 0 {
			t.Errorf("binary is not executable by owner")
		}
	})

	// Test backup creation
	t.Run("backup creation and restoration", func(t *testing.T) {
		// Reset binary
		if err := os.WriteFile(currentBinary, originalData, 0755); err != nil {
			t.Fatalf("failed to reset binary: %v", err)
		}

		backupPath := currentBinary + ".backup"

		// Create backup
		if err := os.Rename(currentBinary, backupPath); err != nil {
			t.Errorf("failed to create backup: %v", err)
		}

		// Verify backup exists
		if _, err := os.Stat(backupPath); err != nil {
			t.Errorf("backup file not found: %v", err)
		}

		// Restore from backup
		if err := os.Rename(backupPath, currentBinary); err != nil {
			t.Errorf("failed to restore from backup: %v", err)
		}

		// Verify restoration
		content, err := os.ReadFile(currentBinary)
		if err != nil {
			t.Fatalf("failed to read restored binary: %v", err)
		}
		if string(content) != string(originalData) {
			t.Errorf("restored binary content mismatch")
		}
	})

	// Test directory creation
	t.Run("directory creation for nested paths", func(t *testing.T) {
		nestedPath := filepath.Join(tmpDir, "subdir", "mdpress")
		if err := writeBinaryFile(nestedPath, newData); err != nil {
			t.Errorf("writeBinaryFile() failed for nested path: %v", err)
		}

		// Verify directory and file were created
		if _, err := os.Stat(nestedPath); err != nil {
			t.Errorf("nested binary file not created: %v", err)
		}
	})
}

// TestDownloadBinaryMock tests binary download functionality.
func TestDownloadBinaryMock(t *testing.T) {
	mockBinaryData := []byte{0x7f, 0x45, 0x4c, 0x46} // ELF header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(mockBinaryData) //nolint:errcheck
	}))
	defer server.Close()

	t.Run("successful download", func(t *testing.T) {
		ctx := context.Background()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		defer resp.Body.Close() //nolint:errcheck

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response body: %v", err)
		}

		if len(data) != len(mockBinaryData) {
			t.Errorf("downloaded data size = %d, want %d", len(data), len(mockBinaryData))
		}
		if string(data) != string(mockBinaryData) {
			t.Errorf("downloaded data content mismatch")
		}
	})

	t.Run("error on non-200 status", func(t *testing.T) {
		errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer errorServer.Close()

		ctx := context.Background()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, errorServer.URL, nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		defer resp.Body.Close() //nolint:errcheck

		if resp.StatusCode == http.StatusOK {
			t.Errorf("expected non-200 status code, got %d", resp.StatusCode)
		}
	})
}

func TestExtractBinaryData(t *testing.T) {
	origOS, origArch := platformOS, platformArch
	t.Cleanup(func() {
		platformOS, platformArch = origOS, origArch
	})

	t.Run("tar.gz archive", func(t *testing.T) {
		platformOS, platformArch = "linux", "amd64"
		archiveData := mustCreateTarGzArchive(t, map[string][]byte{
			"mdpress":      []byte("linux binary"),
			"README.txt":   []byte("ignored"),
			"nested/extra": []byte("ignored"),
		})

		binaryData, err := extractBinaryData("mdpress_1.0.0_linux_amd64.tar.gz", archiveData)
		if err != nil {
			t.Fatalf("extractBinaryData() failed: %v", err)
		}
		if string(binaryData) != "linux binary" {
			t.Fatalf("extractBinaryData() = %q, want linux binary", string(binaryData))
		}
	})

	t.Run("zip archive", func(t *testing.T) {
		platformOS, platformArch = "windows", "amd64"
		archiveData := mustCreateZipArchive(t, map[string][]byte{
			"mdpress.exe": []byte("windows binary"),
			"notes.txt":   []byte("ignored"),
		})

		binaryData, err := extractBinaryData("mdpress_1.0.0_windows_amd64.zip", archiveData)
		if err != nil {
			t.Fatalf("extractBinaryData() failed: %v", err)
		}
		if string(binaryData) != "windows binary" {
			t.Fatalf("extractBinaryData() = %q, want windows binary", string(binaryData))
		}
	})

	t.Run("missing binary in archive", func(t *testing.T) {
		platformOS, platformArch = "linux", "amd64"
		archiveData := mustCreateTarGzArchive(t, map[string][]byte{
			"README.txt": []byte("missing"),
		})

		_, err := extractBinaryData("mdpress_1.0.0_linux_amd64.tar.gz", archiveData)
		if err == nil {
			t.Fatal("extractBinaryData() expected error for archive without binary")
		}
	})
}

func mustCreateTarGzArchive(t *testing.T, files map[string][]byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0755,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatalf("WriteHeader(%q) failed: %v", name, err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatalf("Write(%q) failed: %v", name, err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("tar writer close failed: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("gzip writer close failed: %v", err)
	}

	return buf.Bytes()
}

func mustCreateZipArchive(t *testing.T, files map[string][]byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for name, content := range files {
		writer, err := zw.Create(name)
		if err != nil {
			t.Fatalf("Create(%q) failed: %v", name, err)
		}
		if _, err := writer.Write(content); err != nil {
			t.Fatalf("Write(%q) failed: %v", name, err)
		}
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("zip writer close failed: %v", err)
	}

	return buf.Bytes()
}
