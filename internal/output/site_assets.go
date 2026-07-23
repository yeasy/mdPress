// site_assets.go turns embedded images back into files, and copies a
// project's static/ directory into the site output.
//
// The chapter pipeline embeds local images as base64 data URIs, which is right
// for the single-file HTML and for Chrome's PDF rendering but wrong for a
// deployed site: a 1 MB screenshot becomes ~1.4 MB of markup *on every page
// that references it*, nothing is cacheable across pages, and loading="lazy"
// is a no-op on a data: URI so everything blocks first paint. Extracting the
// images here keeps the pipeline single-pass and confines the difference to
// the one format that needs it.
package output

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yeasy/mdpress/pkg/utils"
)

// siteAssetDir is the subdirectory of the site output that extracted images
// are written to.
const siteAssetDir = "assets"

// staticSourceDir is the project directory whose contents are copied verbatim
// into the site root. It is how a user ships CNAME, .nojekyll, robots.txt, a
// favicon, or anything else the generator does not produce — previously there
// was no supported way, and files placed in _book/ by hand were destroyed by
// the next build's atomic swap.
const staticSourceDir = "static"

// dataURIPattern matches a base64 image data URI inside a src attribute.
var dataURIPattern = regexp.MustCompile(`src="data:(image/[a-zA-Z0-9.+-]+);base64,([^"]+)"`)

// dataURIExtensions maps the media types the pipeline emits to file
// extensions. Anything else keeps its data URI rather than being written out
// under a guessed name.
var dataURIExtensions = map[string]string{
	"image/png":     ".png",
	"image/jpeg":    ".jpg",
	"image/gif":     ".gif",
	"image/webp":    ".webp",
	"image/svg+xml": ".svg",
	"image/avif":    ".avif",
	"image/bmp":     ".bmp",
	"image/x-icon":  ".ico",
}

// assetExtractor writes embedded images to disk, deduplicating by content so
// an image used on several pages (or repeated on the homepage) is stored once.
type assetExtractor struct {
	outputDir string
	// names maps content hash to the written file's name.
	names map[string]string
}

func newAssetExtractor(outputDir string) *assetExtractor {
	return &assetExtractor{outputDir: outputDir, names: map[string]string{}}
}

// Extract rewrites every base64 image in html to a file reference, relative to
// the page at pageFilename. Images it cannot decode or name are left inline,
// so a page never loses an image to this pass.
func (a *assetExtractor) Extract(html, pageFilename string) (string, error) {
	var firstErr error
	result := dataURIPattern.ReplaceAllStringFunc(html, func(match string) string {
		parts := dataURIPattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		mediaType, payload := parts[1], parts[2]
		ext, known := dataURIExtensions[strings.ToLower(mediaType)]
		if !known {
			return match
		}
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return match // not something we can write out; keep it inline
		}

		sum := sha256.Sum256(data)
		hash := hex.EncodeToString(sum[:])
		name, seen := a.names[hash]
		if !seen {
			name = "img-" + hash[:16] + ext
			if err := a.write(name, data); err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return match
			}
			a.names[hash] = name
		}
		return `src="` + relativeSiteHref(pageFilename, siteAssetDir+"/"+name) + `"`
	})
	return result, firstErr
}

// write stores one asset under the output directory's asset folder.
func (a *assetExtractor) write(name string, data []byte) error {
	dir := filepath.Join(a.outputDir, siteAssetDir)
	if err := utils.EnsureDir(dir); err != nil {
		return fmt.Errorf("create asset directory: %w", err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil { //nolint:gosec // G306: published site assets must be world-readable
		return fmt.Errorf("write asset %s: %w", name, err)
	}
	return nil
}

// copyStaticDir copies bookRoot/static into the site output, preserving
// relative layout. A missing directory is not an error. Returns the number of
// files copied.
func copyStaticDir(bookRoot, outputDir string) (int, error) {
	if bookRoot == "" {
		return 0, nil
	}
	source := filepath.Join(bookRoot, staticSourceDir)
	info, err := os.Stat(source)
	if err != nil || !info.IsDir() {
		return 0, nil //nolint:nilerr // no static/ directory is the normal case
	}

	copied := 0
	err = filepath.WalkDir(source, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, relErr := filepath.Rel(source, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(outputDir, rel)
		// Defense in depth: a symlinked entry must not escape the output dir.
		if !strings.HasPrefix(filepath.Clean(target)+string(os.PathSeparator),
			filepath.Clean(outputDir)+string(os.PathSeparator)) {
			return fmt.Errorf("static file %s escapes the output directory", rel)
		}
		if d.IsDir() {
			return utils.EnsureDir(target)
		}
		if !d.Type().IsRegular() {
			return nil // skip symlinks, sockets, devices
		}
		if err := copyFile(path, target); err != nil {
			return err
		}
		copied++
		return nil
	})
	if err != nil {
		return copied, fmt.Errorf("copy %s directory: %w", staticSourceDir, err)
	}
	return copied, nil
}

// copyFile copies one regular file, creating the destination directory.
func copyFile(source, target string) error {
	if err := utils.EnsureDir(filepath.Dir(target)); err != nil {
		return err
	}
	in, err := os.Open(source) //nolint:gosec // G304: path comes from walking the project's own static dir
	if err != nil {
		return err
	}
	defer in.Close() //nolint:errcheck

	out, err := os.Create(target) //nolint:gosec // G304: destination is inside the validated output dir
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close() //nolint:errcheck
		return err
	}
	return out.Close()
}
