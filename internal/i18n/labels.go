package i18n

import (
	"fmt"
	"strings"
)

// CrossRefLabels holds the wording used for figure, table and section
// cross-references and for auto-generated captions. Keeping it here rather
// than in the cross-reference resolver means an English book no longer prints
// "图 1" under its figures.
type CrossRefLabels struct {
	Figure  string // Format string taking the figure number, e.g. "Figure %d".
	Table   string // Format string taking the table number.
	Section string // Format string taking the section number.
}

// FigureLabel renders the label for figure n, e.g. "Figure 1".
func (l CrossRefLabels) FigureLabel(n int) string { return fmt.Sprintf(l.Figure, n) }

// TableLabel renders the label for table n, e.g. "Table 1".
func (l CrossRefLabels) TableLabel(n int) string { return fmt.Sprintf(l.Table, n) }

// SectionLabel renders the label for section n, e.g. "Section 1".
func (l CrossRefLabels) SectionLabel(n int) string { return fmt.Sprintf(l.Section, n) }

var (
	englishCrossRefLabels = CrossRefLabels{Figure: "Figure %d", Table: "Table %d", Section: "Section %d"}
	chineseCrossRefLabels = CrossRefLabels{Figure: "图%d", Table: "表%d", Section: "第%d节"}
)

// CrossRefLabelsFor returns the labels for a book language code such as
// "en-US" or "zh-CN". Unknown languages fall back to English, matching the
// default language in book.yaml.
func CrossRefLabelsFor(lang string) CrossRefLabels {
	base, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(lang)), "-")
	switch base {
	case "zh":
		return chineseCrossRefLabels
	default:
		return englishCrossRefLabels
	}
}
