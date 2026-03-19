package tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const realBooksDirEnv = "MDPRESS_REAL_BOOKS_DIR"

func TestRealSample_DockerPracticeHTMLBuild(t *testing.T) {
	src := requireRealSampleDir(t, "docker_practice")
	workDir := mirrorSampleTree(t, src)
	outputBase := filepath.Join(t.TempDir(), "docker.html")

	runMdpressBuild(t, workDir, "html", outputBase)

	outputPath := strings.TrimSuffix(outputBase, filepath.Ext(outputBase)) + ".html"
	html := readTextFile(t, outputPath)

	if !strings.Contains(html, "Docker 简介") {
		t.Fatalf("output HTML missing expected book title content: %s", outputPath)
	}

	for _, disallowed := range []string{
		`href="../02_basic_concept/README.md"`,
		`href="../04_image/4.5_build.md"`,
		`href="1.1_quickstart.md"`,
		`href="../12_implementation/README.md"`,
	} {
		if strings.Contains(html, disallowed) {
			t.Fatalf("output HTML still contains unrewritten internal markdown link %q", disallowed)
		}
	}
}

func TestRealSample_LearningPickleballLangsIndependentOutputs(t *testing.T) {
	src := requireRealSampleDir(t, "learning_pickleball")
	workDir := mirrorSampleTree(t, src)
	outputBase := filepath.Join(t.TempDir(), "learning.html")

	runMdpressBuild(t, workDir, "html", outputBase)

	landingPath := filepath.Join(filepath.Dir(outputBase), "learning-index.html")
	landingHTML := readTextFile(t, landingPath)
	if !strings.Contains(landingHTML, "Language Variants") {
		t.Fatalf("multilingual landing page missing expected title: %s", landingPath)
	}

	htmlFiles := collectHTMLFiles(t, filepath.Dir(outputBase))
	if len(htmlFiles) < 2 {
		t.Fatalf("expected separate HTML outputs for multilingual build, found %d file(s): %v", len(htmlFiles), htmlFiles)
	}

	var zhFile, enFile string
	var zhHTML, enHTML string

	for _, path := range htmlFiles {
		content := readTextFile(t, path)
		if zhFile == "" && strings.Contains(content, "第 1 章 - 背景知识") {
			zhFile = path
			zhHTML = content
		}
		if enFile == "" && strings.Contains(content, "Chapter 1 - Background Knowledge") {
			enFile = path
			enHTML = content
		}
	}

	if zhFile == "" {
		t.Fatalf("did not find a dedicated Chinese HTML output under %s", workDir)
	}
	if enFile == "" {
		t.Fatalf("did not find a dedicated English HTML output under %s", workDir)
	}
	if zhFile == enFile {
		t.Fatalf("Chinese and English outputs resolved to the same file: %s", zhFile)
	}

	if strings.Contains(zhHTML, "Chapter 1 - Background Knowledge") {
		t.Fatalf("Chinese output %s still contains English chapter content", zhFile)
	}
	if strings.Contains(enHTML, "第 1 章 - 背景知识") {
		t.Fatalf("English output %s still contains Chinese chapter content", enFile)
	}
	if !strings.Contains(landingHTML, filepath.Base(zhFile)) || !strings.Contains(landingHTML, filepath.Base(enFile)) {
		t.Fatalf("multilingual landing page does not link to both language outputs: %s", landingPath)
	}
}

func requireRealSampleDir(t *testing.T, name string) string {
	t.Helper()

	base := os.Getenv(realBooksDirEnv)
	if base == "" {
		t.Skipf("skipping real sample regression: set %s to the books directory containing %s", realBooksDirEnv, name)
	}

	baseAbs, err := filepath.Abs(base)
	if err != nil {
		t.Skipf("skipping real sample regression: cannot resolve %s: %v", base, err)
	}

	sampleDir := filepath.Join(baseAbs, name)
	info, err := os.Stat(sampleDir)
	if err != nil || !info.IsDir() {
		t.Skipf("skipping real sample regression: sample directory not found: %s", sampleDir)
	}

	return sampleDir
}

func mirrorSampleTree(t *testing.T, src string) string {
	t.Helper()

	dstRoot := t.TempDir()
	dst := filepath.Join(dstRoot, filepath.Base(src))
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatalf("create temp mirror dir failed: %v", err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("read sample dir failed: %v", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if err := os.Symlink(srcPath, dstPath); err != nil {
			t.Fatalf("mirror entry %s failed: %v", entry.Name(), err)
		}
	}

	return dst
}

func runMdpressBuild(t *testing.T, sourceDir string, format string, outputPath string) string {
	t.Helper()

	repoRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve repo root failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	args := []string{"run", ".", "build", "--format", format}
	if outputPath != "" {
		args = append(args, "--output", outputPath)
	}
	args = append(args, sourceDir)
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("mdpress build failed: %v\noutput:\n%s", err, string(output))
	}

	return string(output)
}

func collectHTMLFiles(t *testing.T, root string) []string {
	t.Helper()

	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == "node_modules" || base == "_book" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".html") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("collect HTML files failed: %v", err)
	}

	return files
}

func readTextFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file failed (%s): %v", path, err)
	}

	return string(data)
}
