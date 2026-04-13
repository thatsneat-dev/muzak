package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thatsneat-dev/muzak/internal/music"
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
			assert.Equal(t, tt.expected, formatDuration(tt.seconds))
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
			assert.Equal(t, tt.expected, progressBar(tt.position, tt.duration, tt.width))
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
			result := controlsLine(tt.playing, tt.flash, false, "off")

			// Always contains all three icons.
			assert.Contains(t, result, iconPrev)
			if tt.playing {
				assert.Contains(t, result, iconPause)
			} else {
				assert.Contains(t, result, iconPlay)
			}
			assert.Contains(t, result, iconNext)

			// Flashed icon is wrapped in yellow ANSI.
			if tt.flash != "" {
				assert.Contains(t, result, yellow)
				assert.Contains(t, result, reset)
			}

			// Non-flashed state has no ANSI color codes.
			if tt.flash == "" {
				assert.NotContains(t, result, yellow)
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

	lines := buildLines(info, "", false, false, "off")

	require.Len(t, lines, numTextLines)

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
	assert.Contains(t, lines[4], iconPause)
}
