package main

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
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

// sortOrder controls the sorting of browse lists.
type sortOrder uint8

const (
	sortNone sortOrder = iota
	sortAsc
	sortDesc
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

	Sort         sortOrder
	SearchQuery  string
	SearchActive bool
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
		return iconPlaylistMusic + " Playlists"
	case browseFolder:
		if b.FolderName != "" {
			return iconFolder + " " + b.FolderName
		}
		return iconFolder + " Folder"
	case browseAlbums:
		return iconLibrary + " Library"
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
		return len(visiblePlaylists(b))
	case browseFolder:
		return len(visibleFolderItems(b))
	case browseAlbums:
		return len(visibleAlbums(b))
	case browseAlbumTracks:
		return len(visibleAlbumTracks(b))
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

// fuzzyMatch reports whether query matches target using subsequence matching.
// Both are compared case-insensitively.
func fuzzyMatch(target, query string) bool {
	if query == "" {
		return true
	}
	target = strings.ToLower(target)
	query = strings.ToLower(query)
	qi := 0
	for _, r := range target {
		qr, _ := utf8.DecodeRuneInString(query[qi:])
		if r == qr {
			qi += utf8.RuneLen(qr)
			if qi >= len(query) {
				return true
			}
		}
	}
	return false
}

// visiblePlaylists returns top-level playlists filtered and sorted per browse state.
func visiblePlaylists(b *browseState) []music.Playlist {
	items := filterPlaylistsByParent(b.Playlists, "")
	if b.SearchQuery != "" {
		var filtered []music.Playlist
		for _, p := range items {
			if fuzzyMatch(p.Name, b.SearchQuery) {
				filtered = append(filtered, p)
			}
		}
		items = filtered
	}
	return sortPlaylists(items, b.Sort)
}

// visibleFolderItems returns folder items filtered and sorted per browse state.
func visibleFolderItems(b *browseState) []music.Playlist {
	items := b.FolderItems
	if b.SearchQuery != "" {
		var filtered []music.Playlist
		for _, p := range items {
			if fuzzyMatch(p.Name, b.SearchQuery) {
				filtered = append(filtered, p)
			}
		}
		items = filtered
	}
	return sortPlaylists(items, b.Sort)
}

// visibleAlbums returns albums filtered and sorted per browse state.
func visibleAlbums(b *browseState) []music.Album {
	items := b.Albums
	if b.SearchQuery != "" {
		var filtered []music.Album
		for _, a := range items {
			if fuzzyMatch(a.Name, b.SearchQuery) || fuzzyMatch(a.AlbumArtist, b.SearchQuery) {
				filtered = append(filtered, a)
			}
		}
		items = filtered
	}
	return sortAlbums(items, b.Sort)
}

// visibleAlbumTracks returns album tracks filtered and sorted per browse state.
func visibleAlbumTracks(b *browseState) []music.AlbumTrack {
	items := b.AlbumTracks
	if b.SearchQuery != "" {
		var filtered []music.AlbumTrack
		for _, t := range items {
			if fuzzyMatch(t.Name, b.SearchQuery) {
				filtered = append(filtered, t)
			}
		}
		items = filtered
	}
	return sortAlbumTracks(items, b.Sort)
}

func sortPlaylists(items []music.Playlist, order sortOrder) []music.Playlist {
	if order == sortNone || len(items) == 0 {
		return items
	}
	out := make([]music.Playlist, len(items))
	copy(out, items)
	sort.Slice(out, func(i, j int) bool {
		if order == sortDesc {
			return strings.ToLower(out[i].Name) > strings.ToLower(out[j].Name)
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func sortAlbums(items []music.Album, order sortOrder) []music.Album {
	if order == sortNone || len(items) == 0 {
		return items
	}
	out := make([]music.Album, len(items))
	copy(out, items)
	sort.Slice(out, func(i, j int) bool {
		ni, nj := strings.ToLower(out[i].Name), strings.ToLower(out[j].Name)
		if ni != nj {
			if order == sortDesc {
				return ni > nj
			}
			return ni < nj
		}
		ai, aj := strings.ToLower(out[i].AlbumArtist), strings.ToLower(out[j].AlbumArtist)
		if order == sortDesc {
			return ai > aj
		}
		return ai < aj
	})
	return out
}

func sortAlbumTracks(items []music.AlbumTrack, order sortOrder) []music.AlbumTrack {
	if order == sortNone || len(items) == 0 {
		return items
	}
	out := make([]music.AlbumTrack, len(items))
	copy(out, items)
	sort.Slice(out, func(i, j int) bool {
		ni, nj := strings.ToLower(out[i].Name), strings.ToLower(out[j].Name)
		if order == sortDesc {
			return ni > nj
		}
		return ni < nj
	})
	return out
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
	titleLen := visibleLen(title)
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

	// Search bar above the modal (render at padTop-1 if search is active).
	if b.SearchActive || b.SearchQuery != "" {
		searchLine := "\x1b[33m" + iconSearch + "  " + b.SearchQuery
		if b.SearchActive {
			searchLine += "█"
		}
		searchLine = truncateVisible(searchLine, modalWidth)
		searchLine += "\x1b[0m"
		pad := modalWidth - visibleLen(searchLine)
		if pad < 0 {
			pad = 0
		}
		drawLine(padTop-1, modalCol, searchLine+strings.Repeat(" ", pad))
	} else {
		drawLine(padTop-1, modalCol, strings.Repeat(" ", modalWidth))
	}

	// Hint lines below the modal.
	var hints []string
	if b.SearchActive {
		hints = []string{"[type] search", "[enter] confirm", "[esc] clear"}
	} else {
		hints = []string{"[j|k] move", "[/] search", "[a|d] sort", "[enter] select", "[h] back", "[x] close"}
	}
	drawHints(hints, modalWidth, padTop+2+totalRows, modalCol, drawLine)
}

const (
	iconFolder         = "\U000F024B" // 󰉋 nf-md-folder
	iconPlaylistMusic  = "\U000F0CB8" // 󰲸 nf-md-playlist_music
	iconLibrary        = "\uEB9C"     //  nf-cod-library
	iconSearch         = "\uF002"     //  nf-fa-search
)

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
		items := visiblePlaylists(b)
		if idx < len(items) {
			return playlistLabel(items[idx])
		}
	case browseFolder:
		items := visibleFolderItems(b)
		if idx < len(items) {
			return playlistLabel(items[idx])
		}
	case browseAlbums:
		items := visibleAlbums(b)
		if idx < len(items) {
			a := items[idx]
			return fmt.Sprintf("%s — %s", a.Name, a.AlbumArtist)
		}
	case browseAlbumTracks:
		items := visibleAlbumTracks(b)
		if idx < len(items) {
			t := items[idx]
			return fmt.Sprintf("%2d. %s", t.TrackNumber, t.Name)
		}
	default:
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

// visibleLen counts the visible terminal cell width of a string,
// correctly handling ANSI escapes and wide characters (emojis).
func visibleLen(s string) int {
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

// browseSelectedPlaylist returns the playlist at the cursor on the playlists screen.
func browseSelectedPlaylist(b *browseState) (music.Playlist, bool) {
	items := visiblePlaylists(b)
	if b.PlaylistsView.Cursor < len(items) {
		return items[b.PlaylistsView.Cursor], true
	}
	return music.Playlist{}, false
}

// browseSelectedFolderItem returns the playlist at the cursor on the folder screen.
func browseSelectedFolderItem(b *browseState) (music.Playlist, bool) {
	items := visibleFolderItems(b)
	if b.FolderView.Cursor < len(items) {
		return items[b.FolderView.Cursor], true
	}
	return music.Playlist{}, false
}

// browseSelectedAlbum returns the album at the cursor on the albums screen.
func browseSelectedAlbum(b *browseState) (music.Album, bool) {
	items := visibleAlbums(b)
	if b.AlbumsView.Cursor < len(items) {
		return items[b.AlbumsView.Cursor], true
	}
	return music.Album{}, false
}

// drawHints renders hint items below the modal, wrapping to two lines if needed.
// Items are joined with double-space separators and centered under the modal width.
func drawHints(items []string, modalWidth, row, col int, drawLine func(row, col int, s string)) {
	sep := "  "
	joined := strings.Join(items, sep)
	if len(joined) <= modalWidth {
		// Single line, centered.
		pad := modalWidth - len(joined)
		left := pad / 2
		line := "\x1b[2m" + strings.Repeat(" ", left) + joined + strings.Repeat(" ", pad-left) + "\x1b[0m"
		drawLine(row, col, line)
		drawLine(row+1, col, strings.Repeat(" ", modalWidth))
		return
	}

	// Split into two lines: greedily fill line 1, then line 2.
	var line1, line2 []string
	width1 := 0
	for i, item := range items {
		itemLen := len(item)
		needed := itemLen
		if width1 > 0 {
			needed += len(sep)
		}
		if width1+needed > modalWidth {
			line2 = items[i:]
			break
		}
		line1 = append(line1, item)
		width1 += needed
	}
	if len(line2) == 0 {
		line1 = items
	}

	for li, parts := range [][]string{line1, line2} {
		text := strings.Join(parts, sep)
		pad := modalWidth - len(text)
		if pad < 0 {
			pad = 0
		}
		left := pad / 2
		line := "\x1b[2m" + strings.Repeat(" ", left) + text + strings.Repeat(" ", pad-left) + "\x1b[0m"
		drawLine(row+li, col, line)
	}
}

// clearSearch resets search state and cursor position.
func clearSearch(b *browseState) {
	b.SearchQuery = ""
	b.SearchActive = false
}
