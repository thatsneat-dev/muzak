package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/thatsneat-dev/muzak/internal/browse"
	"github.com/thatsneat-dev/muzak/internal/catalog"
	"github.com/thatsneat-dev/muzak/internal/music"
	"github.com/thatsneat-dev/muzak/internal/ui"
	"github.com/thatsneat-dev/muzak/internal/volume"
)

const queueVisible = 5 // max queue tracks visible at once

// albumTracksResult bundles the cache key with fetched tracks.
type albumTracksResult struct {
	key    string
	tracks []music.AlbumTrack
}

// pollResult holds the outcome of a NowPlaying poll.
type pollResult struct {
	info *music.TrackInfo
	err  error
}

// app holds all runtime state for the muzak TUI.
type app struct {
	ctx context.Context
	out *os.File

	layout      ui.Layout
	artworkPath string

	overlayMode    string
	currentTrack   string
	hasArtwork     bool
	spaceAllocated bool
	latestInfo     *music.TrackInfo
	pollRunning    bool
	flashIcon      string
	shuffleOn      bool
	repeatMode     string
	forceRedraw    bool

	queueTracks  []music.QueueTrack
	queueScroll  int
	queueLoading bool
	spinnerFrame int

	browseState       browse.State
	autoBrowsePending bool

	keys          <-chan byte
	winch         <-chan os.Signal
	pollCh        chan pollResult
	queueCh       chan []music.QueueTrack
	playlistsCh   chan []music.Playlist
	albumsCh      chan []music.Album
	albumTracksCh chan albumTracksResult
	catalogCh     chan []catalog.Song

	ticker        *time.Ticker
	refresh       *time.Timer
	flashTimer    *time.Timer
	spinnerTicker *time.Ticker

	drawModalLine ui.LineDrawer
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
		queueCh:       make(chan []music.QueueTrack, 1),
		playlistsCh:   make(chan []music.Playlist, 1),
		albumsCh:      make(chan []music.Album, 1),
		albumTracksCh: make(chan albumTracksResult, 1),
		catalogCh:     make(chan []catalog.Song, 1),

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

var noTrackLines = []string{
	"No track is currently playing.",
	"\x1b[2mPress b to browse, or start a track in Music.\x1b[0m",
}

// handlePollInfo processes a poll result and updates the display.
func (a *app) handlePollInfo(info *music.TrackInfo) {
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
		a.browseState.AlbumTrackCache = make(map[string][]music.AlbumTrack)
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
func (a *app) fetchAlbumTracks(album *music.Album) {
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
