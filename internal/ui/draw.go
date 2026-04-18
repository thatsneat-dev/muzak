package ui

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/thatsneat-dev/muzak/internal/music"
)

// LineDrawer writes a single line at the given row/col.
type LineDrawer func(row, col int, s string)

// NewLineDrawer returns a LineDrawer that uses ANSI cursor save/restore.
func NewLineDrawer(w io.Writer) LineDrawer {
	return func(row, col int, s string) {
		_, _ = fmt.Fprint(w, "\x1b8")
		if row > 0 {
			_, _ = fmt.Fprintf(w, "\x1b[%dB", row)
		}
		if col > 0 {
			_, _ = fmt.Fprintf(w, "\x1b[%dC", col)
		}
		_, _ = fmt.Fprintf(w, "\x1b[K%s", s)
	}
}

// DrawText writes all text lines into the display area, positioned beside
// artwork (if present) and vertically centered.
func DrawText(w io.Writer, layout Layout, lines []string, withArtwork bool) {
	startRow := PadTop + (layout.ArtworkRows-len(lines))/2
	colOffset := layout.TextCol(withArtwork)
	maxWidth := layout.TextWidth(withArtwork)

	for i, line := range lines {
		_, _ = fmt.Fprint(w, "\x1b8")
		row := startRow + i
		if row > 0 {
			_, _ = fmt.Fprintf(w, "\x1b[%dB", row)
		}
		if colOffset > 0 {
			_, _ = fmt.Fprintf(w, "\x1b[%dC", colOffset)
		}
		if maxWidth > 0 {
			line = TruncateVisible(line, maxWidth)
		}
		_, _ = fmt.Fprintf(w, "\x1b[K%s", line)
	}
}

// UpdateDynamicLines overwrites the position, progress bar, and controls lines in place.
func UpdateDynamicLines(w io.Writer, layout Layout, lines []string, withArtwork bool) {
	startRow := PadTop + (layout.ArtworkRows-len(lines))/2
	colOffset := layout.TextCol(withArtwork)
	maxWidth := layout.TextWidth(withArtwork)

	for i := 2; i < len(lines); i++ {
		_, _ = fmt.Fprint(w, "\x1b8")
		row := startRow + i
		if row > 0 {
			_, _ = fmt.Fprintf(w, "\x1b[%dB", row)
		}
		if colOffset > 0 {
			_, _ = fmt.Fprintf(w, "\x1b[%dC", colOffset)
		}
		line := lines[i]
		if maxWidth > 0 {
			line = TruncateVisible(line, maxWidth)
		}
		_, _ = fmt.Fprintf(w, "\x1b[K%s", line)
	}
}

// NowPlayingOptions holds the display options for now-playing lines.
type NowPlayingOptions struct {
	Flash   string
	Shuffle bool
	Repeat  string
}

// BuildLines constructs the five display lines for the current track.
func BuildLines(info *music.TrackInfo, layout Layout, withArtwork bool, opts NowPlayingOptions) []string {
	bw := layout.BarWidth(withArtwork)
	return []string{
		"\x1b[1m" + info.Name + "\x1b[0m",
		info.Artist + " - " + info.Album,
		FormatDuration(info.Position) + " / " + FormatDuration(info.Duration),
		ProgressBar(info.Position, info.Duration, bw),
		ControlsLine(info.Playing, opts.Flash, opts.Shuffle, opts.Repeat),
	}
}

// ControlsLine renders the playback control icons with shuffle/repeat state.
func ControlsLine(playing bool, flash string, shuffle bool, repeat string) string {
	shuf := IconShuffle
	if shuffle {
		shuf = Yellow + IconShuffle + Reset
	}

	prev := IconPrev
	if flash == "prev" {
		prev = Yellow + prev + Reset
	}

	pp := IconPlay
	if playing {
		pp = IconPause
	}
	if flash == "play" {
		pp = Yellow + pp + Reset
	}

	next := IconNext
	if flash == "next" {
		next = Yellow + next + Reset
	}

	var rep string
	switch repeat {
	case "one":
		rep = Yellow + IconRepeatOne + Reset
	case "all":
		rep = Yellow + IconRepeat + Reset
	default:
		rep = IconRepeat
	}

	return shuf + "  " + prev + "  " + pp + "  " + next + "  " + rep
}

// DrawVolumeBar renders a vertical volume indicator to the left of the artwork.
func DrawVolumeBar(w io.Writer, layout Layout, vol float32, flash string) {
	barRows := layout.ArtworkRows - 2
	filled := int(vol*float32(barRows) + 0.5)
	if filled > barRows {
		filled = barRows
	}
	empty := barRows - filled

	for i := range layout.ArtworkRows {
		_, _ = fmt.Fprint(w, "\x1b8")
		row := PadTop + i
		if row > 0 {
			_, _ = fmt.Fprintf(w, "\x1b[%dB", row)
		}
		_, _ = fmt.Fprintf(w, "\x1b[%dC", PadLeft)
		switch {
		case i == 0:
			icon := IconVolHigh
			if flash == "volup" {
				icon = Yellow + icon + Reset
			}
			_, _ = fmt.Fprint(w, icon)
		case i == layout.ArtworkRows-1:
			icon := IconVolOff
			if flash == "voldn" {
				icon = Yellow + icon + Reset
			}
			_, _ = fmt.Fprint(w, icon)
		case i-1 < empty:
			_, _ = fmt.Fprint(w, "░")
		default:
			_, _ = fmt.Fprint(w, "█")
		}
	}
}

const chunkSize = 4096

// SendKittyImage transmits PNG data inline using the Kitty graphics protocol
// with chunked encoding for payloads exceeding 4096 base64 bytes.
func SendKittyImage(w *os.File, layout Layout, pngData []byte) {
	b64 := base64.StdEncoding.EncodeToString(pngData)

	if len(b64) <= chunkSize {
		_, _ = fmt.Fprintf(w, "\x1b_Ga=T,f=100,c=%d,r=%d,q=2;%s\x1b\\",
			layout.ArtworkCols, layout.ArtworkRows, b64)
		return
	}

	_, _ = fmt.Fprintf(w, "\x1b_Ga=T,f=100,c=%d,r=%d,q=2,m=1;%s\x1b\\",
		layout.ArtworkCols, layout.ArtworkRows, b64[:chunkSize])
	b64 = b64[chunkSize:]

	for len(b64) > chunkSize {
		_, _ = fmt.Fprintf(w, "\x1b_Gm=1;%s\x1b\\", b64[:chunkSize])
		b64 = b64[chunkSize:]
	}

	_, _ = fmt.Fprintf(w, "\x1b_Gm=0;%s\x1b\\", b64)
}

// QueueViewModel holds the state needed to render the queue modal.
type QueueViewModel struct {
	Tracks       []music.QueueTrack
	Scroll       int
	Loading      bool
	SpinnerFrame int
	VisibleLimit int
}

// DrawQueue renders the queue as a bordered modal.
func DrawQueue(layout Layout, hasArtwork bool, q QueueViewModel, drawLine LineDrawer) {
	modalCol := PadLeft
	if hasArtwork {
		modalCol = layout.ArtworkLeft() + layout.ArtworkCols + 2
	}
	modalWidth := 40
	if layout.TerminalCols > modalCol+2 {
		modalWidth = layout.TerminalCols - modalCol - 2
	}
	innerWidth := modalWidth - 4

	// Title bar.
	title := "Up Next"
	titleLen := len(title)
	pad := modalWidth - 2 - titleLen - 2
	if pad < 0 {
		pad = 0
	}
	drawLine(PadTop, modalCol,
		"┌ \x1b[1m"+title+"\x1b[0m "+strings.Repeat("─", pad)+"┐")

	// Content rows.
	rows := q.VisibleLimit
	if rows > layout.ArtworkRows-2 {
		rows = layout.ArtworkRows - 2
		if rows < 1 {
			rows = 1
		}
	}

	spinnerChars := [4]string{"⠋", "⠙", "⠹", "⠸"}

	for i := range rows {
		idx := q.Scroll + i
		var content string
		if q.Loading && i == 0 {
			content = spinnerChars[q.SpinnerFrame%len(spinnerChars)] + " Loading…"
		} else if !q.Loading && len(q.Tracks) == 0 && i == 0 {
			content = "No upcoming tracks."
		} else if idx < len(q.Tracks) {
			t := q.Tracks[idx]
			content = fmt.Sprintf("%2d. %s - %s", idx+1, t.Artist, t.Name)
			content = TruncateVisible(content, innerWidth)
		}
		visible := VisibleLen(content)
		trailing := innerWidth - visible
		if trailing < 0 {
			trailing = 0
		}
		drawLine(PadTop+1+i, modalCol,
			"│ "+content+strings.Repeat(" ", trailing)+" │")
	}

	// Bottom border with scroll position.
	bottom := strings.Repeat("─", modalWidth-2)
	if !q.Loading && len(q.Tracks) > 0 {
		pos := fmt.Sprintf(" %d-%d/%d ", q.Scroll+1,
			min(q.Scroll+rows, len(q.Tracks)), len(q.Tracks))
		if len(pos) < modalWidth-2 {
			bottom = strings.Repeat("─", modalWidth-2-len(pos)) + pos
		}
	}
	drawLine(PadTop+1+rows, modalCol,
		"└"+bottom+"┘")
}

// AllocateSpace clears the screen and reserves space for the display area.
func AllocateSpace(w io.Writer, layout Layout) {
	_, _ = fmt.Fprint(w, "\x1b[2J\x1b[3J\x1b[H")
	for range layout.DisplayRows() {
		_, _ = fmt.Fprint(w, "\r\n")
	}
	_, _ = fmt.Fprintf(w, "\x1b[%dA", layout.DisplayRows())
	_, _ = fmt.Fprint(w, "\x1b7")
}

// ClearDisplay clears all rows in the display area and removes any Kitty images.
func ClearDisplay(w io.Writer, layout Layout) {
	_, _ = fmt.Fprint(w, "\x1b8")
	_, _ = fmt.Fprint(w, "\x1b_Ga=d,d=a\x1b\\")
	for range layout.DisplayRows() {
		_, _ = fmt.Fprint(w, "\x1b[2K\x1b[1B")
	}
	_, _ = fmt.Fprint(w, "\x1b8")
}

// SafeReset stops the timer, drains any pending event, and resets it.
func SafeReset(t *time.Timer, d time.Duration) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(d)
}

// DrawHints renders hint items below the modal, wrapping to two lines if needed.
func DrawHints(items []string, modalWidth, row, col int, drawLine LineDrawer) {
	sep := "  "
	joined := strings.Join(items, sep)
	if len(joined) <= modalWidth {
		pad := modalWidth - len(joined)
		left := pad / 2
		line := "\x1b[2m" + strings.Repeat(" ", left) + joined + strings.Repeat(" ", pad-left) + "\x1b[0m"
		drawLine(row, col, line)
		drawLine(row+1, col, strings.Repeat(" ", modalWidth))
		return
	}

	// Split items across two lines, fitting as many as possible per line.
	var rows [2][]string
	widths := [2]int{}
	r := 0
	for _, item := range items {
		itemLen := len(item)
		needed := itemLen
		if widths[r] > 0 {
			needed += len(sep)
		}
		if widths[r]+needed > modalWidth && widths[r] > 0 {
			r++
			if r > 1 {
				break
			}
			needed = itemLen
		}
		rows[r] = append(rows[r], item)
		widths[r] += needed
	}

	for li, parts := range rows {
		text := strings.Join(parts, sep)
		if len(text) > modalWidth {
			text = text[:modalWidth]
		}
		pad := modalWidth - len(text)
		left := pad / 2
		line := "\x1b[2m" + strings.Repeat(" ", left) + text + strings.Repeat(" ", pad-left) + "\x1b[0m"
		drawLine(row+li, col, line)
	}
}
