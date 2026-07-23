package cmd

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// versionJSON requests machine-readable output from `mdpress version`.
var versionJSON bool

// versionInfo is the machine-readable build description. Every field is
// always present, including empty ones: a CI step reading `.commit` should get
// "" for an unstamped build rather than a missing key it has to special-case.
type versionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuiltAt   string `json:"built_at"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of mdpress",
	Long: `Display the version, build time, and other build information for mdpress.

Use --json to get the same information in a form scripts can read:
  mdpress version --json | jq -r .version`,
	Args: cobra.NoArgs,
	// RunE rather than Run, so a JSON encoding failure exits non-zero instead
	// of printing nothing and reporting success.
	RunE: func(cmd *cobra.Command, args []string) error {
		return runVersion()
	},
}

func init() {
	versionCmd.Flags().BoolVar(&versionJSON, "json", false, "Print build information as JSON")
}

// runVersion prints the build information in the requested form.
func runVersion() error {
	if versionJSON {
		return printVersionJSON()
	}
	fmt.Printf("mdpress version %s\n", Version)
	if Commit != "" {
		fmt.Printf("Commit %s\n", Commit)
	}
	if BuildTime != "unknown" && BuildTime != "" {
		fmt.Printf("Built at %s\n", BuildTime)
	}
	return nil
}

// printVersionJSON writes the build information as a single JSON object.
func printVersionJSON() error {
	info := versionInfo{
		Version:   Version,
		Commit:    Commit,
		BuiltAt:   BuildTime,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
	// "unknown" is the placeholder the human-readable output suppresses; in
	// JSON an empty string is the honest way to say "not recorded".
	if info.BuiltAt == "unknown" {
		info.BuiltAt = ""
	}
	encoded, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode version information: %w", err)
	}
	fmt.Println(string(encoded))
	return nil
}
