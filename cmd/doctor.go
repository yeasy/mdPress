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
	"runtime"
	"strings"

	"time"

	"github.com/spf13/cobra"
	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/pdf"
	"github.com/yeasy/mdpress/pkg/utils"
)

// maxPlantUMLScanSize caps how much of each Markdown file is read when scanning for PlantUML blocks.
const maxPlantUMLScanSize = 1024 * 1024

var doctorReportPath string

var doctorCmd = &cobra.Command{
	Use:   "doctor [directory]",
	Short: "Report environment info and check PDF/project readiness",
	Long: `Report mdpress runtime environment details and run a small set of readiness checks, including:
  - Runtime platform information
  - Go runtime version (informational)
  - Chrome/Chromium availability for PDF output
  - Go version (>=1.25 recommended)
  - Git availability for remote source builds
  - Network connectivity to github.com
  - Disk space in output directory
  - CJK font availability for Asian text rendering
  - Plugin health and availability
  - Presence of book.yaml / SUMMARY.md / LANGS.md
  - Whether the project can be loaded
  - Whether Markdown chapter links stay inside the build graph`,
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
	doctorCmd.Flags().StringVar(&doctorReportPath, "report", "", "Write doctor report to .json or .md")
}

type doctorReport struct {
	Platform           string                   `json:"platform"`
	GoVersion          string                   `json:"go_version"`
	CacheDir           string                   `json:"cache_dir,omitempty"`
	CacheDisabled      bool                     `json:"cache_disabled"`
	ChromiumAvailable  bool                     `json:"chromium_available"`
	CJKFontsAvailable  bool                     `json:"cjk_fonts_available"`
	PlantUMLAvailable  bool                     `json:"plantuml_available"`
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
}

func executeDoctor(ctx context.Context, targetDir string) error {
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

	if err := pdf.CheckChromiumAvailable(); err != nil {
		utils.Error("Chromium/Chrome is unavailable (PDF output will fail)")
		fmt.Printf("    %s\n", err.Error())
		report.Warnings = append(report.Warnings, "Chromium/Chrome is unavailable (PDF output will fail)")
	} else {
		report.ChromiumAvailable = true
		utils.Success("Chromium/Chrome is available")
	}

	// Check Go version
	checkGoVersion(&report)

	// Check Git availability
	checkGitAvailable(&report)

	// Check Network connectivity
	checkNetworkConnectivity(&report)

	// Try to load project config once for reuse by multiple checks.
	bookPath := filepath.Join(absDir, "book.yaml")
	var doctorCfg *config.BookConfig
	if _, err := os.Stat(bookPath); err == nil {
		if cfg, loadErr := config.Load(bookPath); loadErr == nil {
			doctorCfg = cfg
		}
	}

	// Check disk space in output directory
	checkDiskSpace(absDir, doctorCfg, &report)

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

	if _, err := os.Stat(bookPath); err == nil {
		report.BookYAMLFound = true
		utils.Success("Detected book.yaml")
	} else {
		utils.Warning("book.yaml not found")
	}
	if _, err := os.Stat(summaryPath); err == nil {
		report.SummaryFound = true
		utils.Success("Detected SUMMARY.md")
	} else {
		utils.Warning("SUMMARY.md not found")
	}
	if _, err := os.Stat(langsPath); err == nil {
		report.LangsFound = true
		utils.Success("Detected LANGS.md")
		utils.Warning("Multi-language projects are built per language directory; root-level mixed-book output is not generated")
		report.Warnings = append(report.Warnings, "Multi-language projects are built per language directory; root-level mixed-book output is not generated")
	} else {
		utils.Warning("LANGS.md not found")
	}

	if doctorCfg != nil {
		report.ProjectLoadable = true
		report.ProjectTitle = doctorCfg.Book.Title
		report.TopLevelChapters = len(doctorCfg.Chapters)
		utils.Success("Config loads successfully: %s (%d top-level chapters)", doctorCfg.Book.Title, len(doctorCfg.Chapters))
		reportDoctorMarkdownLinks(doctorCfg, &report)
	} else if _, err := os.Stat(bookPath); err == nil {
		// book.yaml exists but failed to load — report the error.
		if _, loadErr := config.Load(bookPath); loadErr != nil {
			utils.Error("Failed to load book.yaml: %v", loadErr)
			report.Warnings = append(report.Warnings, fmt.Sprintf("Failed to load book.yaml: %v", loadErr))
		}
	} else if _, err := os.Stat(summaryPath); err == nil {
		cfg, discoverErr := config.Discover(ctx, absDir)
		if discoverErr != nil {
			utils.Error("Failed to auto-discover project from SUMMARY.md: %v", discoverErr)
			report.Warnings = append(report.Warnings, fmt.Sprintf("Failed to auto-discover project from SUMMARY.md: %v", discoverErr))
		} else {
			report.ProjectLoadable = true
			report.ProjectTitle = cfg.Book.Title
			report.TopLevelChapters = len(cfg.Chapters)
			utils.Success("Project can be loaded by auto-discovery: %s (%d top-level chapters)", cfg.Book.Title, len(cfg.Chapters))
			reportDoctorMarkdownLinks(cfg, &report)
		}
	} else {
		utils.Warning("No directly buildable book.yaml or SUMMARY.md found in the target directory")
		report.Warnings = append(report.Warnings, "No directly buildable book.yaml or SUMMARY.md found in the target directory")
	}

	if doctorReportPath != "" {
		if err := writeDoctorReport(doctorReportPath, report); err != nil {
			return fmt.Errorf("failed to write doctor report: %w", err)
		}
	}

	fmt.Println()
	return nil
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

	// Extract version number (e.g., "go1.25.0" -> "1.25.0")
	versionStr := strings.TrimPrefix(version, "go")
	parts := strings.Split(versionStr, ".")
	if len(parts) < 2 {
		utils.Warning("Could not parse Go version")
		return
	}

	// Check if major.minor >= 1.25
	major, minorErr := utils.ParseVersionPart(parts[0])
	if minorErr != nil {
		if verbose {
			utils.Warning("Could not parse Go major version: %v", minorErr)
		}
		return
	}

	minor, patchErr := utils.ParseVersionPart(parts[1])
	if patchErr != nil {
		if verbose {
			utils.Warning("Could not parse Go minor version: %v", patchErr)
		}
		return
	}

	if major > 1 || (major == 1 && minor >= 25) {
		utils.Success("Go version %s (>= 1.25)", versionStr)
	} else {
		utils.Warning("Go version %s (< 1.25) — some features may not work as expected", versionStr)
		report.Warnings = append(report.Warnings, fmt.Sprintf("Go version %s is below recommended 1.25", versionStr))
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

func checkNetworkConnectivity(report *doctorReport) {
	// Try an HTTP HEAD request to github.com (cross-platform connectivity check)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "HEAD", "https://github.com", nil)
	if err != nil {
		utils.Warning("Network connectivity check failed")
		if verbose {
			fmt.Printf("    Error: %v\n", err)
		}
		report.Warnings = append(report.Warnings, "Cannot reach github.com (network connectivity issue)")
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}

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

func checkDiskSpace(targetDir string, cfg *config.BookConfig, report *doctorReport) {
	outputDir := filepath.Join(targetDir, "output")

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
	dfCtx, dfCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dfCancel()
	out, err := exec.CommandContext(dfCtx, "df", "-k", outputDir).Output()
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
	availableGB := float64(availKB) / (1024 * 1024)
	report.DiskSpaceGB = availableGB

	// Threshold: 100 MB = 0.1 GB
	const minDiskSpaceGB = 0.1
	if availableGB < minDiskSpaceGB {
		utils.Error("Disk space critically low (%.2f GB available, < 100 MB required)", availableGB)
		report.Warnings = append(report.Warnings, fmt.Sprintf("Disk space critically low: only %.2f GB available", availableGB))
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

	for _, plugin := range cfg.Plugins {
		if plugin.Path == "" {
			allValid = false
			if verbose {
				utils.Error("Plugin %q has empty path", plugin.Name)
			} else {
				utils.Warning("Plugin %q has empty path", plugin.Name)
			}
			continue
		}

		pluginPath := plugin.Path
		if !filepath.IsAbs(pluginPath) {
			pluginPath = filepath.Join(targetDir, pluginPath)
		}

		// Check if the executable exists
		info, err := os.Stat(pluginPath)
		if err != nil {
			allValid = false
			if verbose {
				utils.Error("Plugin %q not found at %s", plugin.Name, pluginPath)
			} else {
				utils.Warning("Plugin %q not found", plugin.Name)
			}
			continue
		}

		// Check if it's executable
		if !isPluginExecutable(pluginPath, info.Mode()) {
			allValid = false
			if verbose {
				utils.Warning("Plugin %q is not executable", plugin.Name)
			}
		}
	}

	if allValid {
		report.PluginsValid = true
		utils.Success("All %d plugin(s) are valid", len(cfg.Plugins))
	} else {
		report.PluginsValid = false
		report.Warnings = append(report.Warnings, fmt.Sprintf("%d plugin(s) have issues", len(cfg.Plugins)))
	}
}

func isExecutable(mode os.FileMode) bool {
	return (mode & 0111) != 0
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

func checkPlantUML(targetDir string, report *doctorReport) {
	// Check if any markdown files contain plantuml blocks
	hasPlantumlBlocks := hasPlantUMLBlocks(targetDir)
	report.PlantUMLNeeded = hasPlantumlBlocks

	if !hasPlantumlBlocks {
		utils.Success("PlantUML not needed (no diagrams detected)")
		return
	}

	// Check for Java (required for PlantUML)
	_, javaErr := exec.LookPath("java")

	// Check for PLANTUML_JAR environment variable
	plantUMLJar := os.Getenv("PLANTUML_JAR")

	// Check for plantuml command in PATH
	_, plantUMLErr := exec.LookPath("plantuml")

	// Determine status
	if javaErr != nil && plantUMLJar == "" && plantUMLErr != nil {
		// Java not found and no plantuml configuration
		utils.Error("PlantUML not available")
		msg := "PlantUML not configured — diagrams will be skipped"
		report.Warnings = append(report.Warnings, msg)
		if javaErr != nil {
			fmt.Println("    Java not found — PlantUML diagrams will not render")
		}
		fmt.Println("    Install PlantUML: brew install plantuml")
		fmt.Println("    Or set PLANTUML_JAR=/path/to/plantuml.jar environment variable")
		return
	}

	if javaErr != nil {
		// Java is required but not found
		utils.Warning("Java not found — PlantUML diagrams will not render")
		msg := "Java not found — PlantUML diagrams will not render"
		report.Warnings = append(report.Warnings, msg)
		return
	}

	// Java is available; check if we can use plantuml
	if plantUMLJar != "" {
		if _, err := os.Stat(plantUMLJar); err == nil {
			// PLANTUML_JAR is set and points to a valid file
			report.PlantUMLAvailable = true
			utils.Success("PlantUML available (via PLANTUML_JAR)")
		} else {
			// PLANTUML_JAR is set but file doesn't exist
			utils.Warning("PLANTUML_JAR is set but points to non-existent file: %s", plantUMLJar)
			msg := fmt.Sprintf("PLANTUML_JAR is set but points to non-existent file: %s", plantUMLJar)
			report.Warnings = append(report.Warnings, msg)
		}
	} else if plantUMLErr == nil {
		// plantuml command found in PATH
		report.PlantUMLAvailable = true
		utils.Success("PlantUML available (via plantuml command)")
	} else {
		// No plantuml command found, but Java is available
		utils.Warning("PlantUML command not found — install via: brew install plantuml")
		msg := "PlantUML command not found (Java is available but plantuml is not installed)"
		report.Warnings = append(report.Warnings, msg)
	}
}

// hasPlantUMLBlocks checks if any markdown files in the directory contain plantuml code blocks.
func hasPlantUMLBlocks(targetDir string) bool {
	return searchPlantUMLInDir(targetDir)
}

// searchPlantUMLInDir recursively searches for ```plantuml blocks in markdown files.
func searchPlantUMLInDir(dir string) bool {
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
			if searchPlantUMLInDir(path) {
				return true
			}
		} else if strings.HasSuffix(entry.Name(), ".md") {
			// Limit read to 1 MB to prevent OOM on very large files.
			f, err := os.Open(path)
			if err != nil {
				continue
			}
			content, err := io.ReadAll(io.LimitReader(f, maxPlantUMLScanSize))
			f.Close() //nolint:errcheck
			if err != nil {
				continue
			}
			if strings.Contains(string(content), "```plantuml") {
				return true
			}
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
		return os.WriteFile(absPath, data, 0644)
	case ".md":
		return os.WriteFile(absPath, []byte(renderDoctorMarkdown(report)), 0644)
	default:
		return fmt.Errorf("unsupported report extension: %s (use .json or .md)", filepath.Ext(absPath))
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
	fmt.Fprintf(&b, "- CJK fonts available: %t\n", report.CJKFontsAvailable)
	fmt.Fprintf(&b, "- Git available: %t\n", report.GitAvailable)
	fmt.Fprintf(&b, "- Network connectivity: %t\n", report.NetworkAvailable)
	if report.DiskSpaceGB > 0 {
		fmt.Fprintf(&b, "- Disk space available: %.2f GB\n", report.DiskSpaceGB)
	}
	fmt.Fprintf(&b, "- Disk space OK: %t\n", report.DiskSpaceOK)
	fmt.Fprintf(&b, "- PlantUML needed: %t\n", report.PlantUMLNeeded)
	fmt.Fprintf(&b, "- PlantUML available: %t\n", report.PlantUMLAvailable)
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
