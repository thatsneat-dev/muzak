package browse_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thatsneat-dev/muzak/internal/browse"
	"github.com/thatsneat-dev/muzak/internal/music"
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

	items := []music.Playlist{
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

	items := []music.Album{
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
		Playlists: []music.Playlist{
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
		Playlists: []music.Playlist{
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
		Playlists: []music.Playlist{
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
		Albums: []music.Album{
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
		Albums: []music.Album{
			{Name: "Discovery"},
			{Name: "RAM"},
			{Name: "Homework"},
		},
		SearchQuery: "dis",
	}

	assert.Equal(t, 1, b.ItemCount())
}
