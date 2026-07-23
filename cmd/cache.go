// cache.go implements `mdpress cache`, the CLI entry point for inspecting and
// clearing the runtime cache directory. Until it existed the parsed-chapter
// cache was write-only from the user's point of view: it grew with every edit
// of every chapter and the only way to reclaim the space was to know the
// directory layout and delete it by hand.
package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yeasy/mdpress/pkg/utils"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Inspect or clear the mdpress runtime cache",
	Long: `Inspect or clear the caches mdpress keeps between builds.

Subcommands:
  cache info   Show the cache location, entry count, and size
  cache clear  Delete every cached entry

mdpress caches parsed chapters (and other build intermediates) under a runtime
cache directory so unchanged chapters are not re-rendered. Entries unused for
two weeks are pruned automatically; this command is for reclaiming the space
now, or for forcing a fully cold rebuild.

The location can be overridden with --cache-dir or MDPRESS_CACHE_DIR, and
caching can be turned off for a single command with --no-cache.

Examples:
  mdpress cache info
  mdpress cache clear`,

	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		hint := ""
		if suggestions := cmd.SuggestionsFor(args[0]); len(suggestions) > 0 {
			hint = fmt.Sprintf("\n\nDid you mean this?\n\t%s", strings.Join(suggestions, "\n\t"))
		}
		return fmt.Errorf("unknown cache sub-command %q%s\n\nRun 'mdpress cache --help' to see the available sub-commands", args[0], hint)
	},
}

var cacheInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show the cache location, entry count, and size",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeCacheInfo(utils.CacheRootDir())
	},
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Delete every cached entry",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeCacheClear(utils.CacheRootDir())
	},
}

func init() {
	cacheCmd.SuggestionsMinimumDistance = 2
	cacheCmd.AddCommand(cacheInfoCmd)
	cacheCmd.AddCommand(cacheClearCmd)
}

// formatCacheSize renders a byte count in the largest unit that keeps it
// readable, so a 133 MB cache does not report itself as 139460608.
func formatCacheSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	value := float64(n)
	units := []string{"KB", "MB", "GB", "TB"}
	var suffix string
	for _, u := range units {
		value /= unit
		suffix = u
		if value < unit {
			break
		}
	}
	return fmt.Sprintf("%.1f %s", value, suffix)
}

// cacheUsage is the measured size of one cache directory.
type cacheUsage struct {
	// Files counts regular files, i.e. cache entries.
	Files int
	// Bytes is their total size on disk.
	Bytes int64
}

// measureCacheDir walks root and totals the entries it holds. A missing
// directory is not an error: never having built anything is a valid state.
func measureCacheDir(root string) (cacheUsage, error) {
	var usage cacheUsage
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// An unreadable subtree should not abort the whole report.
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			return nil //nolint:nilerr // a file that vanished mid-walk is not a failure
		}
		usage.Files++
		usage.Bytes += info.Size()
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return usage, err
	}
	return usage, nil
}

// cacheSubdirUsage measures each immediate subdirectory of root, so `cache
// info` can attribute the space to the cache that is actually using it.
func cacheSubdirUsage(root string) (map[string]cacheUsage, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	usage := make(map[string]cacheUsage)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sub, err := measureCacheDir(filepath.Join(root, entry.Name()))
		if err != nil {
			return nil, err
		}
		usage[entry.Name()] = sub
	}
	return usage, nil
}

func executeCacheInfo(root string) error {
	utils.Header("mdpress Cache")
	fmt.Println()
	fmt.Printf("  Location: %s\n", root)

	if utils.CacheDisabled() {
		utils.Warning("Caching is disabled for this command (--no-cache / MDPRESS_DISABLE_CACHE)")
	}

	if _, err := os.Stat(root); os.IsNotExist(err) {
		fmt.Println("  Entries:  0 (cache directory does not exist yet)")
		return nil
	}

	total, err := measureCacheDir(root)
	if err != nil {
		return fmt.Errorf("failed to measure cache directory: %w", err)
	}
	fmt.Printf("  Entries:  %d\n", total.Files)
	fmt.Printf("  Size:     %s\n", formatCacheSize(total.Bytes))

	subUsage, err := cacheSubdirUsage(root)
	if err != nil {
		return fmt.Errorf("failed to measure cache directory: %w", err)
	}
	names := make([]string, 0, len(subUsage))
	for name := range subUsage {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) > 0 {
		fmt.Println()
		for _, name := range names {
			u := subUsage[name]
			fmt.Printf("    %-20s %6d entries  %10s\n", name, u.Files, formatCacheSize(u.Bytes))
		}
	}

	fmt.Println()
	fmt.Println("  Run 'mdpress cache clear' to reclaim this space.")
	return nil
}

func executeCacheClear(root string) error {
	usage, err := measureCacheDir(root)
	if err != nil {
		return fmt.Errorf("failed to measure cache directory: %w", err)
	}
	if err := os.RemoveAll(root); err != nil {
		return fmt.Errorf("failed to remove cache directory %s: %w", root, err)
	}
	if usage.Files == 0 {
		utils.Success("Cache is already empty: %s", root)
		return nil
	}
	utils.Success("Cleared %d cache entries (%s) from %s", usage.Files, formatCacheSize(usage.Bytes), root)
	return nil
}
