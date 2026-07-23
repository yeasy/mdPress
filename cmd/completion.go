// completion.go implements the shell completion command for mdpress.
// It generates shell completion scripts for bash, zsh, fish, and powershell.
package cmd

import (
	"fmt"
	"os"
	"strings"

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
  # Load bash completions in your current shell session
  source <(mdpress completion bash)

  # Load zsh completions in your current shell session
  source <(mdpress completion zsh)

  # Load fish completions in your current shell session
  mdpress completion fish | source

  # Generate powershell completion
  mdpress completion powershell`,

	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeCompletion(args[0])
	},
	// The shell names are also the sub-command names, which cobra already
	// completes with their descriptions, so no ValidArgs here: listing them
	// again only produced every candidate twice.
}

// bashCompletionCmd generates bash completion script.
var bashCompletionCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generate bash completion script",
	Long: `Generate bash completion script for mdpress.

To load completions in your current shell session, run:
  source <(mdpress completion bash)

Or to permanently install them (requires the bash-completion package), run once:
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

If shell completion is not already enabled in your environment,
you will need to enable it once first:
  echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session, run:
  source <(mdpress completion zsh)

Or to permanently install them, write the script to a directory
in your $fpath, e.g.:
  mdpress completion zsh > "${fpath[1]}/_mdpress"`,

	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenZshCompletion(os.Stdout)
	},
}

// fishCompletionCmd generates fish completion script.
var fishCompletionCmd = &cobra.Command{
	Use:   "fish",
	Short: "Generate fish shell completion script",
	Long: `Generate fish shell completion script for mdpress.

To load completions in your current shell session, run:
  mdpress completion fish | source

Or to permanently install them, run once:
  mdpress completion fish > ~/.config/fish/completions/mdpress.fish`,

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

To load completions in your current shell session, run:
  mdpress completion powershell | Out-String | Invoke-Expression

Or to permanently install them in your PowerShell profile, run:
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

// Shell completion for flag values.
//
// Cobra only completes flag *names* by itself; a flag whose value comes from a
// fixed set (--format, a theme name) falls back to file-name completion unless
// a completion function is registered for it. That made every value of every
// mdpress flag complete to the contents of the current directory, which is
// never a valid answer for those flags.

// registerFixedFlagCompletion makes flagName complete from a fixed value set.
// Entries may carry a "value\tdescription" suffix, which shells that support
// descriptions display alongside the candidate.
func registerFixedFlagCompletion(cmd *cobra.Command, flagName string, values []string) {
	// The error is only returned for a flag that does not exist, i.e. a typo
	// in the call right below the flag definition.
	_ = cmd.RegisterFlagCompletionFunc(flagName, func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return filterCompletions(values, toComplete), cobra.ShellCompDirectiveNoFileComp
	})
}

// filterCompletions keeps the candidates that start with toComplete. Cobra
// passes the function's results to the shell unfiltered, so completion
// functions have to do this themselves.
func filterCompletions(values []string, toComplete string) []string {
	matches := make([]string, 0, len(values))
	for _, value := range values {
		if strings.HasPrefix(completionValue(value), toComplete) {
			matches = append(matches, value)
		}
	}
	return matches
}

// completionValue strips the optional description from a completion entry.
func completionValue(entry string) string {
	value, _, _ := strings.Cut(entry, "\t")
	return value
}

// completeCommaSeparated completes one element of a comma-separated list,
// keeping the already-typed elements as a prefix and dropping the ones that
// have been chosen. Without this, completing --format stopped working the
// moment a comma was typed.
func completeCommaSeparated(values []string, toComplete string) []string {
	prefix := ""
	last := toComplete
	if idx := strings.LastIndex(toComplete, ","); idx >= 0 {
		prefix = toComplete[:idx+1]
		last = toComplete[idx+1:]
	}

	chosen := make(map[string]bool)
	for _, part := range strings.Split(prefix, ",") {
		if part = strings.TrimSpace(part); part != "" {
			chosen[part] = true
		}
	}

	matches := make([]string, 0, len(values))
	for _, value := range values {
		name, desc, hasDesc := strings.Cut(value, "\t")
		if chosen[name] || !strings.HasPrefix(name, last) {
			continue
		}
		entry := prefix + name
		if hasDesc {
			entry += "\t" + desc
		}
		matches = append(matches, entry)
	}
	return matches
}

// completeDirectories restricts completion to directory names, for arguments
// that name a project directory.
func completeDirectories(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveFilterDirs
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
