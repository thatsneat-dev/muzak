package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/thatsneat-dev/muzak/internal/browse"
	"github.com/thatsneat-dev/muzak/internal/catalog"
	"github.com/thatsneat-dev/muzak/internal/model"
	"github.com/thatsneat-dev/muzak/internal/music"
	"github.com/thatsneat-dev/muzak/internal/ui"
	"github.com/thatsneat-dev/muzak/internal/volume"
)

const queueVisible = 5 // max queue tracks visible at once

// albumTracksResult bundles the cache key with fetched tracks.
type albumTracksResult struct {
	key    string
	tracks []model.AlbumTrack
}

// pollResult holds the outcome of a NowPlaying poll.
type pollResult struct {
	info *model.TrackInfo
	err  error
}

// app holds all runtime state for the muzak TUI.
type app struct {
	ctx context.Context // Application lifecycle context.
	out *os.File        // Terminal output file (stdout).

	layout      ui.Layout // Computed terminal dimensions.
	artworkPath string    // Temp file path for artwork PNG data.

	overlayMode    string           // Active overlay ("queue", "browse", or "").
	currentTrack   string           // Cache key of the currently displayed track.
	hasArtwork     bool             // Whether artwork is currently displayed.
	spaceAllocated bool             // Whether display space has been reserved.
	latestInfo     *model.TrackInfo // Most recent track info from polling.
	pollRunning    bool             // A NowPlaying poll is in flight.
	flashIcon      string           // Currently flashing control icon name.
	shuffleOn      bool             // Shuffle mode is active.
	repeatMode     string           // Current repeat mode ("off", "one", "all").
	forceRedraw    bool             // Forces a full redraw on next poll result.

	queueTracks  []model.QueueTrack // Fetched queue tracks.
	queueScroll  int                // Scroll offset in the queue modal.
	queueLoading bool               // Queue data is being fetched.
	spinnerFrame int                // Current animation frame for loading spinners.

	browseState       browse.State // Browse modal state machine.
	autoBrowsePending bool         // Auto-opens browse when no track is playing.

	keys          <-chan byte             // Raw keypresses from stdin.
	winch         <-chan os.Signal        // SIGWINCH signals for terminal resize.
	pollCh        chan pollResult         // NowPlaying poll results.
	queueCh       chan []model.QueueTrack // Fetched queue tracks.
	playlistsCh   chan []model.Playlist   // Fetched playlists.
	albumsCh      chan []model.Album      // Fetched albums.
	albumTracksCh chan albumTracksResult  // Fetched album tracks.
	catalogCh     chan []model.Song       // Catalog search results.
	shuffleCh     chan bool               // Receives shuffle toggle results.
	repeatCh      chan string             // Receives repeat cycle results.

	ticker        *time.Ticker // Fires every second to trigger polling.
	refresh       *time.Timer  // Triggers a delayed poll after user actions.
	flashTimer    *time.Timer  // Clears the flash icon after a delay.
	spinnerTicker *time.Ticker // Drives loading spinner animation.

	drawModalLine ui.LineDrawer // Line drawer for modal overlays.
}

// newApp creates and initializes a new app instance.
func newApp(ctx context.Context, out *os.File, artworkPath string, layout ui.Layout, keys <-chan byte, winch <-chan os.Signal) *app {
	a := &app{
		ctx:         ctx,
		out:         out,
		layout:      layout,
		artworkPath: artworkPath,

		repeatMode:        "off",
		autoBrowsePending: true,

		keys:          keys,
		winch:         winch,
		pollCh:        make(chan pollResult, 1),
		queueCh:       make(chan []model.QueueTrack, 1),
		playlistsCh:   make(chan []model.Playlist, 1),
		albumsCh:      make(chan []model.Album, 1),
		albumTracksCh: make(chan albumTracksResult, 1),
		catalogCh:     make(chan []model.Song, 1),
		shuffleCh:     make(chan bool, 1),
		repeatCh:      make(chan string, 1),

		ticker:        time.NewTicker(time.Second),
		spinnerTicker: time.NewTicker(100 * time.Millisecond),

		drawModalLine: ui.NewLineDrawer(out),
	}
	a.spinnerTicker.Stop()

	a.refresh = time.NewTimer(0)
	if !a.refresh.Stop() {
		<-a.refresh.C
	}

	a.flashTimer = time.NewTimer(0)
	if !a.flashTimer.Stop() {
		<-a.flashTimer.C
	}

	return a
}

// cleanup releases resources held by the app.
func (a *app) cleanup() {
	a.ticker.Stop()
	a.spinnerTicker.Stop()
}

// nowPlayingOpts returns the current NowPlayingOptions.
func (a *app) nowPlayingOpts() ui.NowPlayingOptions {
	return ui.NowPlayingOptions{
		Flash:   a.flashIcon,
		Shuffle: a.shuffleOn,
		Repeat:  a.repeatMode,
	}
}

// allocateSpace clears the screen and reserves display rows.
func (a *app) allocateSpace() {
	ui.AllocateSpace(a.out, a.layout)
	a.spaceAllocated = true
}

// clearDisplay clears all rows and removes Kitty images.
func (a *app) clearDisplay() {
	ui.ClearDisplay(a.out, a.layout)
}

// startPoll kicks off a NowPlaying query in the background.
func (a *app) startPoll() {
	if a.pollRunning {
		return
	}
	_ = os.Truncate(a.artworkPath, 0)
	a.pollRunning = true
	go func() {
		info, err := music.NowPlaying(a.ctx, a.artworkPath)
		select {
		case a.pollCh <- pollResult{info, err}:
		case <-a.ctx.Done():
		}
	}()
}

// drawQueue renders the queue modal.
func (a *app) drawQueue() {
	ui.DrawQueue(a.layout, a.hasArtwork, ui.QueueViewModel{
		Tracks:       a.queueTracks,
		Scroll:       a.queueScroll,
		Loading:      a.queueLoading,
		SpinnerFrame: a.spinnerFrame,
		VisibleLimit: queueVisible,
	}, a.drawModalLine)
}

// drawBrowse renders the browse modal.
func (a *app) drawBrowse() {
	browse.Draw(&a.browseState, a.layout, a.spinnerFrame, a.hasArtwork, a.drawModalLine)
}

// drawVolume renders the volume bar if artwork is visible.
func (a *app) drawVolume() {
	if a.hasArtwork {
		vol, _ := volume.Get()
		ui.DrawVolumeBar(a.out, a.layout, vol, a.flashIcon)
	}
}

// redrawControls overwrites just the controls line with current flash state.
func (a *app) redrawControls() {
	if a.latestInfo == nil {
		return
	}
	startRow := ui.PadTop + (a.layout.ArtworkRows-ui.NumTextLines)/2
	controlRow := startRow + ui.NumTextLines - 1
	colOffset := a.layout.TextCol(a.hasArtwork)
	_, _ = fmt.Fprint(a.out, "\x1b8")
	if controlRow > 0 {
		_, _ = fmt.Fprintf(a.out, "\x1b[%dB", controlRow)
	}
	if colOffset > 0 {
		_, _ = fmt.Fprintf(a.out, "\x1b[%dC", colOffset)
	}
	_, _ = fmt.Fprintf(a.out, "\x1b[K%s", ui.ControlsLine(a.latestInfo.Playing, a.flashIcon, a.shuffleOn, a.repeatMode))
}

// noTrackLines are the display lines shown when no track is currently playing.
var noTrackLines = []string{
	"No track is currently playing.",
	"\x1b[2mPress b to browse, or start a track in Music.\x1b[0m",
}

// handlePollInfo processes a poll result and updates the display.
func (a *app) handlePollInfo(info *model.TrackInfo) {
	if info == nil {
		if a.currentTrack != "" {
			a.clearDisplay()
			ui.DrawText(a.out, a.layout, noTrackLines, false)
			a.currentTrack = ""
			a.hasArtwork = false
		} else if !a.spaceAllocated {
			a.allocateSpace()
			ui.DrawText(a.out, a.layout, noTrackLines, false)
		}
		return
	}

	trackKey := info.Artist + "\x00" + info.Album + "\x00" + info.Name

	if trackKey != a.currentTrack {
		if !a.spaceAllocated {
			a.allocateSpace()
		} else {
			a.clearDisplay()
		}

		fi, _ := os.Stat(a.artworkPath)
		a.hasArtwork = fi != nil && fi.Size() > 0

		if a.hasArtwork {
			if pngData, err := os.ReadFile(a.artworkPath); err == nil && len(pngData) > 0 {
				_, _ = fmt.Fprintf(a.out, "\x1b8\x1b[%dB\x1b[%dC", ui.PadTop, a.layout.ArtworkLeft())
				ui.SendKittyImage(a.out, a.layout, pngData)
			}
		}

		lines := ui.BuildLines(info, a.layout, a.hasArtwork, a.nowPlayingOpts())
		ui.DrawText(a.out, a.layout, lines, a.hasArtwork)
		if a.hasArtwork {
			vol, _ := volume.Get()
			ui.DrawVolumeBar(a.out, a.layout, vol, a.flashIcon)
		}
		a.currentTrack = trackKey
	} else {
		lines := ui.BuildLines(info, a.layout, a.hasArtwork, a.nowPlayingOpts())
		ui.UpdateDynamicLines(a.out, a.layout, lines, a.hasArtwork)
		if a.hasArtwork {
			vol, _ := volume.Get()
			ui.DrawVolumeBar(a.out, a.layout, vol, a.flashIcon)
		}
	}
}

// exitOverlay returns to the now-playing view.
func (a *app) exitOverlay() {
	a.overlayMode = ""
	a.queueLoading = false
	a.spinnerTicker.Stop()
	colOffset := a.layout.TextCol(a.hasArtwork)
	for i := range a.layout.DisplayRows() + 3 {
		_, _ = fmt.Fprint(a.out, "\x1b8")
		if i > 0 {
			_, _ = fmt.Fprintf(a.out, "\x1b[%dB", i)
		}
		if colOffset > 0 {
			_, _ = fmt.Fprintf(a.out, "\x1b[%dC", colOffset)
		}
		_, _ = fmt.Fprint(a.out, "\x1b[K")
	}
	if a.latestInfo != nil {
		lines := ui.BuildLines(a.latestInfo, a.layout, a.hasArtwork, a.nowPlayingOpts())
		ui.DrawText(a.out, a.layout, lines, a.hasArtwork)
	} else {
		ui.DrawText(a.out, a.layout, noTrackLines, false)
	}
}

// openBrowse initializes and opens the browse modal.
func (a *app) openBrowse(auto bool) {
	a.overlayMode = "browse"
	a.browseState = browse.State{
		Screen:          browse.ScreenRoot,
		AlbumTrackCache: a.browseState.AlbumTrackCache,
		Albums:          a.browseState.Albums,
		AlbumsLoaded:    a.browseState.AlbumsLoaded,
		AutoOpened:      auto,
	}
	if a.browseState.AlbumTrackCache == nil {
		a.browseState.AlbumTrackCache = make(map[string][]model.AlbumTrack)
	}
	a.spinnerFrame = 0
	a.spinnerTicker.Reset(100 * time.Millisecond)
	if !a.spaceAllocated {
		a.allocateSpace()
	}
	a.drawBrowse()
}

// fetchPlaylists starts loading playlists in the background.
func (a *app) fetchPlaylists() {
	if a.browseState.PlaylistsLoading {
		return
	}
	a.browseState.PlaylistsLoading = true
	go func() {
		items, _ := music.ListPlaylists(a.ctx)
		a.playlistsCh <- items
	}()
}

// fetchAlbums starts loading albums in the background.
func (a *app) fetchAlbums() {
	if a.browseState.AlbumsLoading || a.browseState.AlbumsLoaded {
		return
	}
	a.browseState.AlbumsLoading = true
	go func() {
		items, _ := music.ListAlbums(a.ctx)
		a.albumsCh <- items
	}()
}

// fetchAlbumTracks starts loading tracks for the selected album.
func (a *app) fetchAlbumTracks(album *model.Album) {
	if a.browseState.TracksLoading {
		return
	}
	key := browse.AlbumCacheKey(album.Name, album.AlbumArtist)
	if cached, ok := a.browseState.AlbumTrackCache[key]; ok {
		a.browseState.AlbumTracks = cached
		return
	}
	a.browseState.TracksLoading = true
	name, artist := album.Name, album.AlbumArtist
	go func() {
		items, _ := music.ListAlbumTracks(a.ctx, name, artist)
		a.albumTracksCh <- albumTracksResult{key, items}
	}()
}

// fetchCatalog starts a catalog search in the background.
func (a *app) fetchCatalog(query string) {
	if a.browseState.CatalogLoading {
		return
	}
	a.browseState.CatalogLoading = true
	go func() {
		songs, _ := catalog.Search(a.ctx, query)
		a.catalogCh <- songs
	}()
}

// quit handles the exit sequence.
func (a *app) quit() {
	if a.spaceAllocated {
		_, _ = fmt.Fprint(a.out, "\x1b8")
		_, _ = fmt.Fprintf(a.out, "\x1b[%dB\r\n", a.layout.DisplayRows())
	}
}
