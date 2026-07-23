package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/pdf"
	"github.com/yeasy/mdpress/internal/plugin"
	"github.com/yeasy/mdpress/pkg/utils"
)

const (
	// maxPlantUMLScanSize caps how much of each Markdown file is read when scanning for PlantUML blocks.
	maxPlantUMLScanSize = 1024 * 1024
	// networkCheckTimeout is the timeout for the network connectivity check.
	networkCheckTimeout = 5 * time.Second
	// dfCommandTimeout is the timeout for the local df command used to check disk space.
	dfCommandTimeout = 5 * time.Second
	// typstVersionTimeout is the timeout for running `typst --version` in the doctor check.
	typstVersionTimeout = 5 * time.Second
)

var (
	doctorReportPath string
	// doctorStrict makes `mdpress doctor` exit non-zero when any error-level
	// finding is recorded, so it can be used as a CI gate. Default false keeps
	// the historical always-exit-0 behavior for existing scripts.
	doctorStrict bool
)

var doctorCmd = &cobra.Command{
	Use:   "doctor [directory]",
	Short: "Report environment info and check PDF/project readiness",
	Long: `Report mdpress runtime environment details and run a small set of readiness checks, including:
  - Runtime platform information
  - Go runtime version (informational)
  - Chrome/Chromium availability for PDF output
  - Typst availability for --format typst PDF output
  - Go version (>=1.26 recommended)
  - Git availability for remote source builds
  - Network connectivity to github.com
  - Disk space in output directory
  - CJK font availability for Asian text rendering
  - Plugin health and availability
  - Presence of book.yaml / SUMMARY.md / LANGS.md
  - Whether the project can be loaded
  - Whether Markdown chapter links stay inside the build graph

Pass --strict to exit with a non-zero status when any error-level check fails
(for example a missing PDF backend or an unloadable book.yaml), which makes
mdpress doctor usable as a CI gate. Without --strict the command always exits 0.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir := "."
		if len(args) > 0 {
			targetDir = args[0]
		}
		return executeDoctor(cmd.Context(), targetDir)
	},
}

func init() {
	// Shorthand is -r, not -o: every other command spells output with -o, and
	// `mdpress doctor -o ./out` silently meant "write the report there" —
	// then failed at the very end because ./out has no .json/.md extension.
	doctorCmd.Flags().StringVarP(&doctorReportPath, "report", "r", "", "Write doctor report to .json or .md")
	doctorCmd.Flags().BoolVar(&doctorStrict, "strict", false, "Exit with a non-zero status when any error-level check fails (useful as a CI gate)")
}

type doctorReport struct {
	Platform          string `json:"platform"`
	GoVersion         string `json:"go_version"`
	CacheDir          string `json:"cache_dir,omitempty"`
	CacheDisabled     bool   `json:"cache_disabled"`
	ChromiumAvailable bool   `json:"chromium_available"`
	TypstAvailable    bool   `json:"typst_available"`
	TypstVersion      string `json:"typst_version,omitempty"`
	CJKFontsAvailable bool   `json:"cjk_fonts_available"`
	// PlantUMLNeeded reports that the project contains plantuml blocks,
	// which mdpress publishes as plain code rather than rendering.
	PlantUMLNeeded     bool                     `json:"plantuml_needed"`
	GoVersionCheck     string                   `json:"go_version_check,omitempty"`
	GitAvailable       bool                     `json:"git_available"`
	NetworkAvailable   bool                     `json:"network_available"`
	DiskSpaceGB        float64                  `json:"disk_space_gb,omitempty"`
	DiskSpaceOK        bool                     `json:"disk_space_ok"`
	PluginsValid       bool                     `json:"plugins_valid"`
	PluginCount        int                      `json:"plugin_count,omitempty"`
	BookYAMLFound      bool                     `json:"book_yaml_found"`
	SummaryFound       bool                     `json:"summary_found"`
	LangsFound         bool                     `json:"langs_found"`
	ProjectLoadable    bool                     `json:"project_loadable"`
	ProjectTitle       string                   `json:"project_title,omitempty"`
	TopLevelChapters   int                      `json:"top_level_chapters,omitempty"`
	Warnings           []string                 `json:"warnings,omitempty"`
	UnresolvedMarkdown []unresolvedMarkdownLink `json:"unresolved_markdown_links,omitempty"`
	// doctorErrors counts error-level findings (Chromium+Typst both missing,
	// book.yaml load failure, low disk, broken plugins, etc). Not serialized;
	// used only to decide the --strict exit code.
	doctorErrors int
}

// addDoctorError records an error-level finding and its human-readable message
// as a warning entry, so `--strict` can gate the exit code on it.
func (r *doctorReport) addDoctorError(msg string) {
	r.doctorErrors++
	r.Warnings = append(r.Warnings, msg)
}

func executeDoctor(ctx context.Context, targetDir string) error {
	// Reject an unusable --report path before spending a minute on checks whose
	// output is about to be thrown away.
	if doctorReportPath != "" && !doctorReportPathSupported(doctorReportPath) {
		return errUnsupportedDoctorReportExt(doctorReportPath)
	}

	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory path: %w", err)
	}
	info, err := os.Stat(absDir)
	if err != nil {
		return fmt.Errorf("target directory is not accessible: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("target path is not a directory: %s", absDir)
	}

	report := doctorReport{
		Platform:      fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		GoVersion:     runtime.Version(),
		CacheDir:      utils.CacheRootDir(),
		CacheDisabled: utils.CacheDisabled(),
	}

	utils.Header("mdpress Environment Check")
	fmt.Println()
	utils.Success("Platform: %s", report.Platform)
	utils.Success("Go version: %s", report.GoVersion)
	if report.CacheDisabled {
		utils.Warning("Runtime cache is disabled")
	} else {
		utils.Success("Runtime cache: %s", report.CacheDir)
	}

	// Check the two PDF backends. PDF output needs either Chromium/Chrome OR
	// Typst, so a missing backend is only an error when BOTH are unavailable.
	chromiumErr := pdf.CheckChromiumAvailable()
	if chromiumErr == nil {
		report.ChromiumAvailable = true
		utils.Success("Chromium/Chrome is available")
	}

	checkTypst(&report)

	if chromiumErr != nil {
		if report.TypstAvailable {
			utils.Warning("Chromium/Chrome is unavailable — use --format typst for PDF output instead")
			fmt.Printf("    %s\n", chromiumErr.Error())
			report.Warnings = append(report.Warnings, "Chromium/Chrome is unavailable (use --format typst for PDF output)")
		} else {
			utils.Error("No PDF backend available: Chromium/Chrome and Typst are both missing (PDF output will fail)")
			fmt.Printf("    %s\n", chromiumErr.Error())
			fmt.Println("    Install Chrome/Chromium, or install Typst (brew install typst) and build with --format typst")
			report.addDoctorError("No PDF backend available: Chromium/Chrome and Typst are both missing (PDF output will fail)")
		}
	}

	// Check Go version
	checkGoVersion(&report)

	// Check Git availability
	checkGitAvailable(&report)

	// Check Network connectivity
	checkNetworkConnectivity(ctx, &report)

	// Try to load project config once for reuse by multiple checks.
	// --config is resolved the same way build and validate resolve it. doctor
	// used to look only for <dir>/book.yaml, so a project whose config is named
	// anything else was reported as unbuildable however healthy it was.
	bookPath, configDiscoverable := resolveConfigPath(absDir)
	var doctorCfg *config.BookConfig
	if _, err := os.Stat(bookPath); err == nil {
		if cfg, loadErr := config.Load(bookPath); loadErr == nil {
			doctorCfg = cfg
		}
	}

	// Check disk space in output directory
	checkDiskSpace(ctx, absDir, doctorCfg, &report)

	// Check CJK font availability for PDF rendering of Chinese/Japanese/Korean content.
	cjkStatus := utils.CheckCJKFonts()
	report.CJKFontsAvailable = cjkStatus.Available
	if cjkStatus.Available {
		utils.Success("CJK fonts available: %s", strings.Join(cjkStatus.Fonts, ", "))
	} else {
		utils.Warning("No CJK fonts detected — PDF output for Chinese/Japanese/Korean text may show blank squares")
		fmt.Printf("    %s\n", utils.CJKFontInstallHint())
		report.Warnings = append(report.Warnings, "No CJK fonts detected (PDF output may show blank squares for CJK text)")
	}

	// Check PlantUML availability for diagram rendering.
	checkPlantUML(absDir, &report)

	// Check plugins availability
	checkPlugins(absDir, doctorCfg, &report)

	fmt.Println()
	utils.Header("Project Check")

	summaryPath := filepath.Join(absDir, "SUMMARY.md")
	langsPath := filepath.Join(absDir, "LANGS.md")

	configLabel := filepath.Base(bookPath)
	if _, err := os.Stat(bookPath); err == nil {
		report.BookYAMLFound = true
		utils.Success("Detected %s", configLabel)
	} else if !configDiscoverable {
		// An explicit --config that does not exist is a mistake, not a project
		// without a config: discovering some other book instead would hide it.
		utils.Error("Config file not found: %s (--config was given explicitly)", bookPath)
		report.addDoctorError(fmt.Sprintf("Config file not found: %s (--config was given explicitly)", bookPath))
	} else {
		utils.Warning("book.yaml not found")
	}

	// SUMMARY.md and LANGS.md are optional. Reporting their absence as warnings
	// meant a freshly scaffolded project — the one shape mdpress creates
	// itself — greeted the user with two warnings about nothing being wrong.
	if _, err := os.Stat(summaryPath); err == nil {
		report.SummaryFound = true
		utils.Success("Detected SUMMARY.md")
	} else if report.BookYAMLFound && doctorCfg != nil && len(doctorCfg.Chapters) > 0 {
		utils.Info("SUMMARY.md not found (chapters come from %s)", configLabel)
	} else {
		utils.Warning("SUMMARY.md not found")
	}
	if _, err := os.Stat(langsPath); err == nil {
		report.LangsFound = true
		utils.Success("Detected LANGS.md")
		utils.Warning("Multi-language projects are built per language directory; root-level mixed-book output is not generated")
		report.Warnings = append(report.Warnings, "Multi-language projects are built per language directory; root-level mixed-book output is not generated")
	} else if hasLanguageSubdirs(absDir) {
		// Only meaningful when the layout actually looks multi-language.
		utils.Warning("LANGS.md not found, but language-like subdirectories exist — add LANGS.md to build them per language")
		report.Warnings = append(report.Warnings, "LANGS.md not found, but language-like subdirectories exist")
	}

	if doctorCfg != nil {
		report.ProjectLoadable = true
		report.ProjectTitle = doctorCfg.Book.Title
		report.TopLevelChapters = len(doctorCfg.Chapters)
		utils.Success("Config loads successfully: %s (%d top-level chapters)", doctorCfg.Book.Title, len(doctorCfg.Chapters))
		reportDoctorMarkdownLinks(doctorCfg, &report)
	} else if _, err := os.Stat(bookPath); err == nil {
		// The config exists but failed to load — report the error.
		if _, loadErr := config.Load(bookPath); loadErr != nil {
			utils.Error("Failed to load %s: %v", configLabel, loadErr)
			report.addDoctorError(fmt.Sprintf("Failed to load %s: %v", configLabel, loadErr))
		}
	} else if !configDiscoverable {
		// The explicit --config is missing; it was already reported above.
		// Falling back to discovery here would describe a different project
		// than the one the user asked about, which is what build refuses to do.
	} else {
		// Zero-config discovery is a first-class way to build, so ask it rather
		// than pattern-matching on SUMMARY.md: a directory of plain Markdown
		// files was declared unbuildable while `mdpress build` built it fine.
		cfg, discoverErr := config.Discover(ctx, absDir)
		switch {
		case discoverErr != nil && report.SummaryFound:
			utils.Error("Failed to auto-discover project from SUMMARY.md: %v", discoverErr)
			report.addDoctorError(fmt.Sprintf("Failed to auto-discover project from SUMMARY.md: %v", discoverErr))
		case discoverErr != nil:
			utils.Warning("No buildable project found: no config, no SUMMARY.md, and no Markdown files to auto-discover")
			report.Warnings = append(report.Warnings, "No buildable project found: no config, no SUMMARY.md, and no Markdown files to auto-discover")
		default:
			report.ProjectLoadable = true
			report.ProjectTitle = cfg.Book.Title
			report.TopLevelChapters = len(cfg.Chapters)
			utils.Success("Project can be loaded by auto-discovery: %s (%d top-level chapters)", cfg.Book.Title, len(cfg.Chapters))
			if !report.BookYAMLFound {
				utils.Info("Run 'mdpress init' to write a book.yaml and take control of the chapter order")
			}
			reportDoctorMarkdownLinks(cfg, &report)
		}
	}

	if doctorReportPath != "" {
		if err := writeDoctorReport(doctorReportPath, report); err != nil {
			return fmt.Errorf("failed to write doctor report: %w", err)
		}
	}

	fmt.Println()

	// In strict mode, surface error-level findings as a non-zero exit code so
	// `mdpress doctor --strict` can act as a CI gate. Default mode always
	// returns nil (unless the directory itself was inaccessible above).
	if doctorStrict && report.doctorErrors > 0 {
		return fmt.Errorf("doctor found %d error-level issue(s) (run without --strict to ignore)", report.doctorErrors)
	}

	return nil
}

// langDirPattern matches directory names that look like BCP-47 language tags
// ("en", "zh", "zh-CN", "pt_BR"), which is how mdpress lays out multi-language
// books.
var langDirPattern = regexp.MustCompile(`^[a-z]{2,3}([-_][A-Za-z0-9]{2,4})?$`)

// hasLanguageSubdirs reports whether dir contains at least one language-tag
// subdirectory holding Markdown, i.e. whether a missing LANGS.md is worth
// mentioning at all.
func hasLanguageSubdirs(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() || !langDirPattern.MatchString(entry.Name()) {
			continue
		}
		sub, err := os.ReadDir(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		for _, f := range sub {
			if !f.IsDir() && strings.EqualFold(filepath.Ext(f.Name()), ".md") {
				return true
			}
		}
	}
	return false
}

func reportDoctorMarkdownLinks(cfg *config.BookConfig, report *doctorReport) {
	unresolved, err := findUnresolvedMarkdownLinks(cfg)
	if err != nil {
		utils.Error("Markdown chapter link analysis failed: %v", err)
		if report != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("Markdown chapter link analysis failed: %v", err))
		}
		return
	}
	if len(unresolved) == 0 {
		utils.Success("Markdown chapter links resolve within the build graph")
		return
	}

	if report != nil {
		report.UnresolvedMarkdown = unresolved
	}
	utils.Warning("Detected %d Markdown link(s) outside the build graph", len(unresolved))
	limit := len(unresolved)
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		fmt.Printf("    - %s (from %s)\n", unresolved[i].Target, unresolved[i].Source)
	}
	if len(unresolved) > limit {
		fmt.Printf("    - ... and %d more\n", len(unresolved)-limit)
	}
}

func checkGoVersion(report *doctorReport) {
	version := runtime.Version()
	report.GoVersionCheck = version

	// Extract version number (e.g., "go1.26.0" -> "1.26.0")
	versionStr := strings.TrimPrefix(version, "go")
	parts := strings.Split(versionStr, ".")
	if len(parts) < 2 {
		utils.Warning("Could not parse Go version")
		return
	}

	// Check if major.minor >= 1.26
	major, majorErr := utils.ParseVersionPart(parts[0])
	if majorErr != nil {
		if verbose {
			utils.Warning("Could not parse Go major version: %v", majorErr)
		}
		return
	}

	minor, minorErr := utils.ParseVersionPart(parts[1])
	if minorErr != nil {
		if verbose {
			utils.Warning("Could not parse Go minor version: %v", minorErr)
		}
		return
	}

	if major > 1 || (major == 1 && minor >= 26) {
		utils.Success("Go version %s (>= 1.26)", versionStr)
	} else {
		utils.Warning("Go version %s (< 1.26) — some features may not work as expected", versionStr)
		report.Warnings = append(report.Warnings, fmt.Sprintf("Go version %s is below recommended 1.26", versionStr))
	}
}

// checkTypst reports whether the `typst` CLI is available for `mdpress build
// --format typst`. It is intentionally self-contained (no internal/typst
// import) to avoid coupling the doctor command to the Typst backend package.
func checkTypst(report *doctorReport) {
	path, err := exec.LookPath("typst")
	if err != nil {
		utils.Warning("Typst not found (required for --format typst PDF output)")
		report.Warnings = append(report.Warnings, "Typst is not available (required for --format typst PDF output)")
		return
	}

	report.TypstAvailable = true

	// Try to capture the version string; treat failure as non-fatal.
	ctx, cancel := context.WithTimeout(context.Background(), typstVersionTimeout)
	defer cancel()
	out, verErr := exec.CommandContext(ctx, path, "--version").Output()
	if verErr == nil {
		report.TypstVersion = strings.TrimSpace(string(out))
	}

	if report.TypstVersion != "" {
		utils.Success("Typst is available: %s", report.TypstVersion)
	} else {
		utils.Success("Typst is available")
	}
}

func checkGitAvailable(report *doctorReport) {
	_, err := exec.LookPath("git")
	if err != nil {
		utils.Warning("Git not found (required for remote source builds)")
		report.Warnings = append(report.Warnings, "Git is not available (required for remote source builds)")
	} else {
		report.GitAvailable = true
		utils.Success("Git is available")
	}
}

func checkNetworkConnectivity(parentCtx context.Context, report *doctorReport) {
	// Try an HTTP HEAD request to github.com (cross-platform connectivity check).
	// Derive from the parent context so the doctor command's cancellation is respected.
	ctx, cancel := context.WithTimeout(parentCtx, networkCheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, "https://github.com", nil)
	if err != nil {
		utils.Warning("Network connectivity check failed")
		if verbose {
			fmt.Printf("    Error: %v\n", err)
		}
		report.Warnings = append(report.Warnings, "Cannot reach github.com (network connectivity issue)")
		return
	}

	client := &http.Client{
		Timeout:   networkCheckTimeout,
		Transport: utils.SSRFSafeTransport(),
	}

	resp, err := client.Do(req)
	if err != nil {
		utils.Warning("Network connectivity check failed (cannot reach github.com)")
		if verbose {
			fmt.Printf("    Error: %v\n", err)
		}
		report.Warnings = append(report.Warnings, "Cannot reach github.com (network connectivity issue)")
		return
	}
	defer resp.Body.Close() //nolint:errcheck

	report.NetworkAvailable = true
	utils.Success("Network connectivity to github.com available")
}

func checkDiskSpace(ctx context.Context, targetDir string, cfg *config.BookConfig, report *doctorReport) {
	outputDir := filepath.Join(targetDir, "_book")

	// Use loaded config to get the output directory if available.
	if cfg != nil && cfg.Output.Filename != "" {
		outPath := filepath.Dir(cfg.Output.Filename)
		if !filepath.IsAbs(outPath) {
			outPath = filepath.Join(targetDir, outPath)
		}
		if absPath, err := filepath.Abs(outPath); err == nil {
			outputDir = absPath
		}
	}

	// Check output directory without creating it (doctor is read-only)
	if _, err := os.Stat(outputDir); err != nil {
		// Fall back to parent directory for disk space check
		outputDir = filepath.Dir(outputDir)
		if _, err := os.Stat(outputDir); err != nil {
			if verbose {
				utils.Warning("Could not access output directory: %v", err)
			}
			return
		}
	}

	// Get disk space using the df command (works on macOS, Linux, and most Unix systems).
	// On Windows this will fail gracefully and we skip the check.
	dfCtx, dfCancel := context.WithTimeout(ctx, dfCommandTimeout)
	defer dfCancel()
	// Sanitize: if outputDir starts with "-", prefix with "./" to avoid flag injection.
	// Note: "--" is not portable to macOS/BSD df.
	dfDir := outputDir
	if strings.HasPrefix(dfDir, "-") {
		dfDir = "./" + dfDir
	}
	out, err := exec.CommandContext(dfCtx, "df", "-k", dfDir).Output()
	if err != nil {
		if verbose {
			utils.Warning("Could not determine disk space: %v", err)
		}
		return
	}
	// Parse df output: second line, 4th column (available KB).
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		return
	}
	var availKB int64
	if _, err := fmt.Sscanf(fields[3], "%d", &availKB); err != nil {
		return
	}
	const kbPerGB = 1024 * 1024
	availableGB := float64(availKB) / kbPerGB
	report.DiskSpaceGB = availableGB

	// Threshold: 100 MB = 0.1 GB
	const minDiskSpaceGB = 0.1
	if availableGB < minDiskSpaceGB {
		utils.Error("Disk space critically low (%.2f GB available, < 100 MB required)", availableGB)
		report.addDoctorError(fmt.Sprintf("Disk space critically low: only %.2f GB available", availableGB))
		return
	}

	report.DiskSpaceOK = true
	if verbose {
		utils.Success("Disk space available: %.2f GB", availableGB)
	} else {
		utils.Success("Disk space available")
	}
}

func checkPlugins(targetDir string, cfg *config.BookConfig, report *doctorReport) {
	if cfg == nil {
		report.PluginsValid = true
		return
	}

	if len(cfg.Plugins) == 0 {
		// No plugins configured
		report.PluginsValid = true
		return
	}

	report.PluginCount = len(cfg.Plugins)
	allValid := true

	for _, pluginCfg := range cfg.Plugins {
		if pluginCfg.Path == "" {
			allValid = false
			if verbose {
				utils.Error("Plugin %q has empty path", pluginCfg.Name)
			} else {
				utils.Warning("Plugin %q has empty path", pluginCfg.Name)
			}
			continue
		}

		pluginPath := pluginCfg.Path
		if !filepath.IsAbs(pluginPath) {
			pluginPath = filepath.Join(targetDir, pluginPath)
		}

		// Check if the executable exists
		info, err := os.Stat(pluginPath)
		if err != nil {
			allValid = false
			if verbose {
				utils.Error("Plugin %q not found at %s", pluginCfg.Name, pluginPath)
			} else {
				utils.Warning("Plugin %q not found", pluginCfg.Name)
			}
			continue
		}

		// Check if it's executable
		if !isPluginExecutable(pluginPath, info.Mode()) {
			allValid = false
			utils.Warning("Plugin %q is not executable: %s", pluginCfg.Name, pluginPath)
			report.Warnings = append(report.Warnings, fmt.Sprintf("Plugin %q is not executable", pluginCfg.Name))
			continue
		}

		// Existing + executable was the whole test, so doctor happily reported
		// "all plugins are valid" for programs that cannot answer a single
		// mdpress query — and the very next build warned about each of them.
		// Ask the plugin itself.
		probe, probeErr := plugin.Probe(pluginPath)
		if probeErr != nil {
			allValid = false
			utils.Warning("Plugin %q could not be probed: %v", pluginCfg.Name, probeErr)
			report.Warnings = append(report.Warnings, fmt.Sprintf("Plugin %q could not be probed: %v", pluginCfg.Name, probeErr))
			continue
		}
		if !probe.SpeaksProtocol {
			allValid = false
			utils.Warning("Plugin %q does not speak the mdpress plugin protocol (no valid --mdpress-info or --mdpress-hooks response)", pluginCfg.Name)
			if verbose {
				fmt.Printf("    --mdpress-info: %v\n", probe.InfoErr)
				fmt.Printf("    --mdpress-hooks: %v\n", probe.HooksErr)
			}
			report.Warnings = append(report.Warnings,
				fmt.Sprintf("Plugin %q does not speak the mdpress plugin protocol", pluginCfg.Name))
			continue
		}
		if verbose {
			utils.Success("Plugin %q responds (version %s, %d hook(s))", pluginCfg.Name, probe.Version, len(probe.Hooks))
		}
	}

	if allValid {
		report.PluginsValid = true
		utils.Success("All %d plugin(s) are valid", len(cfg.Plugins))
	} else {
		report.PluginsValid = false
		report.addDoctorError(fmt.Sprintf("%d plugin(s) have issues", len(cfg.Plugins)))
	}
}

func isExecutable(mode os.FileMode) bool {
	return (mode & 0o111) != 0
}

func isPluginExecutable(path string, mode os.FileMode) bool {
	if runtime.GOOS == "windows" {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".exe", ".bat", ".cmd", ".com", ".ps1":
			return true
		default:
			return false
		}
	}
	return isExecutable(mode)
}

// checkPlantUML reports that PlantUML blocks are published as plain code.
//
// mdpress ships a PlantUML renderer that no production path constructs — the
// package is not linked into the binary — so a ```plantuml fence renders as a
// code block. This check used to probe for Java, PLANTUML_JAR and the plantuml
// command and tell the user to `brew install plantuml`, which meant they could
// install the toolchain, write diagrams, and only discover from the finished
// artifact that nothing had been drawn.
func checkPlantUML(targetDir string, report *doctorReport) {
	if !hasPlantUMLBlocks(targetDir) {
		return
	}
	report.PlantUMLNeeded = true
	utils.Warning("PlantUML blocks found — mdpress publishes them as plain code, not diagrams")
	report.Warnings = append(report.Warnings,
		"PlantUML blocks are published as plain code; mdpress does not render PlantUML")
	fmt.Println("    Pre-render the diagrams and reference the images instead, or use ```mermaid,")
	fmt.Println("    which mdpress does render.")
}

// hasPlantUMLBlocks checks if any markdown files in the directory contain plantuml code blocks.
func hasPlantUMLBlocks(targetDir string) bool {
	return searchPlantUMLInDir(targetDir)
}

// scanFileForPlantUML reads a single file and checks for ```plantuml blocks.
// Reads at most maxPlantUMLScanSize bytes to prevent OOM on very large files.
func scanFileForPlantUML(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	content, err := func() ([]byte, error) {
		defer f.Close() //nolint:errcheck
		return io.ReadAll(io.LimitReader(f, maxPlantUMLScanSize))
	}()
	if err != nil {
		return false
	}
	return strings.Contains(string(content), "```plantuml")
}

// searchPlantUMLInDir recursively searches for ```plantuml blocks in markdown files.
// maxDepth prevents infinite recursion from symlink loops.
func searchPlantUMLInDir(dir string) bool {
	return searchPlantUMLInDirDepth(dir, 0)
}

const maxPlantUMLSearchDepth = 20

func searchPlantUMLInDirDepth(dir string, depth int) bool {
	if depth > maxPlantUMLSearchDepth {
		return false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		// Skip hidden directories and common excluded directories
		if strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "_") {
			continue
		}
		if entry.Name() == "node_modules" || entry.Name() == "vendor" {
			continue
		}

		path := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			if searchPlantUMLInDirDepth(path, depth+1) {
				return true
			}
		} else if entry.Type()&os.ModeSymlink != 0 {
			// Resolve symlinks: follow directories, scan .md files.
			info, err := os.Stat(path)
			if err != nil {
				continue
			}
			if info.IsDir() {
				if searchPlantUMLInDirDepth(path, depth+1) {
					return true
				}
			} else if strings.HasSuffix(entry.Name(), ".md") && scanFileForPlantUML(path) {
				return true
			}
		} else if strings.HasSuffix(entry.Name(), ".md") && scanFileForPlantUML(path) {
			return true
		}
	}

	return false
}

func writeDoctorReport(path string, report doctorReport) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve report path: %w", err)
	}
	if err := utils.EnsureDir(filepath.Dir(absPath)); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}

	switch strings.ToLower(filepath.Ext(absPath)) {
	case ".json":
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal report: %w", err)
		}
		return os.WriteFile(absPath, data, 0o644)
	case ".md":
		return os.WriteFile(absPath, []byte(renderDoctorMarkdown(report)), 0o644)
	default:
		return errUnsupportedDoctorReportExt(path)
	}
}

// errUnsupportedDoctorReportExt names the path the user typed. Reporting the
// bare extension produced "unsupported report extension: " for an extensionless
// path, which said nothing about what was wrong or what to type instead.
func errUnsupportedDoctorReportExt(path string) error {
	ext := filepath.Ext(path)
	if ext == "" {
		return fmt.Errorf("report path %q has no file extension (use a path ending in .json or .md)", path)
	}
	return fmt.Errorf("unsupported report extension %q in %q (use .json or .md)", ext, path)
}

// doctorReportPathSupported reports whether writeDoctorReport can handle path.
func doctorReportPathSupported(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json", ".md":
		return true
	default:
		return false
	}
}

func renderDoctorMarkdown(report doctorReport) string {
	var b strings.Builder
	b.WriteString("# mdpress Doctor Report\n\n")
	fmt.Fprintf(&b, "- Platform: %s\n", report.Platform)
	fmt.Fprintf(&b, "- Go version: %s\n", report.GoVersion)
	if report.GoVersionCheck != "" {
		fmt.Fprintf(&b, "- Go version check: %s\n", report.GoVersionCheck)
	}
	fmt.Fprintf(&b, "- Cache disabled: %t\n", report.CacheDisabled)
	if report.CacheDir != "" {
		fmt.Fprintf(&b, "- Cache dir: %s\n", report.CacheDir)
	}
	fmt.Fprintf(&b, "- Chromium available: %t\n", report.ChromiumAvailable)
	fmt.Fprintf(&b, "- Typst available: %t\n", report.TypstAvailable)
	if report.TypstVersion != "" {
		fmt.Fprintf(&b, "- Typst version: %s\n", report.TypstVersion)
	}
	fmt.Fprintf(&b, "- CJK fonts available: %t\n", report.CJKFontsAvailable)
	fmt.Fprintf(&b, "- Git available: %t\n", report.GitAvailable)
	fmt.Fprintf(&b, "- Network connectivity: %t\n", report.NetworkAvailable)
	if report.DiskSpaceGB > 0 {
		fmt.Fprintf(&b, "- Disk space available: %.2f GB\n", report.DiskSpaceGB)
	}
	fmt.Fprintf(&b, "- Disk space OK: %t\n", report.DiskSpaceOK)
	fmt.Fprintf(&b, "- PlantUML blocks present (published as plain code): %t\n", report.PlantUMLNeeded)
	fmt.Fprintf(&b, "- Plugins valid: %t\n", report.PluginsValid)
	if report.PluginCount > 0 {
		fmt.Fprintf(&b, "- Plugin count: %d\n", report.PluginCount)
	}
	fmt.Fprintf(&b, "- book.yaml found: %t\n", report.BookYAMLFound)
	fmt.Fprintf(&b, "- SUMMARY.md found: %t\n", report.SummaryFound)
	fmt.Fprintf(&b, "- LANGS.md found: %t\n", report.LangsFound)
	fmt.Fprintf(&b, "- Project loadable: %t\n", report.ProjectLoadable)
	if report.ProjectTitle != "" {
		fmt.Fprintf(&b, "- Project title: %s\n", report.ProjectTitle)
	}
	if report.TopLevelChapters > 0 {
		fmt.Fprintf(&b, "- Top-level chapters: %d\n", report.TopLevelChapters)
	}
	if len(report.Warnings) > 0 {
		b.WriteString("\n## Warnings\n\n")
		for _, warning := range report.Warnings {
			fmt.Fprintf(&b, "- %s\n", warning)
		}
	}
	if len(report.UnresolvedMarkdown) > 0 {
		b.WriteString("\n## Unresolved Markdown Links\n\n")
		for _, item := range report.UnresolvedMarkdown {
			fmt.Fprintf(&b, "- %s (from %s)\n", item.Target, item.Source)
		}
	}
	return b.String()
}
