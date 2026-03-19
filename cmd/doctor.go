package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/pdf"
	"github.com/yeasy/mdpress/pkg/utils"
)

var doctorReportPath string

var doctorCmd = &cobra.Command{
	Use:   "doctor [directory]",
	Short: "Report environment info and check PDF/project readiness",
	Long: `Report mdpress runtime environment details and run a small set of readiness checks, including:
  - Runtime platform information
  - Go runtime version (informational)
  - Chrome/Chromium availability for PDF output
  - Presence of book.yaml / SUMMARY.md / LANGS.md
  - Whether the project can be loaded
  - Whether Markdown chapter links stay inside the build graph`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir := "."
		if len(args) > 0 {
			targetDir = args[0]
		}
		return executeDoctor(targetDir)
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
	BookYAMLFound      bool                     `json:"book_yaml_found"`
	SummaryFound       bool                     `json:"summary_found"`
	LangsFound         bool                     `json:"langs_found"`
	ProjectLoadable    bool                     `json:"project_loadable"`
	ProjectTitle       string                   `json:"project_title,omitempty"`
	TopLevelChapters   int                      `json:"top_level_chapters,omitempty"`
	Warnings           []string                 `json:"warnings,omitempty"`
	UnresolvedMarkdown []unresolvedMarkdownLink `json:"unresolved_markdown_links,omitempty"`
}

func executeDoctor(targetDir string) error {
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

	fmt.Println()
	utils.Header("Project Check")

	bookPath := filepath.Join(absDir, "book.yaml")
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

	if _, err := os.Stat(bookPath); err == nil {
		cfg, loadErr := config.Load(bookPath)
		if loadErr != nil {
			utils.Error("Failed to load book.yaml: %v", loadErr)
			report.Warnings = append(report.Warnings, fmt.Sprintf("Failed to load book.yaml: %v", loadErr))
		} else {
			report.ProjectLoadable = true
			report.ProjectTitle = cfg.Book.Title
			report.TopLevelChapters = len(cfg.Chapters)
			utils.Success("Config loads successfully: %s (%d top-level chapters)", cfg.Book.Title, len(cfg.Chapters))
			reportDoctorMarkdownLinks(cfg, &report)
		}
	} else if _, err := os.Stat(summaryPath); err == nil {
		cfg, discoverErr := config.Discover(absDir)
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

func writeDoctorReport(path string, report doctorReport) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := utils.EnsureDir(filepath.Dir(absPath)); err != nil {
		return err
	}

	switch strings.ToLower(filepath.Ext(absPath)) {
	case ".json":
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
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
	fmt.Fprintf(&b, "- Cache disabled: %t\n", report.CacheDisabled)
	if report.CacheDir != "" {
		fmt.Fprintf(&b, "- Cache dir: %s\n", report.CacheDir)
	}
	fmt.Fprintf(&b, "- Chromium available: %t\n", report.ChromiumAvailable)
	fmt.Fprintf(&b, "- CJK fonts available: %t\n", report.CJKFontsAvailable)
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
