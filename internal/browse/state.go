package browse

import (
	"github.com/thatsneat-dev/muzak/internal/catalog"
	"github.com/thatsneat-dev/muzak/internal/music"
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

const (
	RootItemPlaylists = 0
	RootItemLibrary   = 1
	RootItemCatalog   = 2
	RootItemCount     = 3
)

// State holds all state for the browse modal.
type State struct {
	Screen Screen

	RootView      ListState
	PlaylistsView ListState
	AlbumsView    ListState
	TracksView    ListState
	FolderView    ListState

	Playlists []music.Playlist
	Albums    []music.Album

	SelectedAlbum   *music.Album
	AlbumTracks     []music.AlbumTrack
	AlbumTrackCache map[string][]music.AlbumTrack

	FolderStack []string
	FolderID    string
	FolderName  string
	FolderItems []music.Playlist

	PlaylistsLoading bool
	PlaylistsLoaded  bool
	AlbumsLoading    bool
	AlbumsLoaded     bool
	TracksLoading    bool

	CatalogView    ListState
	CatalogQuery   string
	CatalogResults []catalog.Song
	CatalogLoading bool

	AutoOpened bool

	Sort         SortOrder
	SearchQuery  string
	SearchActive bool
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
