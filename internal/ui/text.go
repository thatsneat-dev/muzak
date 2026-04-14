package ui

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
)

// TruncateVisible truncates s so that its visible (non-ANSI-escape) terminal
// cell width does not exceed maxWidth, correctly handling wide characters.
// When truncated, an ellipsis (…) is appended and fits within maxWidth.
// Any open SGR styling is reset with \x1b[0m.
func TruncateVisible(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	var (
		width   int
		inEsc   bool
		ellCut  int  // byte index where visible width <= maxWidth-1
		hasMore bool // whether there are visible chars beyond maxWidth
	)
	for i, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEsc = false
			}
			continue
		}
		rw := runewidth.RuneWidth(r)
		if width+rw > maxWidth {
			hasMore = true
			break
		}
		width += rw
		if width <= maxWidth-1 {
			ellCut = i + len(string(r))
		}
	}
	if !hasMore {
		return s
	}
	if ellCut > 0 {
		return s[:ellCut] + "\x1b[0m…"
	}
	return "…"
}

// VisibleLen counts the visible terminal cell width of a string,
// correctly handling ANSI escapes and wide characters (emojis).
func VisibleLen(s string) int {
	var width int
	var inEsc bool
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEsc = false
			}
			continue
		}
		width += runewidth.RuneWidth(r)
	}
	return width
}

// FormatDuration converts seconds to HH:MM:SS format.
func FormatDuration(seconds float64) string {
	total := int(seconds)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// ProgressBar returns a text-based progress bar of the given width using filled and empty segments.
func ProgressBar(position, duration float64, width int) string {
	if duration <= 0 {
		return strings.Repeat("─", width)
	}
	filled := int(position / duration * float64(width))
	if filled > width {
		filled = width
	}
	return strings.Repeat("━", filled) + strings.Repeat("─", width-filled)
}
