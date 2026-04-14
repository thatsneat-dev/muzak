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

	"github.com/mattn/go-runewidth"
	"github.com/thatsneat-dev/muzak/internal/music"
	"github.com/thatsneat-dev/muzak/internal/volume"
	"golang.org/x/term"
)

const (
	defaultArtworkCols = 20
	defaultArtworkRows = 10
	barWidth           = 30

	padTop  = 1 // blank lines above artwork
	padLeft = 2 // columns from left edge

	volBarWidth = 1                                // column width of volume bar
	volBarGap   = 2                                // gap between volume bar and artwork
	artworkLeft = padLeft + volBarWidth + volBarGap // artwork start column

	numTextLines = 5 // track name, artist/album, position, progress bar, controls

	// Nerd Font icons.
	iconPrev  = "\U000F04AE" // 󰒮 nf-md-skip_previous
	iconPlay  = "\U000F040A" // 󰐊 nf-md-play
	iconPause = "\U000F03E4" // 󰏤 nf-md-pause
	iconNext  = "\U000F04AD" // 󰒭 nf-md-skip_next

	iconVolHigh = "\U000F057E" // 󰕾 nf-md-volume_high
	iconVolOff  = "\U000F0581" // 󰖁 nf-md-volume_off

	iconShuffle   = "\U000F049D" // 󰒝 nf-md-shuffle_variant
	iconRepeat    = "\U000F0456" // 󰑖 nf-md-repeat
	iconRepeatOne = "\U000F0458" // 󰑘 nf-md-repeat_once

	restartThreshold = 3.0 // seconds before 'h' restarts instead of going previous

	maxQueueTracks = 10 // max upcoming tracks to display
)

var out = os.Stdout

// Dynamic layout dimensions, recalculated on terminal resize.
var (
	artworkCols  = defaultArtworkCols
	artworkRows  = defaultArtworkRows
	displayRows  = padTop + artworkRows
	terminalCols int // 0 means unknown
)

// recalcLayout adjusts artwork and text dimensions to fit the current terminal size.
func recalcLayout() {
	w, h, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		w, h, err = term.GetSize(int(out.Fd()))
	}
	if err != nil || h <= 0 {
		return
	}
	terminalCols = w
	artworkRows = defaultArtworkRows
	if h < padTop+defaultArtworkRows+2 {
		artworkRows = h - 2
		if artworkRows < 1 {
			artworkRows = 1
		}
	}
	artworkCols = artworkRows * 2
	displayRows = padTop + artworkRows
}

// truncateVisible truncates s so that its visible (non-ANSI-escape) terminal
// cell width does not exceed maxWidth, correctly handling wide characters.
// When truncated, an ellipsis (…) is appended and fits within maxWidth.
// Any open SGR styling is reset with \x1b[0m.
func truncateVisible(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	var (
		width    int
		inEsc    bool
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

	recalcLayout()

	// Switch to alternate screen buffer and hide cursor while running.
	fmt.Fprint(out, "\x1b[?1049h\x1b[?25l")
	defer fmt.Fprint(out, "\x1b[?25h\x1b[?1049l")

	// Listen for terminal resize signals.
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)

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

	// overlayMode tracks whether we're showing an overlay.
	// "" = normal now-playing view, "queue" = queue list, "browse" = browse modal.
	var overlayMode string

	const queueVisible = 5 // max queue tracks visible at once

	var (
		currentTrack   string
		hasArtwork     bool
		spaceAllocated bool
		latestInfo     *music.TrackInfo
		pollRunning    bool
		flashIcon      string // "prev", "play", "next", or ""
		shuffleOn      bool
		repeatMode     string // "off", "one", or "all"
		queueTracks    []music.QueueTrack
		queueScroll    int // index of first visible queue track
		queueLoading   bool
		spinnerFrame   int
		forceRedraw    bool // next poll result redraws even during overlay

		browse            browseState
		autoBrowsePending = true // auto-open browse on first nil poll
	)

	spinnerChars := [4]string{"⠋", "⠙", "⠹", "⠸"}

	type pollResult struct {
		info *music.TrackInfo
		err  error
	}
	pollCh := make(chan pollResult, 1)

	allocateSpace := func() {
		fmt.Fprint(out, "\x1b[2J\x1b[3J\x1b[H") // clear screen, scrollback, and move cursor to top-left
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

	// drawModalLine writes a single line at the given row/col using
	// the same save/restore cursor approach as the rest of the rendering.
	drawModalLine := func(row, col int, s string) {
		fmt.Fprint(out, "\x1b8")
		if row > 0 {
			fmt.Fprintf(out, "\x1b[%dB", row)
		}
		if col > 0 {
			fmt.Fprintf(out, "\x1b[%dC", col)
		}
		fmt.Fprintf(out, "\x1b[K%s", s)
	}

	// drawQueue renders the queue as a bordered modal to the right of artwork.
	drawQueue := func() {
		modalCol := padLeft
		if hasArtwork {
			modalCol = artworkLeft + artworkCols + 2
		}
		modalWidth := 40
		if terminalCols > modalCol+2 {
			modalWidth = terminalCols - modalCol - 2
		}
		innerWidth := modalWidth - 4 // borders + padding

		// Title bar.
		title := "Up Next"
		titleLen := len(title)
		pad := modalWidth - 2 - titleLen - 2
		if pad < 0 {
			pad = 0
		}
		drawModalLine(padTop, modalCol,
			"┌ \x1b[1m"+title+"\x1b[0m "+strings.Repeat("─", pad)+"┐")

		// Content rows.
		rows := queueVisible
		if rows > artworkRows-2 {
			rows = artworkRows - 2
			if rows < 1 {
				rows = 1
			}
		}
		for i := range rows {
			idx := queueScroll + i
			var content string
			if queueLoading && i == 0 {
				content = spinnerChars[spinnerFrame%len(spinnerChars)] + " Loading…"
			} else if !queueLoading && len(queueTracks) == 0 && i == 0 {
				content = "No upcoming tracks."
			} else if idx < len(queueTracks) {
				t := queueTracks[idx]
				content = fmt.Sprintf("%2d. %s - %s", idx+1, t.Artist, t.Name)
				content = truncateVisible(content, innerWidth)
			}
			// Pad to fill interior.
			visible := visibleLen(content)
			trailing := innerWidth - visible
			if trailing < 0 {
				trailing = 0
			}
			drawModalLine(padTop+1+i, modalCol,
				"│ "+content+strings.Repeat(" ", trailing)+" │")
		}

		// Scroll indicators in bottom border.
		bottom := strings.Repeat("─", modalWidth-2)
		if !queueLoading && len(queueTracks) > 0 {
			pos := fmt.Sprintf(" %d-%d/%d ", queueScroll+1,
				min(queueScroll+rows, len(queueTracks)), len(queueTracks))
			if len(pos) < modalWidth-2 {
				bottom = strings.Repeat("─", modalWidth-2-len(pos)) + pos
			}
		}
		drawModalLine(padTop+1+rows, modalCol,
			"└"+bottom+"┘")
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

	noTrackLines := []string{
		"No track is currently playing.",
		"\x1b[2mPress b to browse, or start a track in Music.\x1b[0m",
	}

	// handlePoll processes an async poll result and updates the display.
	handlePoll := func(info *music.TrackInfo) {
		if info == nil {
			if currentTrack != "" {
				clearDisplay()
				drawText(noTrackLines, false)
				currentTrack = ""
				hasArtwork = false
			} else if !spaceAllocated {
				allocateSpace()
				drawText(noTrackLines, false)
			}
			return
		}

		trackKey := info.Artist + "\x00" + info.Album + "\x00" + info.Name

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

			lines := buildLines(info, flashIcon, hasArtwork, shuffleOn, repeatMode)
			drawText(lines, hasArtwork)
			if hasArtwork {
				drawVolumeBar(flashIcon)
			}
			currentTrack = trackKey
		} else {
			lines := buildLines(info, flashIcon, hasArtwork, shuffleOn, repeatMode)
			updateDynamicLines(lines, hasArtwork)
			if hasArtwork {
				drawVolumeBar(flashIcon)
			}
		}
	}

	// Launch Music.app in the background (does not steal focus).
	go music.Launch(ctx)

	// Fetch initial playback option state.
	if on, err := music.ShuffleEnabled(ctx); err == nil {
		shuffleOn = on
	}
	if mode, err := music.RepeatMode(ctx); err == nil {
		repeatMode = mode
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

	// spinner ticker animates the queue loading spinner.
	spinnerTicker := time.NewTicker(100 * time.Millisecond)
	spinnerTicker.Stop()

	// queueCh delivers queue results from the background fetch.
	queueCh := make(chan []music.QueueTrack, 1)

	// Browse modal channels.
	playlistsCh := make(chan []music.Playlist, 1)
	albumsCh := make(chan []music.Album, 1)
	albumTracksCh := make(chan struct {
		key    string
		tracks []music.AlbumTrack
	}, 1)

	// openBrowse initialises the browse modal and begins loading playlists.
	openBrowse := func(auto bool) {
		overlayMode = "browse"
		browse = browseState{
			Screen:          browseRoot,
			AlbumTrackCache: browse.AlbumTrackCache,
			Albums:          browse.Albums,
			AlbumsLoaded:    browse.AlbumsLoaded,
			AutoOpened:      auto,
		}
		if browse.AlbumTrackCache == nil {
			browse.AlbumTrackCache = make(map[string][]music.AlbumTrack)
		}
		spinnerFrame = 0
		spinnerTicker.Reset(100 * time.Millisecond)
		if !spaceAllocated {
			allocateSpace()
		}
		drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
	}

	// fetchPlaylists starts loading playlists in the background.
	fetchPlaylists := func() {
		if browse.PlaylistsLoading {
			return
		}
		browse.PlaylistsLoading = true
		go func() {
			items, _ := music.ListPlaylists(ctx)
			playlistsCh <- items
		}()
	}

	// fetchAlbums starts loading albums in the background.
	fetchAlbums := func() {
		if browse.AlbumsLoading || browse.AlbumsLoaded {
			return
		}
		browse.AlbumsLoading = true
		go func() {
			items, _ := music.ListAlbums(ctx)
			albumsCh <- items
		}()
	}

	// fetchAlbumTracks starts loading tracks for the selected album.
	fetchAlbumTracks := func(album *music.Album) {
		if browse.TracksLoading {
			return
		}
		key := albumCacheKey(album.Name, album.AlbumArtist)
		if cached, ok := browse.AlbumTrackCache[key]; ok {
			browse.AlbumTracks = cached
			return
		}
		browse.TracksLoading = true
		name, artist := album.Name, album.AlbumArtist
		go func() {
			items, _ := music.ListAlbumTracks(ctx, name, artist)
			albumTracksCh <- struct {
				key    string
				tracks []music.AlbumTrack
			}{key, items}
		}()
	}

	// exitOverlay returns to the normal now-playing view,
	// redrawing only the text area that the modal overwrote.
	exitOverlay := func() {
		overlayMode = ""
		queueLoading = false
		spinnerTicker.Stop()
		// Clear all rows in the text column area (plus hint lines below modal).
		colOffset := padLeft
		if hasArtwork {
			colOffset = artworkLeft + artworkCols + 2
		}
		for i := range displayRows + 3 {
			fmt.Fprint(out, "\x1b8")
			if i > 0 {
				fmt.Fprintf(out, "\x1b[%dB", i)
			}
			if colOffset > 0 {
				fmt.Fprintf(out, "\x1b[%dC", colOffset)
			}
			fmt.Fprint(out, "\x1b[K")
		}
		if latestInfo != nil {
			lines := buildLines(latestInfo, flashIcon, hasArtwork, shuffleOn, repeatMode)
			drawText(lines, hasArtwork)
		} else {
			drawText([]string{
				"No track is currently playing.",
				"\x1b[2mPress b to browse, or start a track in Music.\x1b[0m",
			}, false)
		}
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
		fmt.Fprintf(out, "\x1b[K%s", controlsLine(latestInfo.Playing, flashIcon, shuffleOn, repeatMode))
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
				if overlayMode == "" {
					fmt.Fprintf(os.Stderr, "muzak: poll: %v\r\n", result.err)
				}
			} else {
				latestInfo = result.info
				// Auto-open browse on first successful nil poll.
				if autoBrowsePending && result.info == nil {
					autoBrowsePending = false
					handlePoll(result.info)
					openBrowse(true)
				} else {
					if autoBrowsePending && result.info != nil {
						autoBrowsePending = false
					}
					if overlayMode == "" || forceRedraw {
						handlePoll(result.info)
						if forceRedraw && overlayMode == "queue" {
							drawQueue()
						}
						if forceRedraw && overlayMode == "browse" {
							drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
						}
						forceRedraw = false
					}
				}
			}
		case key := <-keys:
			// Global keys that work in any mode.
			switch key {
			case 3: // Ctrl+C
				if overlayMode != "" {
					fmt.Fprint(out, "\x1b[2J\x1b[H")
				} else if spaceAllocated {
					fmt.Fprint(out, "\x1b8")
					fmt.Fprintf(out, "\x1b[%dB\r\n", displayRows)
				}
				return
			}

			// Overlay-specific keys.
			if overlayMode == "queue" {
				rows := queueVisible
				if rows > artworkRows-2 {
					rows = artworkRows - 2
					if rows < 1 {
						rows = 1
					}
				}
				switch key {
				case 'q':
					exitOverlay()
				case 'j':
					if queueScroll+rows < len(queueTracks) {
						queueScroll++
						drawQueue()
					}
				case 'k':
					if queueScroll > 0 {
						queueScroll--
						drawQueue()
					}
				}
				continue
			}

			if overlayMode == "browse" {
				if browse.SearchActive {
					switch key {
					case 27: // Escape — clear search
						clearSearch(&browse)
						*browse.currentList() = listState{}
						drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
					case '\r', '\n': // Enter — confirm search, exit input mode
						browse.SearchActive = false
						drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
					case 127, 8: // Backspace
						if len(browse.SearchQuery) > 0 {
							browse.SearchQuery = browse.SearchQuery[:len(browse.SearchQuery)-1]
							*browse.currentList() = listState{}
							drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
						}
					default:
						if key >= 32 && key <= 126 {
							browse.SearchQuery += string(rune(key))
							*browse.currentList() = listState{}
							drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
						}
					}
					continue
				}
				switch key {
				case 'x', 27: // x or Escape
					clearSearch(&browse)
					if browse.Screen == browseRoot {
						exitOverlay()
					} else {
						browse.Screen = browseRoot
						drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
					}
				case 'h':
					clearSearch(&browse)
					switch browse.Screen {
					case browseRoot:
						exitOverlay()
					case browsePlaylists, browseAlbums:
						browse.Screen = browseRoot
						drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
					case browseFolder:
						exitFolder(&browse)
						drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
					case browseAlbumTracks:
						browse.Screen = browseAlbums
						drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
					}
				case 'j':
					browseMoveCursor(&browse, 1)
					drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
				case 'k':
					browseMoveCursor(&browse, -1)
					drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
				case '\r', '\n', 'l':
					switch browse.Screen {
					case browseRoot:
						switch browse.RootView.Cursor {
						case rootItemPlaylists:
							clearSearch(&browse)
							browse.Screen = browsePlaylists
							browse.PlaylistsView = listState{}
							browse.FolderStack = nil
							browse.FolderID = ""
							fetchPlaylists()
							drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
						case rootItemLibrary:
							clearSearch(&browse)
							browse.Screen = browseAlbums
							browse.AlbumsView = listState{}
							fetchAlbums()
							drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
						}
					case browsePlaylists:
						if !browse.PlaylistsLoading {
							p, ok := browseSelectedPlaylist(&browse)
							if ok {
								if p.IsFolder() {
									clearSearch(&browse)
									enterFolder(&browse, p)
									drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
								} else {
									go music.PlayPlaylist(ctx, p.PersistentID)
									exitOverlay()
									safeReset(refresh, 500*time.Millisecond)
								}
							}
						}
					case browseFolder:
						p, ok := browseSelectedFolderItem(&browse)
						if ok {
							if p.IsFolder() {
								clearSearch(&browse)
								enterFolder(&browse, p)
								drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
							} else {
								go music.PlayPlaylist(ctx, p.PersistentID)
								exitOverlay()
								safeReset(refresh, 500*time.Millisecond)
							}
						}
					case browseAlbums:
						if !browse.AlbumsLoading {
							a, ok := browseSelectedAlbum(&browse)
							if ok {
								browse.SelectedAlbum = &a
								browse.Screen = browseAlbumTracks
								browse.TracksView = listState{}
								browse.AlbumTracks = nil
								fetchAlbumTracks(&a)
								drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
							}
						}
					case browseAlbumTracks:
						if !browse.TracksLoading && browse.SelectedAlbum != nil {
							a := browse.SelectedAlbum
							go music.PlayAlbum(ctx, a.Name, a.AlbumArtist)
							exitOverlay()
							safeReset(refresh, 500*time.Millisecond)
						}
					}
				case 'a':
					if browse.Screen != browseRoot && !browse.SearchActive {
						browse.Sort = sortAsc
						*browse.currentList() = listState{}
						drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
					}
				case 'd':
					if browse.Screen != browseRoot && !browse.SearchActive {
						browse.Sort = sortDesc
						*browse.currentList() = listState{}
						drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
					}
				case '/':
					if browse.Screen != browseRoot && !browse.SearchActive {
						browse.SearchActive = true
						browse.SearchQuery = ""
						drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
					}
				}
				continue
			}

			// Normal now-playing keys.
			switch key {
			case 'x':
				if spaceAllocated {
					fmt.Fprint(out, "\x1b8")
					fmt.Fprintf(out, "\x1b[%dB\r\n", displayRows)
				}
				return
			case ' ':
				go music.PlayPause(ctx)
				if latestInfo != nil {
					latestInfo.Playing = !latestInfo.Playing
				}
				flashIcon = "play"
				redrawControls()
				safeReset(flashTimer, 500*time.Millisecond)
			case 'l':
				go music.NextTrack(ctx)
				flashIcon = "next"
				redrawControls()
				safeReset(flashTimer, 500*time.Millisecond)
			case 'h':
				if latestInfo != nil && latestInfo.Position > restartThreshold {
					go music.RestartTrack(ctx)
				} else {
					go music.PreviousTrack(ctx)
				}
				flashIcon = "prev"
				redrawControls()
				safeReset(flashTimer, 500*time.Millisecond)
			case 'j':
				_ = volume.Down()
				flashIcon = "voldn"
				if hasArtwork {
					drawVolumeBar(flashIcon)
				}
				safeReset(flashTimer, 500*time.Millisecond)
			case 'k':
				_ = volume.Up()
				flashIcon = "volup"
				if hasArtwork {
					drawVolumeBar(flashIcon)
				}
				safeReset(flashTimer, 500*time.Millisecond)
			case 's':
				go func() {
					if on, err := music.ToggleShuffle(ctx); err == nil {
						shuffleOn = on
						redrawControls()
					}
				}()
			case 'r':
				go func() {
					if mode, err := music.CycleRepeat(ctx); err == nil {
						repeatMode = mode
						redrawControls()
					}
				}()
			case 'q':
				overlayMode = "queue"
				queueScroll = 0
				queueLoading = true
				spinnerFrame = 0
				drawQueue()
				spinnerTicker.Reset(100 * time.Millisecond)
				go func() {
					tracks, _ := music.Queue(ctx)
					queueCh <- tracks
				}()
			case 'b':
				openBrowse(false)
			}
			// Schedule a refresh to pick up the new state.
			safeReset(refresh, 300*time.Millisecond)
		case <-flashTimer.C:
			wasVol := flashIcon == "volup" || flashIcon == "voldn"
			flashIcon = ""
			if wasVol && hasArtwork {
				drawVolumeBar(flashIcon)
			} else {
				redrawControls()
			}
		case <-spinnerTicker.C:
			spinnerFrame++
			if queueLoading && overlayMode == "queue" {
				drawQueue()
			}
			if browse.isLoading() && overlayMode == "browse" {
				drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
			}
		case tracks := <-queueCh:
			queueLoading = false
			spinnerTicker.Stop()
			if tracks != nil {
				queueTracks = tracks
			}
			if overlayMode == "queue" {
				drawQueue()
			}
		case items := <-playlistsCh:
			browse.PlaylistsLoading = false
			browse.PlaylistsLoaded = true
			if items != nil {
				browse.Playlists = items
			}
			if overlayMode == "browse" {
				drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
			}
		case items := <-albumsCh:
			browse.AlbumsLoading = false
			browse.AlbumsLoaded = true
			if items != nil {
				browse.Albums = items
			}
			if overlayMode == "browse" {
				drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
			}
		case result := <-albumTracksCh:
			browse.TracksLoading = false
			if result.tracks != nil {
				browse.AlbumTracks = result.tracks
				browse.AlbumTrackCache[result.key] = result.tracks
			}
			if overlayMode == "browse" {
				drawBrowse(&browse, spinnerFrame, hasArtwork, drawModalLine)
			}
		case <-winch:
			// Terminal was resized — recalculate layout and force a full redraw.
			recalcLayout()
			currentTrack = ""
			spaceAllocated = false
			hasArtwork = false
			forceRedraw = overlayMode != ""
			allocateSpace()
			startPoll()
		case <-refresh.C:
			startPoll()
		case <-ticker.C:
			startPoll()
		}
	}
}

// buildLines constructs the five display lines for the current track.
func buildLines(info *music.TrackInfo, flash string, withArtwork bool, shuffle bool, repeat string) []string {
	bw := barWidth
	if terminalCols > 0 {
		colOffset := padLeft
		if withArtwork {
			colOffset = artworkLeft + artworkCols + 2
		}
		avail := terminalCols - colOffset - 2
		if avail < bw && avail > 0 {
			bw = avail
		}
	}
	return []string{
		"\x1b[1m" + info.Name + "\x1b[0m",
		info.Artist + " - " + info.Album,
		formatDuration(info.Position) + " / " + formatDuration(info.Duration),
		progressBar(info.Position, info.Duration, bw),
		controlsLine(info.Playing, flash, shuffle, repeat),
	}
}

const (
	yellow = "\x1b[33m"
	reset  = "\x1b[0m"
)

// controlsLine renders the playback control icons with shuffle/repeat state.
func controlsLine(playing bool, flash string, shuffle bool, repeat string) string {
	shuf := iconShuffle
	if shuffle {
		shuf = yellow + iconShuffle + reset
	}

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

	var rep string
	switch repeat {
	case "one":
		rep = yellow + iconRepeatOne + reset
	case "all":
		rep = yellow + iconRepeat + reset
	default:
		rep = iconRepeat
	}

	return shuf + "  " + prev + "  " + pp + "  " + next + "  " + rep
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

// safeReset stops the timer, drains any pending event, and resets it.
// This avoids stale timer fires per the time.Timer.Reset documentation.
func safeReset(t *time.Timer, d time.Duration) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(d)
}

// drawText writes all text lines into the display area, positioned beside
// artwork (if present) and vertically centered.
func drawText(lines []string, withArtwork bool) {
	startRow := padTop + (artworkRows-len(lines))/2
	colOffset := padLeft
	if withArtwork {
		colOffset = artworkLeft + artworkCols + 2
	}
	maxWidth := 0
	if terminalCols > colOffset {
		maxWidth = terminalCols - colOffset
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
		if maxWidth > 0 {
			line = truncateVisible(line, maxWidth)
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
	maxWidth := 0
	if terminalCols > colOffset {
		maxWidth = terminalCols - colOffset
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
		line := lines[i]
		if maxWidth > 0 {
			line = truncateVisible(line, maxWidth)
		}
		fmt.Fprintf(out, "\x1b[K%s", line)
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
