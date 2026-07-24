// workflow_example_test.go checks the shipped GitHub Actions example against
// how mdpress actually writes its output.
//
// The example told users to copy it into their own book repository, and its
// deploy job uploaded "<output>_site/" to GitHub Pages. Since v0.8.0
// "--format site --output X" writes into X itself -- the "_site" sibling only
// appears when --output is shared with other formats -- so the deploy failed
// with "No files were found with the provided path" and named a directory that
// exists nowhere in mdpress's output. No CI job runs this example, so nothing
// caught it; this test is the cheap substitute.
package tests

import (
	"os"
	"path"
	"regexp"
	"strings"
	"testing"
)

const exampleWorkflowPath = "../.github/workflows/examples/mdpress-build.yml"

// expandWorkflowEnv resolves ${{ env.NAME }} references from the workflow's own
// env block.
func expandWorkflowEnv(t *testing.T, value string, env map[string]string) string {
	t.Helper()
	ref := regexp.MustCompile(`\$\{\{\s*env\.(\w+)\s*\}\}`)
	return ref.ReplaceAllStringFunc(value, func(match string) string {
		name := ref.FindStringSubmatch(match)[1]
		resolved, ok := env[name]
		if !ok {
			t.Fatalf("workflow references ${{ env.%s }} but the env block does not define it", name)
		}
		return resolved
	})
}

// TestExampleWorkflowUploadsTheDirectoryItBuilds asserts the Pages upload path
// is the directory the site build was told to write to.
func TestExampleWorkflowUploadsTheDirectoryItBuilds(t *testing.T) {
	raw, err := os.ReadFile(exampleWorkflowPath)
	if err != nil {
		t.Fatalf("cannot read %s: %v", exampleWorkflowPath, err)
	}
	workflow := string(raw)

	env := map[string]string{}
	for _, name := range []string{"BOOK_DIR", "OUTPUT_DIR"} {
		match := regexp.MustCompile(`(?m)^\s+` + name + `:\s*"([^"]*)"`).FindStringSubmatch(workflow)
		if match == nil {
			t.Fatalf("workflow has no %s in its env block", name)
		}
		env[name] = match[1]
	}

	// The captures run to end of line on purpose: "${{ env.X }}" contains
	// spaces, so a \S+ capture stops at the first brace and compares nothing.
	buildMatch := regexp.MustCompile(`(?m)^\s*mdpress build --format site --output +(.+?)\s*$`).FindStringSubmatch(workflow)
	if buildMatch == nil {
		t.Fatal("workflow no longer contains a 'mdpress build --format site --output ...' step")
	}
	uploadMatch := regexp.MustCompile(`(?m)uses: actions/upload-pages-artifact@[^\n]*\n(?:[^\n]*\n)*?\s+path: +(.+?)\s*$`).FindStringSubmatch(workflow)
	if uploadMatch == nil {
		t.Fatal("workflow no longer contains an actions/upload-pages-artifact step with a path")
	}

	// The build step runs with working-directory: BOOK_DIR, the upload path is
	// repository-relative, so BOOK_DIR is prefixed to compare like for like.
	built := path.Clean(path.Join(env["BOOK_DIR"], expandWorkflowEnv(t, buildMatch[1], env)))
	uploaded := path.Clean(strings.TrimSuffix(expandWorkflowEnv(t, uploadMatch[1], env), "/"))

	if built != uploaded {
		t.Errorf("the deploy job builds the site into %q but uploads %q to Pages; "+
			"'--format site --output X' writes to X, so the upload path must be X",
			built, uploaded)
	}
}
