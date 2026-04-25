// Package browse implements the browse modal state machine for navigating
// playlists, albums, album tracks, and catalog search results within the
// muzak TUI.
package browse

import (
	"github.com/thatsneat-dev/muzak/internal/model"
)

// Screen identifies the current screen within the browse modal.
type Screen uint8

const (
	ScreenRoot Screen = iota
	ScreenPlaylists
	ScreenAlbums
	ScreenAlbumTracks
	ScreenFolder
	ScreenCatalogSearch
	ScreenCatalogResults
)

// SortOrder controls the sorting of browse lists.
type SortOrder uint8

const (
	SortNone SortOrder = iota
	SortAsc
	SortDesc
)

// ListState tracks cursor and scroll position for a scrollable list.
type ListState struct {
	Cursor int
	Scroll int
}

// Root menu item indices for the browse modal's top-level screen.
const (
	RootItemPlaylists = 0 // Playlists entry.
	RootItemLibrary   = 1 // Library entry.
	RootItemCatalog   = 2 // Search Catalog entry.
	RootItemCount     = 3 // Total number of root menu items.
)

// State holds all state for the browse modal.
type State struct {
	Screen Screen // Currently active browse screen.

	RootView      ListState // Cursor state for the root menu.
	PlaylistsView ListState // Cursor state for the playlists screen.
	AlbumsView    ListState // Cursor state for the albums screen.
	TracksView    ListState // Cursor state for the album tracks screen.
	FolderView    ListState // Cursor state for the folder screen.

	Playlists []model.Playlist // Full list of user playlists.
	Albums    []model.Album    // Full list of library albums.

	SelectedAlbum   *model.Album                  // Album currently being viewed.
	AlbumTracks     []model.AlbumTrack            // Tracks for the selected album.
	AlbumTrackCache map[string][]model.AlbumTrack // Cached album tracks by cache key.

	FolderStack []string         // Parent folder IDs for back navigation.
	FolderID    string           // Persistent ID of the current folder.
	FolderName  string           // Display name of the current folder.
	FolderItems []model.Playlist // Contents of the current folder.

	PlaylistsLoading bool // Playlists are being fetched.
	PlaylistsLoaded  bool // Playlists have been fetched at least once.
	AlbumsLoading    bool // Albums are being fetched.
	AlbumsLoaded     bool // Albums have been fetched at least once.
	TracksLoading    bool // Album tracks are being fetched.

	CatalogView    ListState    // Cursor state for catalog results.
	CatalogQuery   string       // Current catalog search input.
	CatalogResults []model.Song // Search results from the iTunes catalog.
	CatalogLoading bool         // Catalog search is in progress.

	AutoOpened bool // Browse modal opened automatically (no track playing).

	Sort         SortOrder // Current sort order for list screens.
	SearchQuery  string    // Current fuzzy search input.
	SearchActive bool      // Search input is focused.
}

// CurrentList returns the active list state for the current screen.
func (b *State) CurrentList() *ListState {
	switch b.Screen {
	case ScreenPlaylists:
		return &b.PlaylistsView
	case ScreenFolder:
		return &b.FolderView
	case ScreenAlbums:
		return &b.AlbumsView
	case ScreenAlbumTracks:
		return &b.TracksView
	case ScreenCatalogResults:
		return &b.CatalogView
	default:
		return &b.RootView
	}
}

// ItemCount returns the number of items on the current screen.
func (b *State) ItemCount() int {
	switch b.Screen {
	case ScreenPlaylists:
		return len(VisiblePlaylists(b))
	case ScreenFolder:
		return len(VisibleFolderItems(b))
	case ScreenAlbums:
		return len(VisibleAlbums(b))
	case ScreenAlbumTracks:
		return len(VisibleAlbumTracks(b))
	case ScreenCatalogResults:
		return len(b.CatalogResults)
	default:
		return RootItemCount
	}
}

// IsLoading returns whether the current screen is loading data.
func (b *State) IsLoading() bool {
	switch b.Screen {
	case ScreenPlaylists, ScreenFolder:
		return b.PlaylistsLoading
	case ScreenAlbums:
		return b.AlbumsLoading
	case ScreenAlbumTracks:
		return b.TracksLoading
	case ScreenCatalogResults:
		return b.CatalogLoading
	default:
		return false
	}
}
