package browse_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thatsneat-dev/muzak/internal/browse"
	"github.com/thatsneat-dev/muzak/internal/model"
)

func TestFuzzyMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target string
		query  string
		want   bool
	}{
		{"empty query matches all", "anything", "", true},
		{"exact match", "Rock", "Rock", true},
		{"case insensitive", "Rock", "rock", true},
		{"prefix", "Album", "alb", true},
		{"substring chars in order", "Best Of Taylor", "bt", true},
		{"subsequence", "Discovery", "dsy", true},
		{"no match", "Jazz", "rock", false},
		{"query longer than target", "Hi", "Hello", false},
		{"empty target no query", "", "", true},
		{"empty target with query", "", "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, browse.FuzzyMatch(tt.target, tt.query))
		})
	}
}

func TestSortPlaylists(t *testing.T) {
	t.Parallel()

	items := []model.Playlist{
		{Name: "Chill"},
		{Name: "Ambient"},
		{Name: "Beats"},
	}

	t.Run("sortNone preserves order", func(t *testing.T) {
		t.Parallel()
		result := browse.SortPlaylists(items, browse.SortNone)
		assert.Equal(t, "Chill", result[0].Name)
	})

	t.Run("sortAsc", func(t *testing.T) {
		t.Parallel()
		result := browse.SortPlaylists(items, browse.SortAsc)
		assert.Equal(t, "Ambient", result[0].Name)
		assert.Equal(t, "Beats", result[1].Name)
		assert.Equal(t, "Chill", result[2].Name)
	})

	t.Run("sortDesc", func(t *testing.T) {
		t.Parallel()
		result := browse.SortPlaylists(items, browse.SortDesc)
		assert.Equal(t, "Chill", result[0].Name)
		assert.Equal(t, "Beats", result[1].Name)
		assert.Equal(t, "Ambient", result[2].Name)
	})

	t.Run("does not mutate original", func(t *testing.T) {
		t.Parallel()
		_ = browse.SortPlaylists(items, browse.SortAsc)
		assert.Equal(t, "Chill", items[0].Name)
	})
}

func TestSortAlbums(t *testing.T) {
	t.Parallel()

	items := []model.Album{
		{Name: "Zen", AlbumArtist: "A"},
		{Name: "Zen", AlbumArtist: "B"},
		{Name: "Alpha", AlbumArtist: "C"},
	}

	t.Run("sortAsc by name then artist", func(t *testing.T) {
		t.Parallel()
		result := browse.SortAlbums(items, browse.SortAsc)
		assert.Equal(t, "Alpha", result[0].Name)
		assert.Equal(t, "A", result[1].AlbumArtist)
		assert.Equal(t, "B", result[2].AlbumArtist)
	})

	t.Run("sortDesc", func(t *testing.T) {
		t.Parallel()
		result := browse.SortAlbums(items, browse.SortDesc)
		assert.Equal(t, "Zen", result[0].Name)
		assert.Equal(t, "B", result[0].AlbumArtist)
	})
}

func TestVisiblePlaylistsWithSearch(t *testing.T) {
	t.Parallel()

	b := &browse.State{
		Screen: browse.ScreenPlaylists,
		Playlists: []model.Playlist{
			{Name: "Rock Hits", ParentID: ""},
			{Name: "Jazz Vibes", ParentID: ""},
			{Name: "Rock Classics", ParentID: ""},
		},
		SearchQuery: "rock",
	}

	result := browse.VisiblePlaylists(b)
	assert.Len(t, result, 2)
	assert.Equal(t, "Rock Hits", result[0].Name)
	assert.Equal(t, "Rock Classics", result[1].Name)
}

func TestVisiblePlaylistsWithSortAndSearch(t *testing.T) {
	t.Parallel()

	b := &browse.State{
		Screen: browse.ScreenPlaylists,
		Playlists: []model.Playlist{
			{Name: "Rock Hits", ParentID: ""},
			{Name: "Jazz Vibes", ParentID: ""},
			{Name: "Rock Classics", ParentID: ""},
		},
		SearchQuery: "rock",
		Sort:        browse.SortAsc,
	}

	result := browse.VisiblePlaylists(b)
	assert.Len(t, result, 2)
	assert.Equal(t, "Rock Classics", result[0].Name)
	assert.Equal(t, "Rock Hits", result[1].Name)
}

func TestClearSearch(t *testing.T) {
	t.Parallel()

	b := &browse.State{
		SearchQuery:  "test",
		SearchActive: true,
	}
	browse.ClearSearch(b)
	assert.Empty(t, b.SearchQuery)
	assert.False(t, b.SearchActive)
}

func TestBrowseSelectedPlaylist(t *testing.T) {
	t.Parallel()

	b := &browse.State{
		Screen: browse.ScreenPlaylists,
		Playlists: []model.Playlist{
			{Name: "Alpha", ParentID: ""},
			{Name: "Beta", ParentID: ""},
		},
		PlaylistsView: browse.ListState{Cursor: 1},
	}

	p, ok := browse.SelectedPlaylist(b)
	assert.True(t, ok)
	assert.Equal(t, "Beta", p.Name)
}

func TestBrowseSelectedAlbum(t *testing.T) {
	t.Parallel()

	b := &browse.State{
		Screen: browse.ScreenAlbums,
		Albums: []model.Album{
			{Name: "Discovery", AlbumArtist: "Daft Punk"},
			{Name: "RAM", AlbumArtist: "Daft Punk"},
		},
		AlbumsView: browse.ListState{Cursor: 0},
	}

	a, ok := browse.SelectedAlbum(b)
	assert.True(t, ok)
	assert.Equal(t, "Discovery", a.Name)
}

func TestItemCountWithSearch(t *testing.T) {
	t.Parallel()

	b := &browse.State{
		Screen: browse.ScreenAlbums,
		Albums: []model.Album{
			{Name: "Discovery"},
			{Name: "RAM"},
			{Name: "Homework"},
		},
		SearchQuery: "dis",
	}

	assert.Equal(t, 1, b.ItemCount())
}

func TestAlbumCacheKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, album, artist, want string
	}{
		{"normal", "Discovery", "Daft Punk", "Discovery\x00Daft Punk"},
		{"empty album", "", "Artist", "\x00Artist"},
		{"empty artist", "Album", "", "Album\x00"},
		{"both empty", "", "", "\x00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, browse.AlbumCacheKey(tt.album, tt.artist))
		})
	}
}

func TestFilterPlaylistsByParent(t *testing.T) {
	t.Parallel()

	playlists := []model.Playlist{
		{Name: "Top", ParentID: ""},
		{Name: "Child A", ParentID: "F01"},
		{Name: "Child B", ParentID: "F01"},
		{Name: "Other", ParentID: "F02"},
	}

	t.Run("root items", func(t *testing.T) {
		t.Parallel()
		result := browse.FilterPlaylistsByParent(playlists, "")
		assert.Len(t, result, 1)
		assert.Equal(t, "Top", result[0].Name)
	})

	t.Run("children of F01", func(t *testing.T) {
		t.Parallel()
		result := browse.FilterPlaylistsByParent(playlists, "F01")
		assert.Len(t, result, 2)
	})

	t.Run("no match", func(t *testing.T) {
		t.Parallel()
		result := browse.FilterPlaylistsByParent(playlists, "nonexistent")
		assert.Empty(t, result)
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		result := browse.FilterPlaylistsByParent(nil, "")
		assert.Empty(t, result)
	})
}

func TestVisibleFolderItems(t *testing.T) {
	t.Parallel()

	b := &browse.State{
		Screen: browse.ScreenFolder,
		FolderItems: []model.Playlist{
			{Name: "Rock"},
			{Name: "Jazz"},
			{Name: "Pop"},
		},
	}

	t.Run("no filter", func(t *testing.T) {
		t.Parallel()
		s := *b
		result := browse.VisibleFolderItems(&s)
		assert.Len(t, result, 3)
	})

	t.Run("with search", func(t *testing.T) {
		t.Parallel()
		s := *b
		s.SearchQuery = "rock"
		result := browse.VisibleFolderItems(&s)
		assert.Len(t, result, 1)
		assert.Equal(t, "Rock", result[0].Name)
	})

	t.Run("with sort", func(t *testing.T) {
		t.Parallel()
		s := *b
		s.Sort = browse.SortAsc
		result := browse.VisibleFolderItems(&s)
		assert.Equal(t, "Jazz", result[0].Name)
	})
}

func TestVisibleAlbums(t *testing.T) {
	t.Parallel()

	b := &browse.State{
		Screen: browse.ScreenAlbums,
		Albums: []model.Album{
			{Name: "Discovery", AlbumArtist: "Daft Punk"},
			{Name: "RAM", AlbumArtist: "Daft Punk"},
			{Name: "Thriller", AlbumArtist: "Michael Jackson"},
		},
	}

	t.Run("search by name", func(t *testing.T) {
		t.Parallel()
		s := *b
		s.SearchQuery = "dis"
		result := browse.VisibleAlbums(&s)
		assert.Len(t, result, 1)
		assert.Equal(t, "Discovery", result[0].Name)
	})

	t.Run("search by artist", func(t *testing.T) {
		t.Parallel()
		s := *b
		s.SearchQuery = "michael"
		result := browse.VisibleAlbums(&s)
		assert.Len(t, result, 1)
		assert.Equal(t, "Thriller", result[0].Name)
	})
}

func TestVisibleAlbumTracks(t *testing.T) {
	t.Parallel()

	b := &browse.State{
		Screen: browse.ScreenAlbumTracks,
		AlbumTracks: []model.AlbumTrack{
			{Name: "One More Time", TrackNumber: 1},
			{Name: "Aerodynamic", TrackNumber: 2},
			{Name: "Veridis Quo", TrackNumber: 3},
		},
	}

	t.Run("search filter", func(t *testing.T) {
		t.Parallel()
		s := *b
		s.SearchQuery = "aero"
		result := browse.VisibleAlbumTracks(&s)
		assert.Len(t, result, 1)
		assert.Equal(t, "Aerodynamic", result[0].Name)
	})

	t.Run("sort ascending", func(t *testing.T) {
		t.Parallel()
		s := *b
		s.Sort = browse.SortAsc
		result := browse.VisibleAlbumTracks(&s)
		assert.Equal(t, "Aerodynamic", result[0].Name)
		assert.Equal(t, "Veridis Quo", result[2].Name)
	})
}

func TestSortAlbumTracks(t *testing.T) {
	t.Parallel()

	items := []model.AlbumTrack{
		{Name: "Charlie"},
		{Name: "Alpha"},
		{Name: "Bravo"},
	}

	t.Run("sortNone", func(t *testing.T) {
		t.Parallel()
		result := browse.SortAlbumTracks(items, browse.SortNone)
		assert.Equal(t, "Charlie", result[0].Name)
	})

	t.Run("sortAsc", func(t *testing.T) {
		t.Parallel()
		result := browse.SortAlbumTracks(items, browse.SortAsc)
		assert.Equal(t, "Alpha", result[0].Name)
		assert.Equal(t, "Charlie", result[2].Name)
	})

	t.Run("sortDesc", func(t *testing.T) {
		t.Parallel()
		result := browse.SortAlbumTracks(items, browse.SortDesc)
		assert.Equal(t, "Charlie", result[0].Name)
		assert.Equal(t, "Alpha", result[2].Name)
	})

	t.Run("does not mutate original", func(t *testing.T) {
		t.Parallel()
		_ = browse.SortAlbumTracks(items, browse.SortAsc)
		assert.Equal(t, "Charlie", items[0].Name)
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		result := browse.SortAlbumTracks(nil, browse.SortAsc)
		assert.Empty(t, result)
	})
}

func TestMoveCursor(t *testing.T) {
	t.Parallel()

	t.Run("move down", func(t *testing.T) {
		t.Parallel()
		b := &browse.State{
			Screen: browse.ScreenRoot,
		}
		browse.MoveCursor(b, 1)
		assert.Equal(t, 1, b.RootView.Cursor)
	})

	t.Run("move up", func(t *testing.T) {
		t.Parallel()
		b := &browse.State{
			Screen:   browse.ScreenRoot,
			RootView: browse.ListState{Cursor: 2},
		}
		browse.MoveCursor(b, -1)
		assert.Equal(t, 1, b.RootView.Cursor)
	})

	t.Run("clamp at top", func(t *testing.T) {
		t.Parallel()
		b := &browse.State{
			Screen:   browse.ScreenRoot,
			RootView: browse.ListState{Cursor: 0},
		}
		browse.MoveCursor(b, -1)
		assert.Equal(t, 0, b.RootView.Cursor)
	})

	t.Run("clamp at bottom", func(t *testing.T) {
		t.Parallel()
		b := &browse.State{
			Screen:   browse.ScreenRoot,
			RootView: browse.ListState{Cursor: browse.RootItemCount - 1},
		}
		browse.MoveCursor(b, 1)
		assert.Equal(t, browse.RootItemCount-1, b.RootView.Cursor)
	})

	t.Run("empty list no-op", func(t *testing.T) {
		t.Parallel()
		b := &browse.State{
			Screen: browse.ScreenAlbums,
		}
		browse.MoveCursor(b, 1)
		assert.Equal(t, 0, b.AlbumsView.Cursor)
	})
}

func TestSelectedFolderItem(t *testing.T) {
	t.Parallel()

	t.Run("valid selection", func(t *testing.T) {
		t.Parallel()
		b := &browse.State{
			Screen: browse.ScreenFolder,
			FolderItems: []model.Playlist{
				{Name: "A"},
				{Name: "B"},
			},
			FolderView: browse.ListState{Cursor: 1},
		}
		p, ok := browse.SelectedFolderItem(b)
		assert.True(t, ok)
		assert.Equal(t, "B", p.Name)
	})

	t.Run("out of bounds", func(t *testing.T) {
		t.Parallel()
		b := &browse.State{
			Screen:     browse.ScreenFolder,
			FolderView: browse.ListState{Cursor: 5},
		}
		_, ok := browse.SelectedFolderItem(b)
		assert.False(t, ok)
	})
}

func TestEnterFolder(t *testing.T) {
	t.Parallel()

	playlists := []model.Playlist{
		{Name: "Root", PersistentID: "F01", SpecialKind: "folder"},
		{Name: "Child", PersistentID: "P01", ParentID: "F01"},
	}

	b := &browse.State{
		Screen:    browse.ScreenPlaylists,
		Playlists: playlists,
	}

	browse.EnterFolder(b, playlists[0])

	assert.Equal(t, browse.ScreenFolder, b.Screen)
	assert.Equal(t, "F01", b.FolderID)
	assert.Equal(t, "Root", b.FolderName)
	assert.Len(t, b.FolderItems, 1)
	assert.Equal(t, "Child", b.FolderItems[0].Name)
	assert.Len(t, b.FolderStack, 1)
	assert.Equal(t, "", b.FolderStack[0])
}

func TestExitFolder(t *testing.T) {
	t.Parallel()

	t.Run("exit to playlists (empty stack)", func(t *testing.T) {
		t.Parallel()
		b := &browse.State{
			Screen:      browse.ScreenFolder,
			FolderStack: nil,
		}
		browse.ExitFolder(b)
		assert.Equal(t, browse.ScreenPlaylists, b.Screen)
	})

	t.Run("exit to parent folder", func(t *testing.T) {
		t.Parallel()
		b := &browse.State{
			Screen:      browse.ScreenFolder,
			FolderStack: []string{"", "F01"},
			FolderID:    "F02",
			Playlists: []model.Playlist{
				{Name: "Parent", PersistentID: "F01", SpecialKind: "folder"},
				{Name: "Child", PersistentID: "F02", ParentID: "F01"},
			},
		}
		browse.ExitFolder(b)
		assert.Equal(t, browse.ScreenFolder, b.Screen)
		assert.Equal(t, "F01", b.FolderID)
		assert.Equal(t, "Parent", b.FolderName)
	})

	t.Run("exit to root from nested", func(t *testing.T) {
		t.Parallel()
		b := &browse.State{
			Screen:      browse.ScreenFolder,
			FolderStack: []string{""},
			FolderID:    "F01",
		}
		browse.ExitFolder(b)
		assert.Equal(t, browse.ScreenPlaylists, b.Screen)
	})
}

func TestCurrentList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		screen browse.Screen
	}{
		{"root", browse.ScreenRoot},
		{"playlists", browse.ScreenPlaylists},
		{"folder", browse.ScreenFolder},
		{"albums", browse.ScreenAlbums},
		{"album tracks", browse.ScreenAlbumTracks},
		{"catalog results", browse.ScreenCatalogResults},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b := &browse.State{Screen: tt.screen}
			list := b.CurrentList()
			assert.NotNil(t, list)
			// Verify it's a pointer to the correct field by modifying it.
			list.Cursor = 42
			assert.Equal(t, 42, b.CurrentList().Cursor)
		})
	}
}

func TestItemCountAllScreens(t *testing.T) {
	t.Parallel()

	t.Run("root", func(t *testing.T) {
		t.Parallel()
		b := &browse.State{Screen: browse.ScreenRoot}
		assert.Equal(t, browse.RootItemCount, b.ItemCount())
	})

	t.Run("playlists", func(t *testing.T) {
		t.Parallel()
		b := &browse.State{
			Screen:    browse.ScreenPlaylists,
			Playlists: []model.Playlist{{Name: "A", ParentID: ""}, {Name: "B", ParentID: ""}},
		}
		assert.Equal(t, 2, b.ItemCount())
	})

	t.Run("folder", func(t *testing.T) {
		t.Parallel()
		b := &browse.State{
			Screen:      browse.ScreenFolder,
			FolderItems: []model.Playlist{{Name: "X"}},
		}
		assert.Equal(t, 1, b.ItemCount())
	})

	t.Run("albums", func(t *testing.T) {
		t.Parallel()
		b := &browse.State{
			Screen: browse.ScreenAlbums,
			Albums: []model.Album{{Name: "A"}, {Name: "B"}, {Name: "C"}},
		}
		assert.Equal(t, 3, b.ItemCount())
	})

	t.Run("album tracks", func(t *testing.T) {
		t.Parallel()
		b := &browse.State{
			Screen:      browse.ScreenAlbumTracks,
			AlbumTracks: []model.AlbumTrack{{Name: "T1"}, {Name: "T2"}},
		}
		assert.Equal(t, 2, b.ItemCount())
	})

	t.Run("catalog results", func(t *testing.T) {
		t.Parallel()
		b := &browse.State{
			Screen:         browse.ScreenCatalogResults,
			CatalogResults: []model.Song{{Name: "S1"}},
		}
		assert.Equal(t, 1, b.ItemCount())
	})
}

func TestIsLoading(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		state browse.State
		want  bool
	}{
		{"root never loading", browse.State{Screen: browse.ScreenRoot}, false},
		{"playlists loading", browse.State{Screen: browse.ScreenPlaylists, PlaylistsLoading: true}, true},
		{"playlists not loading", browse.State{Screen: browse.ScreenPlaylists}, false},
		{"folder loading", browse.State{Screen: browse.ScreenFolder, PlaylistsLoading: true}, true},
		{"albums loading", browse.State{Screen: browse.ScreenAlbums, AlbumsLoading: true}, true},
		{"tracks loading", browse.State{Screen: browse.ScreenAlbumTracks, TracksLoading: true}, true},
		{"catalog loading", browse.State{Screen: browse.ScreenCatalogResults, CatalogLoading: true}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.state.IsLoading())
		})
	}
}
