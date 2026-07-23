package cmd

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// captureDoctorRun runs `doctor` against dir and returns everything it printed
// (utils.Error goes to stderr, the rest to stdout) with the command's error.
func captureDoctorRun(t *testing.T, dir string) (string, error) {
	t.Helper()

	origStdout, origStderr := os.Stdout, os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w

	done := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(r)
		done <- string(data)
	}()

	runErr := executeDoctor(context.Background(), dir)
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	os.Stdout, os.Stderr = origStdout, origStderr
	return <-done, runErr
}

// TestDoctorHonorsConfigFlag covers finding 3: --config was accepted and then
// ignored, so a healthy project whose config is not named book.yaml was
// reported as unloadable.
func TestDoctorHonorsConfigFlag(t *testing.T) {
	restoreGlobalFlags(t)

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "ch1.md"), "# One\n\nBody.\n")
	writeTestFile(t, filepath.Join(dir, "release.yaml"), `book:
  title: "Renamed Config Book"
chapters:
  - title: "One"
    file: "ch1.md"
`)

	cfgFile = "release.yaml"
	doctorReportPath = ""
	doctorStrict = false
	t.Chdir(dir)

	out, err := captureDoctorRun(t, dir)
	if err != nil {
		t.Fatalf("doctor returned error: %v", err)
	}
	if !strings.Contains(out, "Renamed Config Book") {
		t.Errorf("doctor should load the project named by --config, got:\n%s", out)
	}
	if strings.Contains(out, "No buildable project found") {
		t.Errorf("doctor should not call a project with an explicit --config unbuildable, got:\n%s", out)
	}
}

// TestDoctorReportsMissingExplicitConfig makes sure the --config path is not
// simply substituted by discovery when it does not exist.
func TestDoctorReportsMissingExplicitConfig(t *testing.T) {
	restoreGlobalFlags(t)

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "ch1.md"), "# One\n\nBody.\n")

	cfgFile = "nope.yaml"
	doctorReportPath = ""
	doctorStrict = true
	t.Chdir(dir)

	out, err := captureDoctorRun(t, dir)
	if err == nil {
		t.Fatal("--strict should fail when the explicit --config does not exist")
	}
	if !strings.Contains(out, "nope.yaml") {
		t.Errorf("doctor should name the missing config, got:\n%s", out)
	}
}

// TestDoctorReportsPlainMarkdownDirAsBuildable covers finding 39: a directory
// of plain Markdown was declared unbuildable while `mdpress build` built it.
func TestDoctorReportsPlainMarkdownDirAsBuildable(t *testing.T) {
	restoreGlobalFlags(t)

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "a.md"), "# A\n\nBody.\n")

	cfgFile = defaultConfigName
	doctorReportPath = ""
	doctorStrict = false
	t.Chdir(dir)

	out, err := captureDoctorRun(t, dir)
	if err != nil {
		t.Fatalf("doctor returned error: %v", err)
	}
	if !strings.Contains(out, "auto-discovery") {
		t.Errorf("doctor should report a plain Markdown directory as buildable, got:\n%s", out)
	}
	if strings.Contains(out, "No directly buildable") {
		t.Errorf("doctor still reports a buildable directory as unbuildable, got:\n%s", out)
	}
}

// TestDoctorIsQuietOnScaffoldedProject covers finding 1025: a freshly created
// project immediately collected warnings about SUMMARY.md and LANGS.md, which
// the scaffold deliberately does not create.
func TestDoctorIsQuietOnScaffoldedProject(t *testing.T) {
	restoreGlobalFlags(t)

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "README.md"), "# Intro\n\nBody.\n")
	writeTestFile(t, filepath.Join(dir, "book.yaml"), `book:
  title: "Scaffolded"
chapters:
  - title: "Intro"
    file: "README.md"
`)

	cfgFile = defaultConfigName
	doctorReportPath = ""
	doctorStrict = false
	t.Chdir(dir)

	out, _ := captureDoctorRun(t, dir)
	projectSection := out
	if idx := strings.Index(out, "Project Check"); idx >= 0 {
		projectSection = out[idx:]
	}
	for _, unwanted := range []string{"SUMMARY.md not found", "LANGS.md not found"} {
		for _, line := range strings.Split(projectSection, "\n") {
			if strings.Contains(line, unwanted) && strings.Contains(line, "⚠") {
				t.Errorf("a scaffolded project should not warn about optional files, got: %q", strings.TrimSpace(line))
			}
		}
	}
}

// TestDoctorWarnsAboutMissingLangsOnlyForMultiLanguageLayout keeps the LANGS.md
// hint for the layout where it is actually actionable.
func TestDoctorWarnsAboutMissingLangsOnlyForMultiLanguageLayout(t *testing.T) {
	dir := t.TempDir()
	if hasLanguageSubdirs(dir) {
		t.Fatal("an empty directory has no language subdirectories")
	}
	if err := os.MkdirAll(filepath.Join(dir, "en"), 0o755); err != nil {
		t.Fatalf("create en dir: %v", err)
	}
	if hasLanguageSubdirs(dir) {
		t.Error("a language-named directory without Markdown is not a multi-language layout")
	}
	writeTestFile(t, filepath.Join(dir, "en", "a.md"), "# A\n")
	if !hasLanguageSubdirs(dir) {
		t.Error("en/ holding Markdown should count as a multi-language layout")
	}
	if err := os.MkdirAll(filepath.Join(dir, "images"), 0o755); err != nil {
		t.Fatalf("create images dir: %v", err)
	}
	writeTestFile(t, filepath.Join(dir, "images", "a.md"), "# A\n")
	other := t.TempDir()
	if err := os.MkdirAll(filepath.Join(other, "images"), 0o755); err != nil {
		t.Fatalf("create images dir: %v", err)
	}
	writeTestFile(t, filepath.Join(other, "images", "a.md"), "# A\n")
	if hasLanguageSubdirs(other) {
		t.Error("images/ is not a language tag")
	}
}

// TestDoctorProbesPluginProtocol covers finding 1018: an executable that
// cannot answer a single mdpress query was reported as a valid plugin, and the
// next build warned about it.
func TestDoctorProbesPluginProtocol(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fixture plugin is a POSIX shell script")
	}
	restoreGlobalFlags(t)

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "a.md"), "# A\n\nBody.\n")
	notAPlugin := filepath.Join(dir, "hello.sh")
	writeTestFile(t, notAPlugin, "#!/bin/sh\necho hello\n")
	if err := os.Chmod(notAPlugin, 0o755); err != nil {
		t.Fatalf("chmod fixture plugin: %v", err)
	}
	writeTestFile(t, filepath.Join(dir, "book.yaml"), `book:
  title: "Plugin Book"
chapters:
  - title: "A"
    file: "a.md"
plugins:
  - name: hello
    path: "./hello.sh"
`)

	cfgFile = defaultConfigName
	doctorReportPath = ""
	doctorStrict = true
	t.Chdir(dir)

	out, err := captureDoctorRun(t, dir)
	if err == nil {
		t.Fatal("--strict should fail when a configured plugin does not speak the protocol")
	}
	if strings.Contains(out, "plugin(s) are valid") {
		t.Errorf("an executable that answers no mdpress query is not a valid plugin, got:\n%s", out)
	}
	if !strings.Contains(out, "plugin protocol") {
		t.Errorf("doctor should say what is wrong with the plugin, got:\n%s", out)
	}
}

// TestDoctorAcceptsProtocolSpeakingPlugin is the other half of the check: a
// plugin that answers the handshake must still pass.
func TestDoctorAcceptsProtocolSpeakingPlugin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fixture plugin is a POSIX shell script")
	}
	restoreGlobalFlags(t)

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "a.md"), "# A\n\nBody.\n")
	good := filepath.Join(dir, "good.sh")
	writeTestFile(t, good, "#!/bin/sh\ncase \"$1\" in\n  --mdpress-info) echo '{\"version\":\"1.2.3\"}' ;;\n  --mdpress-hooks) echo '[\"after_parse\"]' ;;\nesac\n")
	if err := os.Chmod(good, 0o755); err != nil {
		t.Fatalf("chmod fixture plugin: %v", err)
	}
	writeTestFile(t, filepath.Join(dir, "book.yaml"), `book:
  title: "Plugin Book"
chapters:
  - title: "A"
    file: "a.md"
plugins:
  - name: good
    path: "./good.sh"
`)

	cfgFile = defaultConfigName
	doctorReportPath = ""
	doctorStrict = true
	t.Chdir(dir)

	out, err := captureDoctorRun(t, dir)
	if err != nil && strings.Contains(out, "plugin") && strings.Contains(out, "protocol") {
		t.Fatalf("a protocol-speaking plugin should pass the plugin check, got:\n%s", out)
	}
	if !strings.Contains(out, "plugin(s) are valid") {
		t.Errorf("doctor should accept a plugin that answers the handshake, got:\n%s", out)
	}
}

// TestDoctorReportShorthandIsNotOutput covers finding 5: -o meant "report" on
// doctor and "output" everywhere else, and `doctor -o ./out` ran every check
// before failing on the extension.
func TestDoctorReportShorthandIsNotOutput(t *testing.T) {
	if flag := doctorCmd.Flags().Lookup("report"); flag == nil {
		t.Error("doctor should have a --report flag")
	} else if flag.Shorthand != "r" {
		t.Errorf("doctor --report shorthand should be -r, got %q", flag.Shorthand)
	}
	if doctorCmd.Flags().ShorthandLookup("o") != nil {
		t.Error("doctor should not define -o: it means --output on every other command")
	}
}

// TestDoctorRejectsUnusableReportPathUpFront covers the second half of finding
// 5: the extension was only checked after every check had run, and the message
// ("unsupported report extension: ") named nothing.
func TestDoctorRejectsUnusableReportPathUpFront(t *testing.T) {
	restoreGlobalFlags(t)
	defer suppressOutput(t)()

	dir := t.TempDir()
	doctorReportPath = filepath.Join(dir, "out")
	doctorStrict = false

	err := executeDoctor(context.Background(), dir)
	if err == nil {
		t.Fatal("an extensionless --report path should be rejected")
	}
	if !strings.Contains(err.Error(), "out") || !strings.Contains(err.Error(), ".json") {
		t.Errorf("error should name the path and the supported extensions, got: %v", err)
	}
}
