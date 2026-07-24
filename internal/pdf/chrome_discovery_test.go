package pdf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// windowsEnv is a stand-in for the Windows environment so the Windows branch of
// platformChromiumInstallPaths can be exercised from any host OS.
func windowsEnv(name string) string {
	switch name {
	case "ProgramFiles":
		return `C:\Program Files`
	case "ProgramFiles(x86)":
		return `C:\Program Files (x86)`
	case "LOCALAPPDATA":
		return `C:\Users\amy\AppData\Local`
	}
	return ""
}

func emptyEnv(string) string { return "" }

// TestPlatformChromiumInstallPathsPerGOOS pins the well-known install locations
// per platform. Before this table existed the only filesystem fallback was the
// pair of macOS bundle paths, so a Windows box with Chrome installed exactly as
// the README instructs resolved nothing and every --format pdf build failed.
func TestPlatformChromiumInstallPathsPerGOOS(t *testing.T) {
	tests := []struct {
		name      string
		goos      string
		lookupEnv func(string) string
		want      []string
		absent    []string
	}{
		{
			name:      "windows finds Chrome and Edge under every install root",
			goos:      "windows",
			lookupEnv: windowsEnv,
			want: []string{
				filepath.Join(`C:\Program Files`, "Google", "Chrome", "Application", "chrome.exe"),
				filepath.Join(`C:\Program Files (x86)`, "Google", "Chrome", "Application", "chrome.exe"),
				filepath.Join(`C:\Users\amy\AppData\Local`, "Google", "Chrome", "Application", "chrome.exe"),
				filepath.Join(`C:\Program Files (x86)`, "Microsoft", "Edge", "Application", "msedge.exe"),
			},
			absent: []string{"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"},
		},
		{
			name:      "windows falls back to the default program-files roots",
			goos:      "windows",
			lookupEnv: emptyEnv,
			want: []string{
				filepath.Join(`C:\Program Files`, "Google", "Chrome", "Application", "chrome.exe"),
				filepath.Join(`C:\Program Files (x86)`, "Microsoft", "Edge", "Application", "msedge.exe"),
			},
		},
		{
			name:      "darwin keeps the application bundles",
			goos:      "darwin",
			lookupEnv: emptyEnv,
			want: []string{
				"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
				"/Applications/Chromium.app/Contents/MacOS/Chromium",
			},
			absent: []string{filepath.Join(`C:\Program Files`, "Google", "Chrome", "Application", "chrome.exe")},
		},
		{
			name:      "linux does not stat macOS bundle paths",
			goos:      "linux",
			lookupEnv: emptyEnv,
			want:      []string{"/opt/google/chrome/chrome"},
			absent: []string{
				"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
				filepath.Join(`C:\Program Files`, "Google", "Chrome", "Application", "chrome.exe"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := platformChromiumInstallPaths(tt.goos, tt.lookupEnv)
			for _, want := range tt.want {
				if !containsString(got, want) {
					t.Errorf("platformChromiumInstallPaths(%q) missing %q; got %v", tt.goos, want, got)
				}
			}
			for _, absent := range tt.absent {
				if containsString(got, absent) {
					t.Errorf("platformChromiumInstallPaths(%q) should not offer %q; got %v", tt.goos, absent, got)
				}
			}
		})
	}
}

// TestChromiumExecutableCandidatesIncludeWindowsNames guards the PATH lookup:
// the Windows binaries are called chrome.exe and msedge.exe, so a user who put
// Chrome's directory on PATH got nothing out of the Unix-only candidate list.
// The names are spelled without ".exe" because exec.LookPath applies PATHEXT.
func TestChromiumExecutableCandidatesIncludeWindowsNames(t *testing.T) {
	for _, want := range []string{"chrome", "msedge"} {
		if !containsString(chromiumExecutableCandidates, want) {
			t.Errorf("chromiumExecutableCandidates missing %q: %v", want, chromiumExecutableCandidates)
		}
	}
	for _, unwanted := range chromiumExecutableCandidates {
		if strings.HasSuffix(unwanted, ".exe") {
			t.Errorf("candidate %q must not carry .exe — exec.LookPath applies PATHEXT on Windows", unwanted)
		}
	}
}

// TestResolveChromiumPathFindsWindowsInstall walks the real resolver over a fake
// Program Files tree, proving the Windows locations are actually reached when
// nothing is on PATH — the exact situation on a stock Windows install.
func TestResolveChromiumPathFindsWindowsInstall(t *testing.T) {
	t.Setenv("MDPRESS_CHROME_PATH", "")
	t.Setenv("CHROME_BIN", "")
	t.Setenv("PATH", t.TempDir()) // nothing resolvable on PATH

	programFiles := filepath.Join(t.TempDir(), "Program Files")
	chromePath := filepath.Join(programFiles, "Google", "Chrome", "Application", "chrome.exe")
	if err := os.MkdirAll(filepath.Dir(chromePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chromePath, []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}

	original := chromiumInstallPaths
	chromiumInstallPaths = platformChromiumInstallPaths("windows", func(name string) string {
		if name == "ProgramFiles" {
			return programFiles
		}
		return ""
	})
	defer func() { chromiumInstallPaths = original }()

	got, err := resolveChromiumPath()
	if err != nil {
		t.Fatalf("resolveChromiumPath() returned error: %v", err)
	}
	if got != chromePath {
		t.Fatalf("resolveChromiumPath() = %q, want %q", got, chromePath)
	}
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
