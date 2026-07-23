// init_chapters.go turns the flat list of Markdown files `mdpress init` finds
// into the nested chapter tree written to book.yaml. A project organized into
// directories used to come out as one long flat list, so the structure the
// author had already expressed on disk was thrown away at init time.
package cmd

import (
	"slices"
	"strings"

	"github.com/yeasy/mdpress/internal/config"
)

// chapterNode is one entry in the generated `chapters:` tree.
type chapterNode struct {
	Title    string
	File     string
	Sections []chapterNode
}

// dirGroup collects the files and subdirectories of one directory.
type dirGroup struct {
	files map[string]discoveredFile
	subs  map[string]*dirGroup
}

func newDirGroup() *dirGroup {
	return &dirGroup{files: map[string]discoveredFile{}, subs: map[string]*dirGroup{}}
}

// buildChapterTree groups files by directory so book.yaml mirrors the project
// layout. A directory's README.md becomes that section's own file; the rest of
// its contents become its nested sections.
func buildChapterTree(files []discoveredFile) []chapterNode {
	root := newDirGroup()
	for _, f := range files {
		g := root
		segments := strings.Split(f.RelPath, "/")
		for _, dir := range segments[:len(segments)-1] {
			sub, ok := g.subs[dir]
			if !ok {
				sub = newDirGroup()
				g.subs[dir] = sub
			}
			g = sub
		}
		g.files[segments[len(segments)-1]] = f
	}
	return rootNodes(root)
}

// rootNodes renders the project root: the top-level README.md leads, then the
// remaining top-level files, then one section per subdirectory.
func rootNodes(g *dirGroup) []chapterNode {
	var nodes []chapterNode
	names := sortedFileNames(g)
	if readme := readmeName(g); readme != "" {
		nodes = append(nodes, chapterNode{
			Title: chapterTitle(g.files[readme], "Preface"),
			File:  g.files[readme].RelPath,
		})
		names = slices.DeleteFunc(names, func(n string) bool { return n == readme })
	}
	for _, name := range names {
		f := g.files[name]
		nodes = append(nodes, chapterNode{Title: chapterTitle(f, ""), File: f.RelPath})
	}
	return append(nodes, subdirNodes(g)...)
}

// subdirNodes renders each immediate subdirectory as one section.
func subdirNodes(g *dirGroup) []chapterNode {
	var nodes []chapterNode
	for _, name := range sortedSubNames(g) {
		nodes = append(nodes, dirNode(name, g.subs[name])...)
	}
	return nodes
}

// dirNode renders one directory as a section. The section needs a file of its
// own — a file-less chapter is not a shape the rest of mdpress understands —
// so it takes the directory's README.md, or its first file when there is none.
// A directory holding nothing but subdirectories contributes its children
// directly rather than an empty entry.
func dirNode(dirName string, g *dirGroup) []chapterNode {
	names := sortedFileNames(g)
	if len(names) == 0 {
		return subdirNodes(g)
	}
	entryName := readmeName(g)
	if entryName == "" {
		entryName = names[0]
	}
	entry := g.files[entryName]
	node := chapterNode{
		Title: chapterTitle(entry, config.TitleFromDirName(dirName)),
		File:  entry.RelPath,
	}
	for _, name := range names {
		if name == entryName {
			continue
		}
		f := g.files[name]
		node.Sections = append(node.Sections, chapterNode{Title: chapterTitle(f, ""), File: f.RelPath})
	}
	node.Sections = append(node.Sections, subdirNodes(g)...)
	return []chapterNode{node}
}

// chapterTitle prefers the file's own H1, then the caller's fallback, then a
// title derived from the path.
func chapterTitle(f discoveredFile, fallback string) string {
	if f.Title != "" {
		return f.Title
	}
	if fallback != "" {
		return fallback
	}
	return inferTitleFromPath(f.RelPath)
}

func readmeName(g *dirGroup) string {
	for name := range g.files {
		if strings.EqualFold(name, "README.md") {
			return name
		}
	}
	return ""
}

func sortedFileNames(g *dirGroup) []string {
	names := make([]string, 0, len(g.files))
	for name := range g.files {
		names = append(names, name)
	}
	slices.SortFunc(names, config.NaturalCompare)
	return names
}

func sortedSubNames(g *dirGroup) []string {
	names := make([]string, 0, len(g.subs))
	for name := range g.subs {
		names = append(names, name)
	}
	slices.SortFunc(names, config.NaturalCompare)
	return names
}

// writeChapterNodes renders the tree as YAML list entries at the given depth.
func writeChapterNodes(b *strings.Builder, nodes []chapterNode, depth int) {
	itemIndent := strings.Repeat(" ", 2+depth*4)
	fieldIndent := itemIndent + "  "
	for _, n := range nodes {
		b.WriteString(itemIndent + "- title: " + yamlQuote(n.Title) + "\n")
		b.WriteString(fieldIndent + "file: " + yamlQuote(n.File) + "\n")
		if len(n.Sections) > 0 {
			b.WriteString(fieldIndent + "sections:\n")
			writeChapterNodes(b, n.Sections, depth+1)
		}
	}
}

// yamlQuote renders a double-quoted YAML scalar.
func yamlQuote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
