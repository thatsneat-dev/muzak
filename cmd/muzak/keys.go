package main

import (
	"fmt"
	"time"

	"github.com/thatsneat-dev/muzak/internal/browse"
	"github.com/thatsneat-dev/muzak/internal/music"
	"github.com/thatsneat-dev/muzak/internal/ui"
	"github.com/thatsneat-dev/muzak/internal/volume"
)

// handleKey dispatches a keypress to the appropriate handler.
// Returns true if the app should exit.
func (a *app) handleKey(key byte) bool {
	// Global keys.
	if key == 3 { // Ctrl+C
		if a.overlayMode != "" {
			_, _ = fmt.Fprint(a.out, "\x1b[2J\x1b[H")
		} else {
			a.quit()
		}
		return true
	}

	switch a.overlayMode {
	case "queue":
		a.handleQueueKey(key)
	case "browse":
		a.handleBrowseKey(key)
	default:
		if a.handleNowPlayingKey(key) {
			return true
		}
	}
	return false
}

// handleQueueKey handles keys while the queue overlay is active.
func (a *app) handleQueueKey(key byte) {
	rows := queueVisible
	if rows > a.layout.ArtworkRows-2 {
		rows = a.layout.ArtworkRows - 2
		if rows < 1 {
			rows = 1
		}
	}
	switch key {
	case 'q', 'x':
		a.exitOverlay()
	case 'j':
		if a.queueScroll+rows < len(a.queueTracks) {
			a.queueScroll++
			a.drawQueue()
		}
	case 'k':
		if a.queueScroll > 0 {
			a.queueScroll--
			a.drawQueue()
		}
	}
}

// handleBrowseKey handles keys while the browse overlay is active.
func (a *app) handleBrowseKey(key byte) {
	if a.browseState.Screen == browse.ScreenCatalogSearch {
		a.handleCatalogSearchKey(key)
		return
	}
	if a.browseState.SearchActive {
		a.handleBrowseSearchKey(key)
		return
	}

	switch key {
	case 'x', 27: // x or Escape
		browse.ClearSearch(&a.browseState)
		if a.browseState.Screen == browse.ScreenRoot {
			a.exitOverlay()
		} else {
			a.browseState.Screen = browse.ScreenRoot
			a.drawBrowse()
		}
	case 'h':
		browse.ClearSearch(&a.browseState)
		switch a.browseState.Screen {
		case browse.ScreenRoot:
			a.exitOverlay()
		case browse.ScreenPlaylists, browse.ScreenAlbums:
			a.browseState.Screen = browse.ScreenRoot
			a.drawBrowse()
		case browse.ScreenFolder:
			browse.ExitFolder(&a.browseState)
			a.drawBrowse()
		case browse.ScreenAlbumTracks:
			a.browseState.Screen = browse.ScreenAlbums
			a.drawBrowse()
		case browse.ScreenCatalogResults:
			a.browseState.Screen = browse.ScreenRoot
			a.drawBrowse()
		}
	case 'j':
		browse.MoveCursor(&a.browseState, 1)
		a.drawBrowse()
	case 'k':
		browse.MoveCursor(&a.browseState, -1)
		a.drawBrowse()
	case '\r', '\n', 'l':
		a.handleBrowseSelect()
	case 'a':
		if a.browseState.Screen != browse.ScreenRoot && a.browseState.Screen != browse.ScreenCatalogResults && !a.browseState.SearchActive {
			a.browseState.Sort = browse.SortAsc
			*a.browseState.CurrentList() = browse.ListState{}
			a.drawBrowse()
		}
	case 'd':
		if a.browseState.Screen != browse.ScreenRoot && a.browseState.Screen != browse.ScreenCatalogResults && !a.browseState.SearchActive {
			a.browseState.Sort = browse.SortDesc
			*a.browseState.CurrentList() = browse.ListState{}
			a.drawBrowse()
		}
	case '/':
		if a.browseState.Screen == browse.ScreenCatalogResults {
			a.browseState.Screen = browse.ScreenCatalogSearch
			a.browseState.CatalogQuery = ""
			a.drawBrowse()
		} else if a.browseState.Screen != browse.ScreenRoot && !a.browseState.SearchActive {
			a.browseState.SearchActive = true
			a.browseState.SearchQuery = ""
			a.drawBrowse()
		}
	}
}

// handleBrowseSearchKey handles keys while browse search input is active.
func (a *app) handleBrowseSearchKey(key byte) {
	switch key {
	case 27: // Escape — clear search
		browse.ClearSearch(&a.browseState)
		*a.browseState.CurrentList() = browse.ListState{}
		a.drawBrowse()
	case '\r', '\n': // Enter — confirm search
		a.browseState.SearchActive = false
		a.drawBrowse()
	case 127, 8: // Backspace
		if len(a.browseState.SearchQuery) > 0 {
			a.browseState.SearchQuery = a.browseState.SearchQuery[:len(a.browseState.SearchQuery)-1]
			*a.browseState.CurrentList() = browse.ListState{}
			a.drawBrowse()
		}
	default:
		if key >= 32 && key <= 126 {
			a.browseState.SearchQuery += string(rune(key))
			*a.browseState.CurrentList() = browse.ListState{}
			a.drawBrowse()
		}
	}
}

// handleCatalogSearchKey handles keys while the catalog search input is active.
func (a *app) handleCatalogSearchKey(key byte) {
	switch key {
	case 27: // Escape — go back
		a.browseState.Screen = browse.ScreenRoot
		a.drawBrowse()
	case '\r', '\n': // Enter — run search
		if a.browseState.CatalogQuery != "" {
			a.browseState.Screen = browse.ScreenCatalogResults
			a.browseState.CatalogView = browse.ListState{}
			a.browseState.CatalogResults = nil
			a.fetchCatalog(a.browseState.CatalogQuery)
			a.drawBrowse()
		}
	case 127, 8: // Backspace
		if len(a.browseState.CatalogQuery) > 0 {
			a.browseState.CatalogQuery = a.browseState.CatalogQuery[:len(a.browseState.CatalogQuery)-1]
			a.drawBrowse()
		}
	default:
		if key >= 32 && key <= 126 {
			a.browseState.CatalogQuery += string(rune(key))
			a.drawBrowse()
		}
	}
}

// handleBrowseSelect handles enter/select in the browse modal.
func (a *app) handleBrowseSelect() {
	switch a.browseState.Screen {
	case browse.ScreenRoot:
		switch a.browseState.RootView.Cursor {
		case browse.RootItemPlaylists:
			browse.ClearSearch(&a.browseState)
			a.browseState.Screen = browse.ScreenPlaylists
			a.browseState.PlaylistsView = browse.ListState{}
			a.browseState.FolderStack = nil
			a.browseState.FolderID = ""
			a.fetchPlaylists()
			a.drawBrowse()
		case browse.RootItemLibrary:
			browse.ClearSearch(&a.browseState)
			a.browseState.Screen = browse.ScreenAlbums
			a.browseState.AlbumsView = browse.ListState{}
			a.fetchAlbums()
			a.drawBrowse()
		case browse.RootItemCatalog:
			a.browseState.Screen = browse.ScreenCatalogSearch
			a.browseState.CatalogQuery = ""
			a.drawBrowse()
		}
	case browse.ScreenPlaylists:
		if a.browseState.PlaylistsLoading {
			return
		}
		p, ok := browse.SelectedPlaylist(&a.browseState)
		if !ok {
			return
		}
		if p.IsFolder() {
			browse.ClearSearch(&a.browseState)
			browse.EnterFolder(&a.browseState, p)
			a.drawBrowse()
		} else {
			go func() { _ = music.PlayPlaylist(a.ctx, p.PersistentID) }()
			a.exitOverlay()
			ui.SafeReset(a.refresh, 500*time.Millisecond)
		}
	case browse.ScreenFolder:
		p, ok := browse.SelectedFolderItem(&a.browseState)
		if !ok {
			return
		}
		if p.IsFolder() {
			browse.ClearSearch(&a.browseState)
			browse.EnterFolder(&a.browseState, p)
			a.drawBrowse()
		} else {
			go func() { _ = music.PlayPlaylist(a.ctx, p.PersistentID) }()
			a.exitOverlay()
			ui.SafeReset(a.refresh, 500*time.Millisecond)
		}
	case browse.ScreenAlbums:
		if a.browseState.AlbumsLoading {
			return
		}
		al, ok := browse.SelectedAlbum(&a.browseState)
		if !ok {
			return
		}
		a.browseState.SelectedAlbum = &al
		a.browseState.Screen = browse.ScreenAlbumTracks
		a.browseState.TracksView = browse.ListState{}
		a.browseState.AlbumTracks = nil
		a.fetchAlbumTracks(&al)
		a.drawBrowse()
	case browse.ScreenAlbumTracks:
		if a.browseState.TracksLoading || a.browseState.SelectedAlbum == nil {
			return
		}
		al := a.browseState.SelectedAlbum
		go func() { _ = music.PlayAlbum(a.ctx, al.Name, al.AlbumArtist) }()
		a.exitOverlay()
		ui.SafeReset(a.refresh, 500*time.Millisecond)
	case browse.ScreenCatalogResults:
		if a.browseState.CatalogLoading {
			return
		}
		idx := a.browseState.CatalogView.Cursor
		if idx < len(a.browseState.CatalogResults) {
			song := a.browseState.CatalogResults[idx]
			go func() { _ = music.PlayCatalogTrack(a.ctx, song.TrackID) }()
			a.exitOverlay()
			ui.SafeReset(a.refresh, 1500*time.Millisecond)
		}
	}
}

// handleNowPlayingKey handles keys in the normal now-playing view.
// Returns true if the app should exit.
func (a *app) handleNowPlayingKey(key byte) bool {
	switch key {
	case 'x':
		a.quit()
		return true
	case ' ':
		go func() { _ = music.PlayPause(a.ctx) }()
		if a.latestInfo != nil {
			a.latestInfo.Playing = !a.latestInfo.Playing
		}
		a.flashIcon = "play"
		a.redrawControls()
		ui.SafeReset(a.flashTimer, 500*time.Millisecond)
	case 'l':
		go func() { _ = music.NextTrack(a.ctx) }()
		a.flashIcon = "next"
		a.redrawControls()
		ui.SafeReset(a.flashTimer, 500*time.Millisecond)
	case 'h':
		if a.latestInfo != nil && a.latestInfo.Position > ui.RestartThreshold {
			go func() { _ = music.RestartTrack(a.ctx) }()
		} else {
			go func() { _ = music.PreviousTrack(a.ctx) }()
		}
		a.flashIcon = "prev"
		a.redrawControls()
		ui.SafeReset(a.flashTimer, 500*time.Millisecond)
	case 'j':
		_ = volume.Down()
		a.flashIcon = "voldn"
		a.drawVolume()
		ui.SafeReset(a.flashTimer, 500*time.Millisecond)
	case 'k':
		_ = volume.Up()
		a.flashIcon = "volup"
		a.drawVolume()
		ui.SafeReset(a.flashTimer, 500*time.Millisecond)
	case 's':
		go func() {
			if on, err := music.ToggleShuffle(a.ctx); err == nil {
				a.shuffleOn = on
				a.redrawControls()
			}
		}()
	case 'r':
		go func() {
			if mode, err := music.CycleRepeat(a.ctx); err == nil {
				a.repeatMode = mode
				a.redrawControls()
			}
		}()
	case 'q':
		a.overlayMode = "queue"
		a.queueScroll = 0
		a.queueLoading = true
		a.spinnerFrame = 0
		if !a.spaceAllocated {
			a.allocateSpace()
		}
		a.drawQueue()
		a.spinnerTicker.Reset(100 * time.Millisecond)
		go func() {
			tracks, _ := music.Queue(a.ctx)
			a.queueCh <- tracks
		}()
	case 'b':
		a.openBrowse(false)
	}
	ui.SafeReset(a.refresh, 300*time.Millisecond)
	return false
}
