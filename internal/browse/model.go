package browse

import (
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/thatsneat-dev/muzak/internal/model"
)

// AlbumCacheKey returns a cache key for an album.
func AlbumCacheKey(name, artist string) string {
	return name + "\x00" + artist
}

// FilterPlaylistsByParent returns playlists whose ParentID matches the given id.
func FilterPlaylistsByParent(all []model.Playlist, parentID string) []model.Playlist {
	var result []model.Playlist
	for _, p := range all {
		if p.ParentID == parentID {
			result = append(result, p)
		}
	}
	return result
}

// FuzzyMatch reports whether query matches target using subsequence matching.
func FuzzyMatch(target, query string) bool {
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

// VisiblePlaylists returns top-level playlists filtered and sorted per browse state.
func VisiblePlaylists(b *State) []model.Playlist {
	items := FilterPlaylistsByParent(b.Playlists, "")
	if b.SearchQuery != "" {
		var filtered []model.Playlist
		for _, p := range items {
			if FuzzyMatch(p.Name, b.SearchQuery) {
				filtered = append(filtered, p)
			}
		}
		items = filtered
	}
	return SortPlaylists(items, b.Sort)
}

// VisibleFolderItems returns folder items filtered and sorted per browse state.
func VisibleFolderItems(b *State) []model.Playlist {
	items := b.FolderItems
	if b.SearchQuery != "" {
		var filtered []model.Playlist
		for _, p := range items {
			if FuzzyMatch(p.Name, b.SearchQuery) {
				filtered = append(filtered, p)
			}
		}
		items = filtered
	}
	return SortPlaylists(items, b.Sort)
}

// VisibleAlbums returns albums filtered and sorted per browse state.
func VisibleAlbums(b *State) []model.Album {
	items := b.Albums
	if b.SearchQuery != "" {
		var filtered []model.Album
		for _, a := range items {
			if FuzzyMatch(a.Name, b.SearchQuery) || FuzzyMatch(a.AlbumArtist, b.SearchQuery) {
				filtered = append(filtered, a)
			}
		}
		items = filtered
	}
	return SortAlbums(items, b.Sort)
}

// VisibleAlbumTracks returns album tracks filtered and sorted per browse state.
func VisibleAlbumTracks(b *State) []model.AlbumTrack {
	items := b.AlbumTracks
	if b.SearchQuery != "" {
		var filtered []model.AlbumTrack
		for _, t := range items {
			if FuzzyMatch(t.Name, b.SearchQuery) {
				filtered = append(filtered, t)
			}
		}
		items = filtered
	}
	return SortAlbumTracks(items, b.Sort)
}

// SortPlaylists returns a sorted copy of playlists.
func SortPlaylists(items []model.Playlist, order SortOrder) []model.Playlist {
	if order == SortNone || len(items) == 0 {
		return items
	}
	out := make([]model.Playlist, len(items))
	copy(out, items)
	sort.Slice(out, func(i, j int) bool {
		if order == SortDesc {
			return strings.ToLower(out[i].Name) > strings.ToLower(out[j].Name)
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

// SortAlbums returns a sorted copy of albums.
func SortAlbums(items []model.Album, order SortOrder) []model.Album {
	if order == SortNone || len(items) == 0 {
		return items
	}
	out := make([]model.Album, len(items))
	copy(out, items)
	sort.Slice(out, func(i, j int) bool {
		ni, nj := strings.ToLower(out[i].Name), strings.ToLower(out[j].Name)
		if ni != nj {
			if order == SortDesc {
				return ni > nj
			}
			return ni < nj
		}
		ai, aj := strings.ToLower(out[i].AlbumArtist), strings.ToLower(out[j].AlbumArtist)
		if order == SortDesc {
			return ai > aj
		}
		return ai < aj
	})
	return out
}

// SortAlbumTracks returns a sorted copy of album tracks.
func SortAlbumTracks(items []model.AlbumTrack, order SortOrder) []model.AlbumTrack {
	if order == SortNone || len(items) == 0 {
		return items
	}
	out := make([]model.AlbumTrack, len(items))
	copy(out, items)
	sort.Slice(out, func(i, j int) bool {
		ni, nj := strings.ToLower(out[i].Name), strings.ToLower(out[j].Name)
		if order == SortDesc {
			return ni > nj
		}
		return ni < nj
	})
	return out
}

// MoveCursor moves the cursor up or down within the current list.
func MoveCursor(b *State, delta int) {
	list := b.CurrentList()
	count := b.ItemCount()
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

// SelectedPlaylist returns the playlist at the cursor on the playlists screen.
func SelectedPlaylist(b *State) (model.Playlist, bool) {
	items := VisiblePlaylists(b)
	if b.PlaylistsView.Cursor < len(items) {
		return items[b.PlaylistsView.Cursor], true
	}
	return model.Playlist{}, false
}

// SelectedFolderItem returns the playlist at the cursor on the folder screen.
func SelectedFolderItem(b *State) (model.Playlist, bool) {
	items := VisibleFolderItems(b)
	if b.FolderView.Cursor < len(items) {
		return items[b.FolderView.Cursor], true
	}
	return model.Playlist{}, false
}

// SelectedAlbum returns the album at the cursor on the albums screen.
func SelectedAlbum(b *State) (model.Album, bool) {
	items := VisibleAlbums(b)
	if b.AlbumsView.Cursor < len(items) {
		return items[b.AlbumsView.Cursor], true
	}
	return model.Album{}, false
}

// EnterFolder sets up the browse state to display a folder's contents.
func EnterFolder(b *State, folder model.Playlist) {
	b.FolderStack = append(b.FolderStack, b.FolderID)
	b.FolderID = folder.PersistentID
	b.FolderName = folder.Name
	b.FolderItems = FilterPlaylistsByParent(b.Playlists, folder.PersistentID)
	b.FolderView = ListState{}
	b.Screen = ScreenFolder
}

// ExitFolder pops back to the parent folder or the top-level playlists view.
func ExitFolder(b *State) {
	if len(b.FolderStack) == 0 {
		b.Screen = ScreenPlaylists
		return
	}
	parentID := b.FolderStack[len(b.FolderStack)-1]
	b.FolderStack = b.FolderStack[:len(b.FolderStack)-1]
	b.FolderID = parentID
	if parentID == "" {
		b.Screen = ScreenPlaylists
	} else {
		for _, p := range b.Playlists {
			if p.PersistentID == parentID {
				b.FolderName = p.Name
				break
			}
		}
		b.FolderItems = FilterPlaylistsByParent(b.Playlists, parentID)
		b.FolderView = ListState{}
	}
}

// ClearSearch resets search state and cursor position.
func ClearSearch(b *State) {
	b.SearchQuery = ""
	b.SearchActive = false
}
