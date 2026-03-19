// local.go implements local directory sources.
package source

import (
	"fmt"
	"os"
	"path/filepath"
)

// LocalSource reads content directly from the local filesystem.
type LocalSource struct {
	path string  // Local path.
	opts Options // Source options.
}

// NewLocalSource creates a local source.
func NewLocalSource(path string, opts Options) *LocalSource {
	return &LocalSource{
		path: path,
		opts: opts,
	}
}

// Prepare validates the local directory and returns its path.
func (s *LocalSource) Prepare() (string, error) {
	// Resolve the path to an absolute directory.
	absPath, err := filepath.Abs(s.path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	// Ensure the path exists and is a directory.
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("path does not exist: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", absPath)
	}

	// Apply an optional subdirectory.
	targetDir := absPath
	if s.opts.SubDir != "" {
		targetDir = filepath.Join(absPath, s.opts.SubDir)
		info, err = os.Stat(targetDir)
		if err != nil {
			return "", fmt.Errorf("subdirectory does not exist: %w", err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("subdirectory path is not a directory: %s", targetDir)
		}
	}

	return targetDir, nil
}

// Cleanup is a no-op for local sources.
func (s *LocalSource) Cleanup() error {
	return nil
}

// Type returns the source type.
func (s *LocalSource) Type() string {
	return "local"
}
