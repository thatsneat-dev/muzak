package ui_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thatsneat-dev/muzak/internal/music"
	"github.com/thatsneat-dev/muzak/internal/ui"
)

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		seconds  float64
		expected string
	}{
		{"zero", 0, "00:00:00"},
		{"seconds only", 45, "00:00:45"},
		{"minutes and seconds", 125, "00:02:05"},
		{"one hour", 3600, "01:00:00"},
		{"hours minutes seconds", 3661, "01:01:01"},
		{"fractional truncates", 90.9, "00:01:30"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, ui.FormatDuration(tt.seconds))
		})
	}
}

func TestProgressBar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		position float64
		duration float64
		width    int
		expected string
	}{
		{"zero duration", 0, 0, 10, strings.Repeat("─", 10)},
		{"negative duration", 0, -1, 10, strings.Repeat("─", 10)},
		{"at start", 0, 100, 10, strings.Repeat("─", 10)},
		{"half way", 50, 100, 10, strings.Repeat("━", 5) + strings.Repeat("─", 5)},
		{"complete", 100, 100, 10, strings.Repeat("━", 10)},
		{"past end clamped", 120, 100, 10, strings.Repeat("━", 10)},
		{"quarter", 25, 100, 20, strings.Repeat("━", 5) + strings.Repeat("─", 15)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, ui.ProgressBar(tt.position, tt.duration, tt.width))
		})
	}
}

func TestControlsLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		playing bool
		flash   string
	}{
		{"paused no flash", false, ""},
		{"playing no flash", true, ""},
		{"flash prev", false, "prev"},
		{"flash play", true, "play"},
		{"flash next", false, "next"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ui.ControlsLine(tt.playing, tt.flash, false, "off")

			// Always contains all three icons.
			assert.Contains(t, result, ui.IconPrev)
			if tt.playing {
				assert.Contains(t, result, ui.IconPause)
			} else {
				assert.Contains(t, result, ui.IconPlay)
			}
			assert.Contains(t, result, ui.IconNext)

			// Flashed icon is wrapped in yellow ANSI.
			if tt.flash != "" {
				assert.Contains(t, result, ui.Yellow)
				assert.Contains(t, result, ui.Reset)
			}

			// Non-flashed state has no ANSI color codes.
			if tt.flash == "" {
				assert.NotContains(t, result, ui.Yellow)
			}
		})
	}
}

func TestBuildLines(t *testing.T) {
	t.Parallel()

	info := &music.TrackInfo{
		Name:     "Test Song",
		Artist:   "Test Artist",
		Album:    "Test Album",
		Duration: 240,
		Position: 60,
		Playing:  true,
	}

	layout := ui.ComputeLayout(80, 24, nil)
	lines := ui.BuildLines(info, layout, false, ui.NowPlayingOptions{Shuffle: false, Repeat: "off"})

	require.Len(t, lines, ui.NumTextLines)

	// Line 0: bold track name.
	assert.Contains(t, lines[0], info.Name)
	assert.Contains(t, lines[0], "\x1b[1m")

	// Line 1: artist - album.
	assert.Equal(t, "Test Artist - Test Album", lines[1])

	// Line 2: position / duration.
	assert.Equal(t, "00:01:00 / 00:04:00", lines[2])

	// Line 3: progress bar (non-empty).
	assert.NotEmpty(t, lines[3])

	// Line 4: controls with pause icon (playing=true).
	assert.Contains(t, lines[4], ui.IconPause)
}

func TestVisibleLen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"plain ascii", "hello", 5},
		{"empty", "", 0},
		{"ansi escape", "\x1b[1mbold\x1b[0m", 4},
		{"emoji takes 2 cells", "🎵", 2},
		{"mixed emoji and text", "hi 🎵 yo", 8},
		{"nerd font icon", ui.IconFolder, 1},
		{"ansi with emoji", "\x1b[33m🎵\x1b[0m", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, ui.VisibleLen(tt.input))
		})
	}
}

func TestTruncateVisibleWideChars(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		maxWidth int
		contains string
	}{
		{"ascii within limit", "hello", 10, "hello"},
		{"ascii truncated", "hello world", 5, "hell"},
		{"emoji at boundary", "ab🎵cd", 4, "ab"},
		{"zero width", "hello", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ui.TruncateVisible(tt.input, tt.maxWidth)
			assert.Contains(t, result, tt.contains)
			assert.LessOrEqual(t, ui.VisibleLen(result), tt.maxWidth)
		})
	}
}

func TestTruncateVisibleEllipsisFitsInMaxWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		maxWidth int
		expected int
	}{
		{"exact fit no truncation", "hello", 5, 5},
		{"truncated with ellipsis", "hello world", 8, 8},
		{"long string narrow width", "abcdefghij", 4, 4},
		{"width 1 truncation", "abcdef", 1, 1},
		{"ansi styled truncation", "\x1b[1mhello world\x1b[0m", 6, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ui.TruncateVisible(tt.input, tt.maxWidth)
			assert.Equal(t, tt.expected, ui.VisibleLen(result))
		})
	}
}

func TestDrawHintsSingleLine(t *testing.T) {
	t.Parallel()

	var lines []string
	draw := func(row, col int, s string) {
		lines = append(lines, s)
	}
	items := []string{"[j|k] move", "[enter] select", "[x] close"}
	ui.DrawHints(items, 60, 0, 0, draw)

	assert.Len(t, lines, 2) // content line + blank clear line
	assert.Contains(t, lines[0], "[j|k] move")
	assert.Contains(t, lines[0], "[enter] select")
	assert.Contains(t, lines[0], "[x] close")
}

func TestDrawHintsTwoLines(t *testing.T) {
	t.Parallel()

	var lines []string
	draw := func(row, col int, s string) {
		lines = append(lines, s)
	}
	items := []string{"[j|k] move", "[/] search", "[a|d] sort", "[enter] select", "[h] back", "[x] close"}
	ui.DrawHints(items, 40, 0, 0, draw)

	assert.Len(t, lines, 2)
	// First line should NOT contain all items.
	assert.NotContains(t, lines[0], "[x] close")
	// Second line should contain the overflow items.
	assert.Contains(t, lines[1], "[h] back")
	assert.Contains(t, lines[1], "[x] close")
	// Lines should not exceed modalWidth visible chars.
	for _, l := range lines {
		assert.LessOrEqual(t, ui.VisibleLen(l), 40)
	}
}
