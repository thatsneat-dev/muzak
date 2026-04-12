// Command muzak displays the currently playing Apple Music track info
// alongside album artwork rendered via the Kitty graphics protocol
// (supported by Kitty, Ghostty, WezTerm, and other terminals).
// It runs continuously, updating the playback position every second
// and refreshing artwork when a new track begins.
package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/thatsneat-dev/muzak/internal/music"
	"github.com/thatsneat-dev/muzak/internal/volume"
	"golang.org/x/term"
)

const (
	artworkCols = 20
	artworkRows = 10
	barWidth    = 30

	padTop  = 1 // blank lines above artwork
	padLeft = 2 // columns from left edge

	volBarWidth = 1                                 // column width of volume bar
	volBarGap   = 2                                 // gap between volume bar and artwork
	artworkLeft = padLeft + volBarWidth + volBarGap // artwork start column

	displayRows = padTop + artworkRows // total rows reserved

	numTextLines = 5 // track name, artist/album, position, progress bar, controls

	// Nerd Font icons.
	iconPrev  = "\U000F04AE" // 󰒮 nf-md-skip_previous
	iconPlay  = "\U000F040A" // 󰐊 nf-md-play
	iconPause = "\U000F03E4" // 󰏤 nf-md-pause
	iconNext  = "\U000F04AD" // 󰒭 nf-md-skip_next

	iconVolHigh = "\U000F057E" // 󰕾 nf-md-volume_high
	iconVolOff  = "\U000F0581" // 󰖁 nf-md-volume_off

	restartThreshold = 3.0 // seconds before 'h' restarts instead of going previous
)

var out = os.Stdout

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Put terminal into raw mode for single-keypress input.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error setting raw mode: %v\n", err)
		os.Exit(1)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	tmp, err := os.CreateTemp("", "tty-graphics-protocol-*.png")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	artworkPath := tmp.Name()
	tmp.Close()
	defer os.Remove(artworkPath)

	// Hide cursor while running.
	fmt.Fprint(out, "\x1b[?25l")
	defer fmt.Fprint(out, "\x1b[?25h")

	// Read keypresses in the background.
	keys := make(chan byte, 10)
	go func() {
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				return
			}
			keys <- buf[0]
		}
	}()

	var (
		currentTrack   string
		hasArtwork     bool
		spaceAllocated bool
		latestInfo     *music.TrackInfo
		pollRunning    bool
		flashIcon      string // "prev", "play", "next", or ""
	)

	type pollResult struct {
		info *music.TrackInfo
		err  error
	}
	pollCh := make(chan pollResult, 1)

	allocateSpace := func() {
		for range displayRows {
			fmt.Fprint(out, "\r\n")
		}
		fmt.Fprintf(out, "\x1b[%dA", displayRows)
		fmt.Fprint(out, "\x1b7")
		spaceAllocated = true
	}

	clearDisplay := func() {
		fmt.Fprint(out, "\x1b8")
		fmt.Fprint(out, "\x1b_Ga=d,d=a\x1b\\")
		for range displayRows {
			fmt.Fprint(out, "\x1b[2K\x1b[1B")
		}
		fmt.Fprint(out, "\x1b8")
	}

	// startPoll kicks off a NowPlaying query in the background.
	// Only one poll runs at a time.
	startPoll := func() {
		if pollRunning {
			return
		}
		os.Truncate(artworkPath, 0)
		pollRunning = true
		go func() {
			info, err := music.NowPlaying(ctx, artworkPath)
			select {
			case pollCh <- pollResult{info, err}:
			case <-ctx.Done():
			}
		}()
	}

	// handlePoll processes an async poll result and updates the display.
	handlePoll := func(info *music.TrackInfo) {
		latestInfo = info

		if info == nil {
			if currentTrack != "" {
				clearDisplay()
				drawText([]string{"No track is currently playing."}, false)
				currentTrack = ""
				hasArtwork = false
			} else if !spaceAllocated {
				allocateSpace()
				drawText([]string{"No track is currently playing."}, false)
			}
			return
		}

		trackKey := info.Artist + "\x00" + info.Album + "\x00" + info.Name
		lines := buildLines(info, flashIcon)

		if trackKey != currentTrack {
			if !spaceAllocated {
				allocateSpace()
			} else {
				clearDisplay()
			}

			fi, _ := os.Stat(artworkPath)
			hasArtwork = fi != nil && fi.Size() > 0

			if hasArtwork {
				if pngData, err := os.ReadFile(artworkPath); err == nil && len(pngData) > 0 {
					fmt.Fprintf(out, "\x1b8\x1b[%dB\x1b[%dC", padTop, artworkLeft)
					sendKittyImage(out, pngData)
				}
			}

			drawText(lines, hasArtwork)
			if hasArtwork {
				drawVolumeBar(flashIcon)
			}
			currentTrack = trackKey
		} else {
			updateDynamicLines(lines, hasArtwork)
			if hasArtwork {
				drawVolumeBar(flashIcon)
			}
		}
	}

	startPoll()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	// refresh fires shortly after a control action so the display
	// updates without blocking key input.
	refresh := time.NewTimer(0)
	if !refresh.Stop() {
		<-refresh.C
	}

	// flash timer clears the highlighted control icon after 500ms.
	flashTimer := time.NewTimer(0)
	if !flashTimer.Stop() {
		<-flashTimer.C
	}

	// redrawControls overwrites just the controls line with current flash state.
	redrawControls := func() {
		if latestInfo == nil {
			return
		}
		startRow := padTop + (artworkRows-numTextLines)/2
		controlRow := startRow + numTextLines - 1
		colOffset := padLeft
		if hasArtwork {
			colOffset = artworkLeft + artworkCols + 2
		}
		fmt.Fprint(out, "\x1b8")
		if controlRow > 0 {
			fmt.Fprintf(out, "\x1b[%dB", controlRow)
		}
		if colOffset > 0 {
			fmt.Fprintf(out, "\x1b[%dC", colOffset)
		}
		fmt.Fprintf(out, "\x1b[K%s", controlsLine(latestInfo.Playing, flashIcon))
	}

	for {
		select {
		case <-ctx.Done():
			if spaceAllocated {
				fmt.Fprint(out, "\x1b8")
				fmt.Fprintf(out, "\x1b[%dB\r\n", displayRows)
			}
			return
		case result := <-pollCh:
			pollRunning = false
			if result.err != nil {
				fmt.Fprintf(os.Stderr, "muzak: poll: %v\r\n", result.err)
			} else {
				handlePoll(result.info)
			}
		case key := <-keys:
			switch key {
			case ' ':
				go music.PlayPause(ctx)
				if latestInfo != nil {
					latestInfo.Playing = !latestInfo.Playing
				}
				flashIcon = "play"
				redrawControls()
				flashTimer.Reset(500 * time.Millisecond)
			case 'l':
				go music.NextTrack(ctx)
				flashIcon = "next"
				redrawControls()
				flashTimer.Reset(500 * time.Millisecond)
			case 'h':
				if latestInfo != nil && latestInfo.Position > restartThreshold {
					go music.RestartTrack(ctx)
				} else {
					go music.PreviousTrack(ctx)
				}
				flashIcon = "prev"
				redrawControls()
				flashTimer.Reset(500 * time.Millisecond)
			case 'j':
				_ = volume.Down()
				flashIcon = "voldn"
				if hasArtwork {
					drawVolumeBar(flashIcon)
				}
				flashTimer.Reset(500 * time.Millisecond)
			case 'k':
				_ = volume.Up()
				flashIcon = "volup"
				if hasArtwork {
					drawVolumeBar(flashIcon)
				}
				flashTimer.Reset(500 * time.Millisecond)
			case 'q', 3: // 'q' or Ctrl+C
				if spaceAllocated {
					fmt.Fprint(out, "\x1b8")
					fmt.Fprintf(out, "\x1b[%dB\r\n", displayRows)
				}
				return
			}
			// Schedule a refresh to pick up the new state.
			refresh.Reset(300 * time.Millisecond)
		case <-flashTimer.C:
			wasVol := flashIcon == "volup" || flashIcon == "voldn"
			flashIcon = ""
			if wasVol && hasArtwork {
				drawVolumeBar(flashIcon)
			} else {
				redrawControls()
			}
		case <-refresh.C:
			startPoll()
		case <-ticker.C:
			startPoll()
		}
	}
}

// buildLines constructs the five display lines for the current track.
func buildLines(info *music.TrackInfo, flash string) []string {
	return []string{
		"\x1b[1m" + info.Name + "\x1b[0m",
		info.Artist + " - " + info.Album,
		formatDuration(info.Position) + " / " + formatDuration(info.Duration),
		progressBar(info.Position, info.Duration, barWidth),
		controlsLine(info.Playing, flash),
	}
}

const (
	yellow = "\x1b[33m"
	reset  = "\x1b[0m"
)

// controlsLine renders the playback control icons, highlighting the active one during a flash.
func controlsLine(playing bool, flash string) string {
	prev := iconPrev
	if flash == "prev" {
		prev = yellow + prev + reset
	}

	pp := iconPlay
	if playing {
		pp = iconPause
	}
	if flash == "play" {
		pp = yellow + pp + reset
	}

	next := iconNext
	if flash == "next" {
		next = yellow + next + reset
	}

	return prev + "  " + pp + "  " + next
}

// progressBar returns a text-based progress bar of the given width using filled and empty segments.
func progressBar(position, duration float64, width int) string {
	if duration <= 0 {
		return strings.Repeat("─", width)
	}
	filled := int(position / duration * float64(width))
	if filled > width {
		filled = width
	}
	return strings.Repeat("━", filled) + strings.Repeat("─", width-filled)
}

// formatDuration converts seconds to HH:MM:SS format.
func formatDuration(seconds float64) string {
	total := int(seconds)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// drawText writes all text lines into the display area, positioned beside
// artwork (if present) and vertically centered.
func drawText(lines []string, withArtwork bool) {
	startRow := padTop + (artworkRows-len(lines))/2
	colOffset := padLeft
	if withArtwork {
		colOffset = artworkLeft + artworkCols + 2
	}

	for i, line := range lines {
		fmt.Fprint(out, "\x1b8")
		row := startRow + i
		if row > 0 {
			fmt.Fprintf(out, "\x1b[%dB", row)
		}
		if colOffset > 0 {
			fmt.Fprintf(out, "\x1b[%dC", colOffset)
		}
		fmt.Fprintf(out, "\x1b[K%s", line)
	}
}

// updateDynamicLines overwrites the position, progress bar, and controls lines in place.
func updateDynamicLines(lines []string, withArtwork bool) {
	startRow := padTop + (artworkRows-len(lines))/2
	colOffset := padLeft
	if withArtwork {
		colOffset = artworkLeft + artworkCols + 2
	}

	for i := 2; i < len(lines); i++ {
		fmt.Fprint(out, "\x1b8")
		row := startRow + i
		if row > 0 {
			fmt.Fprintf(out, "\x1b[%dB", row)
		}
		if colOffset > 0 {
			fmt.Fprintf(out, "\x1b[%dC", colOffset)
		}
		fmt.Fprintf(out, "\x1b[K%s", lines[i])
	}
}

// drawVolumeBar renders a vertical volume indicator to the left of the artwork,
// with a high icon at the top and an off icon at the bottom.
func drawVolumeBar(flash string) {
	vol, _ := volume.Get()
	barRows := artworkRows - 2 // reserve top and bottom for icons
	filled := int(vol*float32(barRows) + 0.5)
	if filled > barRows {
		filled = barRows
	}
	empty := barRows - filled

	for i := range artworkRows {
		fmt.Fprint(out, "\x1b8")
		row := padTop + i
		if row > 0 {
			fmt.Fprintf(out, "\x1b[%dB", row)
		}
		fmt.Fprintf(out, "\x1b[%dC", padLeft)
		switch {
		case i == 0:
			icon := iconVolHigh
			if flash == "volup" {
				icon = yellow + icon + reset
			}
			fmt.Fprint(out, icon)
		case i == artworkRows-1:
			icon := iconVolOff
			if flash == "voldn" {
				icon = yellow + icon + reset
			}
			fmt.Fprint(out, icon)
		case i-1 < empty:
			fmt.Fprint(out, "░")
		default:
			fmt.Fprint(out, "█")
		}
	}
}

const chunkSize = 4096

// sendKittyImage transmits PNG data inline using the Kitty graphics protocol
// with chunked encoding for payloads exceeding 4096 base64 bytes.
func sendKittyImage(out *os.File, pngData []byte) {
	b64 := base64.StdEncoding.EncodeToString(pngData)

	if len(b64) <= chunkSize {
		fmt.Fprintf(out, "\x1b_Ga=T,f=100,c=%d,r=%d,q=2;%s\x1b\\",
			artworkCols, artworkRows, b64)
		return
	}

	fmt.Fprintf(out, "\x1b_Ga=T,f=100,c=%d,r=%d,q=2,m=1;%s\x1b\\",
		artworkCols, artworkRows, b64[:chunkSize])
	b64 = b64[chunkSize:]

	for len(b64) > chunkSize {
		fmt.Fprintf(out, "\x1b_Gm=1;%s\x1b\\", b64[:chunkSize])
		b64 = b64[chunkSize:]
	}

	fmt.Fprintf(out, "\x1b_Gm=0;%s\x1b\\", b64)
}
