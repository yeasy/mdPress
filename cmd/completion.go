// completion.go implements the shell completion command for mdpress.
// It generates shell completion scripts for bash, zsh, fish, and powershell.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// completionCmd is the main completion subcommand.
// When invoked as "mdpress completion <shell>", it generates the completion
// script with descriptions enabled (the default). For finer control, use the
// per-shell subcommands which accept --no-descriptions where supported.
var completionCmd = &cobra.Command{
	Use:   "completion <shell>",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for mdpress.

The completion subcommands generate shell completion scripts that allow
mdpress commands and options to be auto-completed when typing in a shell.

Supported shells:
  bash       - Bash completion script
  zsh        - Zsh completion script
  fish       - Fish shell completion script
  powershell - PowerShell completion script

Examples:
  # Generate bash completion and load it immediately
  mdpress completion bash | source

  # Generate zsh completion and load it immediately
  mdpress completion zsh | source

  # Generate fish completion
  mdpress completion fish

  # Generate powershell completion
  mdpress completion powershell`,

	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeCompletion(args[0])
	},
}

// bashCompletionCmd generates bash completion script.
var bashCompletionCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generate bash completion script",
	Long: `Generate bash completion script for mdpress.

To use this, run:
  mdpress completion bash | source

Or to permanently install it, save to a file:
  mdpress completion bash | sudo tee /etc/bash_completion.d/mdpress`,

	RunE: func(cmd *cobra.Command, args []string) error {
		noDesc, _ := cmd.Flags().GetBool("no-descriptions")
		return rootCmd.GenBashCompletionV2(os.Stdout, !noDesc)
	},
}

// zshCompletionCmd generates zsh completion script.
var zshCompletionCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generate zsh completion script",
	Long: `Generate zsh completion script for mdpress.

To use this, run:
  mdpress completion zsh | source

Or to permanently install it, add to your ~/.zshrc:
  mdpress completion zsh | source`,

	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenZshCompletion(os.Stdout)
	},
}

// fishCompletionCmd generates fish completion script.
var fishCompletionCmd = &cobra.Command{
	Use:   "fish",
	Short: "Generate fish shell completion script",
	Long: `Generate fish shell completion script for mdpress.

To use this, run:
  mdpress completion fish | source

Or to permanently install it:
  mdpress completion fish | sudo tee /usr/share/fish/vendor_completions.d/mdpress.fish`,

	RunE: func(cmd *cobra.Command, args []string) error {
		noDesc, _ := cmd.Flags().GetBool("no-descriptions")
		return rootCmd.GenFishCompletion(os.Stdout, !noDesc)
	},
}

// powershellCompletionCmd generates powershell completion script.
var powershellCompletionCmd = &cobra.Command{
	Use:   "powershell",
	Short: "Generate PowerShell completion script",
	Long: `Generate PowerShell completion script for mdpress.

To use this in your PowerShell profile, run:
  mdpress completion powershell | Out-String | Out-File -FilePath $PROFILE -Append`,

	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenPowerShellCompletion(os.Stdout)
	},
}

// executeCompletion handles the parent-command path ("mdpress completion <shell>").
// Descriptions are always enabled here; use the per-shell subcommands with
// --no-descriptions for control over that behavior.
func executeCompletion(shell string) error {
	switch shell {
	case "bash":
		return rootCmd.GenBashCompletionV2(os.Stdout, true)
	case "zsh":
		return rootCmd.GenZshCompletion(os.Stdout)
	case "fish":
		return rootCmd.GenFishCompletion(os.Stdout, true)
	case "powershell":
		return rootCmd.GenPowerShellCompletion(os.Stdout)
	default:
		return fmt.Errorf("unsupported shell: %s\nSupported shells: bash, zsh, fish, powershell", shell)
	}
}

func init() {
	// Register completion subcommands.
	completionCmd.AddCommand(bashCompletionCmd)
	completionCmd.AddCommand(zshCompletionCmd)
	completionCmd.AddCommand(fishCompletionCmd)
	completionCmd.AddCommand(powershellCompletionCmd)

	// Add --no-descriptions flag to shells that support it (bash and fish).
	// Zsh and PowerShell generators do not accept a descriptions toggle.
	bashCompletionCmd.Flags().Bool("no-descriptions", false, "Disable completion item descriptions")
	fishCompletionCmd.Flags().Bool("no-descriptions", false, "Disable completion item descriptions")
}
