package main

import (
	"fmt"
	"strings"

	"github.com/thatsneat-dev/muzak/internal/music"
)

// browseScreen identifies the current screen within the browse modal.
type browseScreen uint8

const (
	browseRoot browseScreen = iota
	browsePlaylists
	browseAlbums
	browseAlbumTracks
	browseFolder
)

// listState tracks cursor and scroll position for a scrollable list.
type listState struct {
	Cursor int
	Scroll int
}

// browseState holds all state for the browse modal.
type browseState struct {
	Screen browseScreen

	RootView      listState
	PlaylistsView listState
	AlbumsView    listState
	TracksView    listState
	FolderView    listState

	Playlists []music.Playlist
	Albums    []music.Album

	SelectedAlbum  *music.Album
	AlbumTracks    []music.AlbumTrack
	AlbumTrackCache map[string][]music.AlbumTrack

	// Folder navigation: stack of parent IDs for back-navigation.
	FolderStack   []string // stack of parent folder IDs
	FolderID      string   // current folder's persistent ID (empty = top-level)
	FolderName    string   // current folder's display name
	FolderItems   []music.Playlist // filtered playlists in the current folder

	PlaylistsLoading bool
	PlaylistsLoaded  bool
	AlbumsLoading    bool
	AlbumsLoaded     bool
	TracksLoading    bool

	AutoOpened bool
}

const (
	rootItemPlaylists = 0
	rootItemLibrary   = 1
	rootItemCount     = 2
)

// browseTitle returns the title for the current screen.
func (b *browseState) browseTitle() string {
	switch b.Screen {
	case browsePlaylists:
		return "Playlists"
	case browseFolder:
		if b.FolderName != "" {
			return b.FolderName
		}
		return "Folder"
	case browseAlbums:
		return "Library"
	case browseAlbumTracks:
		if b.SelectedAlbum != nil {
			return b.SelectedAlbum.Name
		}
		return "Tracks"
	default:
		return "Browse"
	}
}

// browseSubtitle returns an optional subtitle line for the current screen.
func (b *browseState) browseSubtitle() string {
	if b.Screen == browseAlbumTracks && b.SelectedAlbum != nil {
		return b.SelectedAlbum.AlbumArtist
	}
	return ""
}

// currentList returns the active list state for the current screen.
func (b *browseState) currentList() *listState {
	switch b.Screen {
	case browsePlaylists:
		return &b.PlaylistsView
	case browseFolder:
		return &b.FolderView
	case browseAlbums:
		return &b.AlbumsView
	case browseAlbumTracks:
		return &b.TracksView
	default:
		return &b.RootView
	}
}

// itemCount returns the number of items on the current screen.
func (b *browseState) itemCount() int {
	switch b.Screen {
	case browsePlaylists:
		return len(filterPlaylistsByParent(b.Playlists, ""))
	case browseFolder:
		return len(b.FolderItems)
	case browseAlbums:
		return len(b.Albums)
	case browseAlbumTracks:
		return len(b.AlbumTracks)
	default:
		return rootItemCount
	}
}

// isLoading returns whether the current screen is loading data.
func (b *browseState) isLoading() bool {
	switch b.Screen {
	case browsePlaylists, browseFolder:
		return b.PlaylistsLoading
	case browseAlbums:
		return b.AlbumsLoading
	case browseAlbumTracks:
		return b.TracksLoading
	default:
		return false
	}
}

// albumCacheKey returns a cache key for an album.
func albumCacheKey(name, artist string) string {
	return name + "\x00" + artist
}

// filterPlaylistsByParent returns playlists whose ParentID matches the given id.
// For the top-level playlists view, pass "" to get playlists with no parent.
func filterPlaylistsByParent(all []music.Playlist, parentID string) []music.Playlist {
	var result []music.Playlist
	for _, p := range all {
		if p.ParentID == parentID {
			result = append(result, p)
		}
	}
	return result
}

// enterFolder sets up the browse state to display a folder's contents.
func enterFolder(b *browseState, folder music.Playlist) {
	b.FolderStack = append(b.FolderStack, b.FolderID)
	b.FolderID = folder.PersistentID
	b.FolderName = folder.Name
	b.FolderItems = filterPlaylistsByParent(b.Playlists, folder.PersistentID)
	b.FolderView = listState{}
	b.Screen = browseFolder
}

// exitFolder pops back to the parent folder or the top-level playlists view.
func exitFolder(b *browseState) {
	if len(b.FolderStack) == 0 {
		b.Screen = browsePlaylists
		return
	}
	parentID := b.FolderStack[len(b.FolderStack)-1]
	b.FolderStack = b.FolderStack[:len(b.FolderStack)-1]
	b.FolderID = parentID
	if parentID == "" {
		b.Screen = browsePlaylists
	} else {
		// Find the parent folder name.
		for _, p := range b.Playlists {
			if p.PersistentID == parentID {
				b.FolderName = p.Name
				break
			}
		}
		b.FolderItems = filterPlaylistsByParent(b.Playlists, parentID)
		b.FolderView = listState{}
	}
}

// drawBrowse renders the browse modal.
// drawLine writes a single line at the given row/col (same as drawModalLine in main).
func drawBrowse(b *browseState, spinnerFrame int, artwork bool, drawLine func(row, col int, s string)) {
	modalCol := padLeft
	if artwork {
		modalCol = artworkLeft + artworkCols + 2
	}
	modalWidth := 50
	if terminalCols > modalCol+2 {
		avail := terminalCols - modalCol - 2
		if avail < modalWidth {
			modalWidth = avail
		}
	}
	if modalWidth < 20 {
		modalWidth = 20
	}
	innerWidth := modalWidth - 4 // borders + padding

	visibleRows := artworkRows - 2 // reserve top/bottom borders
	if visibleRows < 1 {
		visibleRows = 1
	}

	// Title bar.
	title := b.browseTitle()
	subtitle := b.browseSubtitle()
	titleLen := len(title)
	pad := modalWidth - 2 - titleLen - 2
	if pad < 0 {
		pad = 0
	}
	drawLine(padTop, modalCol,
		"┌ \x1b[1m"+title+"\x1b[0m "+strings.Repeat("─", pad)+"┐")

	list := b.currentList()
	count := b.itemCount()

	// Ensure cursor/scroll are in bounds.
	if list.Cursor >= count && count > 0 {
		list.Cursor = count - 1
	}
	if list.Scroll > list.Cursor {
		list.Scroll = list.Cursor
	}
	if list.Cursor >= list.Scroll+visibleRows {
		list.Scroll = list.Cursor - visibleRows + 1
	}

	// Content rows.
	subtitleRow := -1
	contentStart := 0
	if subtitle != "" {
		subtitleRow = 0
		contentStart = 1
		visibleRows-- // one less row for content
		if visibleRows < 1 {
			visibleRows = 1
		}
	}

	totalRows := visibleRows + contentStart
	for i := range totalRows {
		var content string

		if i == subtitleRow {
			content = "\x1b[2m" + truncateVisible(subtitle, innerWidth) + "\x1b[0m"
		} else if b.isLoading() && i == contentStart {
			spinnerChars := [4]string{"⠋", "⠙", "⠹", "⠸"}
			content = spinnerChars[spinnerFrame%len(spinnerChars)] + " Loading…"
		} else if !b.isLoading() && count == 0 && i == contentStart {
			content = browseEmptyMessage(b.Screen)
		} else {
			idx := list.Scroll + (i - contentStart)
			if idx >= 0 && idx < count {
				label := browseItemLabel(b, idx)
				cursor := "  "
				if idx == list.Cursor {
					cursor = "\x1b[33m▸\x1b[0m "
				}
				content = cursor + truncateVisible(label, innerWidth-2)
			}
		}

		visible := visibleLen(content)
		trailing := innerWidth - visible
		if trailing < 0 {
			trailing = 0
		}
		drawLine(padTop+1+i, modalCol,
			"│ "+content+strings.Repeat(" ", trailing)+" │")
	}

	// Bottom border with position info.
	bottom := strings.Repeat("─", modalWidth-2)
	if !b.isLoading() && count > 0 {
		pos := fmt.Sprintf(" %d/%d ", list.Cursor+1, count)
		if len(pos) < modalWidth-2 {
			bottom = strings.Repeat("─", modalWidth-2-len(pos)) + pos
		}
	}
	drawLine(padTop+1+totalRows, modalCol,
		"└"+bottom+"┘")

	// Hint line below the modal.
	hint := "\x1b[2mj/k move  enter select  h back  q close\x1b[0m"
	drawLine(padTop+2+totalRows, modalCol, truncateVisible(hint, modalWidth))
}

const iconFolder = "\U000F024B" // 󰉋 nf-md-folder

// playlistLabel formats a playlist item, showing a folder icon for folders.
func playlistLabel(p music.Playlist) string {
	if p.IsFolder() {
		return iconFolder + "  " + p.Name
	}
	return fmt.Sprintf("%s  \x1b[2m(%d)\x1b[0m", p.Name, p.TrackCount)
}

// browseItemLabel returns the display label for an item at the given index.
func browseItemLabel(b *browseState, idx int) string {
	switch b.Screen {
	case browsePlaylists:
		items := filterPlaylistsByParent(b.Playlists, "")
		if idx < len(items) {
			return playlistLabel(items[idx])
		}
	case browseFolder:
		if idx < len(b.FolderItems) {
			return playlistLabel(b.FolderItems[idx])
		}
	case browseAlbums:
		if idx < len(b.Albums) {
			a := b.Albums[idx]
			return fmt.Sprintf("%s — %s", a.Name, a.AlbumArtist)
		}
	case browseAlbumTracks:
		if idx < len(b.AlbumTracks) {
			t := b.AlbumTracks[idx]
			return fmt.Sprintf("%2d. %s", t.TrackNumber, t.Name)
		}
	default:
		// Root items.
		switch idx {
		case rootItemPlaylists:
			return "Playlists"
		case rootItemLibrary:
			return "Library"
		}
	}
	return ""
}

// browseEmptyMessage returns the empty-state text for a screen.
func browseEmptyMessage(screen browseScreen) string {
	switch screen {
	case browsePlaylists:
		return "No playlists found."
	case browseFolder:
		return "Folder is empty."
	case browseAlbums:
		return "No albums in library."
	case browseAlbumTracks:
		return "No tracks found."
	default:
		return ""
	}
}

// visibleLen counts the visible (non-ANSI-escape) character count of a string.
func visibleLen(s string) int {
	var visible int
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
		visible++
	}
	return visible
}

// browseMoveCursor moves the cursor up or down within the current list.
func browseMoveCursor(b *browseState, delta int) {
	list := b.currentList()
	count := b.itemCount()
	if count == 0 {
		return
	}
	list.Cursor += delta
	if list.Cursor < 0 {
		list.Cursor = 0
	}
	if list.Cursor >= count {
		list.Cursor = count - 1
	}
}
