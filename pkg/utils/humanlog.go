// humanlog.go renders log records the way the rest of the terminal UI speaks.
//
// The default logger used slog's key=value TextHandler at every verbosity, so
// the first warning a user ever saw was a machine line — spliced mid-line into
// the styled step progress, timestamps and quoting included. mdbook prints
// "warning: …", Docusaurus "[WARNING] …"; next to them the raw handler read
// like debug output that escaped into production. The TextHandler remains the
// right tool under --verbose, where the audience is a person debugging mdpress
// itself; this handler serves everyone else.
package utils

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
)

// HumanLogHandler formats records as single styled lines:
//
//	⚠ intro.md: Markdown link target is outside the build graph: ./nothere.md
//
// A record's attributes are folded into the sentence rather than appended as
// key=value pairs: one attribute names the subject (file, path, src, …), one
// carries the substance (detail, error, hint), and whatever remains prints as
// a short parenthetical. It also coordinates with the step progress so a
// warning never lands in the middle of a pending "[2/5] … ..." line.
type HumanLogHandler struct {
	mu    *sync.Mutex
	out   *os.File
	level slog.Leveler
	attrs []slog.Attr
}

// NewHumanLogHandler returns a handler writing styled lines to out.
func NewHumanLogHandler(out *os.File, level slog.Leveler) *HumanLogHandler {
	return &HumanLogHandler{mu: &sync.Mutex{}, out: out, level: level}
}

// Enabled implements slog.Handler.
func (h *HumanLogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

// subjectKeys name the record's subject, in preference order; bodyKeys carry
// its substance. Everything else becomes the parenthetical.
var subjectKeys = []string{"file", "path", "src", "key", "config", "trigger", "image", "setting"}
var bodyKeys = []string{"detail", "error", "hint", "reason"}

// Handle implements slog.Handler.
func (h *HumanLogHandler) Handle(_ context.Context, r slog.Record) error {
	kv := map[string]string{}
	order := []string{}
	add := func(a slog.Attr) {
		if _, seen := kv[a.Key]; !seen {
			order = append(order, a.Key)
		}
		kv[a.Key] = a.Value.String()
	}
	for _, a := range h.attrs {
		add(a)
	}
	r.Attrs(func(a slog.Attr) bool { add(a); return true })

	take := func(keys []string) string {
		for _, k := range keys {
			if v, ok := kv[k]; ok && v != "" {
				delete(kv, k)
				return v
			}
		}
		return ""
	}
	subject := take(subjectKeys)
	body := take(bodyKeys)

	var line strings.Builder
	line.WriteString("  ")
	mark, color := "⚠", colorYellow
	if r.Level >= slog.LevelError {
		mark, color = "✗", colorRed
	}
	if colorEnabled.Load() {
		line.WriteString(color + mark + colorReset + " ")
	} else {
		line.WriteString(mark + " ")
	}
	if subject != "" {
		line.WriteString(subject + ": ")
	}
	line.WriteString(r.Message)
	if body != "" {
		line.WriteString(": " + body)
	}
	var rest []string
	for _, k := range order {
		if v, ok := kv[k]; ok {
			rest = append(rest, k+": "+v)
		}
	}
	if len(rest) > 0 {
		line.WriteString(" (" + strings.Join(rest, ", ") + ")")
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	// Break out of a pending progress line first; the tracker reprints the
	// whole step line when it completes, so nothing is lost.
	// A terminal write failing has nowhere better to be reported than the
	// caller, and slog.Handler lets errors propagate — so they do.
	if InterruptPendingLine() {
		if _, err := fmt.Fprintln(h.out); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(h.out, line.String())
	return err
}

// WithAttrs implements slog.Handler.
func (h *HumanLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	c := *h
	c.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &c
}

// WithGroup implements slog.Handler. mdpress does not log groups; the name is
// dropped rather than modeled.
func (h *HumanLogHandler) WithGroup(string) slog.Handler { return h }
