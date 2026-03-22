package cmd

import (
	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/internal/renderer"
	"github.com/yeasy/mdpress/internal/toc"
)

type flattenedChapter struct {
	Def   config.ChapterDef
	Depth int
}

type navHeading struct {
	Title    string
	ID       string
	Children []navHeading
}

func flattenChaptersWithDepth(chapters []config.ChapterDef) []flattenedChapter {
	var result []flattenedChapter

	var walk func([]config.ChapterDef, int)
	walk = func(items []config.ChapterDef, depth int) {
		for _, ch := range items {
			result = append(result, flattenedChapter{Def: ch, Depth: depth})
			if len(ch.Sections) > 0 {
				walk(ch.Sections, depth+1)
			}
		}
	}

	walk(chapters, 0)
	return result
}

func buildHeadingTree(headings []markdown.HeadingInfo, chapterID string) []navHeading {
	if len(headings) == 0 {
		return nil
	}

	tocHeadings := make([]toc.HeadingInfo, 0, len(headings))
	for _, h := range headings {
		tocHeadings = append(tocHeadings, toc.HeadingInfo{
			Level: h.Level,
			Text:  h.Text,
			ID:    h.ID,
		})
	}

	entries := toc.NewGenerator().Generate(tocHeadings)
	if len(entries) == 0 {
		return nil
	}

	// The first heading is usually the chapter root, so avoid repeating it in the sidebar.
	if entries[0].ID == chapterID {
		trimmed := make([]toc.TOCEntry, 0, len(entries[0].Children)+len(entries)-1)
		trimmed = append(trimmed, entries[0].Children...)
		if len(entries) > 1 {
			trimmed = append(trimmed, entries[1:]...)
		}
		entries = trimmed

		// If stripping removed all entries, return nil
		if len(entries) == 0 {
			return nil
		}
	}

	result := toNavHeadings(entries)
	return result
}

func toNavHeadings(entries []toc.TOCEntry) []navHeading {
	result := make([]navHeading, 0, len(entries))
	for _, entry := range entries {
		result = append(result, navHeading{
			Title:    entry.Title,
			ID:       entry.ID,
			Children: toNavHeadings(entry.Children),
		})
	}
	return result
}

func toRendererNavHeadings(items []navHeading) []renderer.NavHeading {
	result := make([]renderer.NavHeading, 0, len(items))
	for _, item := range items {
		result = append(result, renderer.NavHeading{
			Title:    item.Title,
			ID:       item.ID,
			Children: toRendererNavHeadings(item.Children),
		})
	}
	return result
}
