package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
			assert.Equal(t, tt.want, fuzzyMatch(tt.target, tt.query))
		})
	}
}

func TestVisibleLen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"plain ascii", "hello", 5},
		{"empty", "", 0},
		{"ansi escape", "\x1b[1mbold\x1b[0m", 4},
		{"emoji takes 2 cells", "🎵", 2},
		{"mixed emoji and text", "hi 🎵 yo", 8},
		{"nerd font icon", iconFolder, 1},
		{"ansi with emoji", "\x1b[33m🎵\x1b[0m", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, visibleLen(tt.input))
		})
	}
}

func TestTruncateVisibleWideChars(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		maxWidth int
		contains string
	}{
		{"ascii within limit", "hello", 10, "hello"},
		{"ascii truncated", "hello world", 5, "hell"},
		{"emoji at boundary", "ab🎵cd", 4, "ab"},
		{"zero width", "hello", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := truncateVisible(tt.input, tt.maxWidth)
			assert.Contains(t, result, tt.contains)
			assert.LessOrEqual(t, visibleLen(result), tt.maxWidth)
		})
	}
}

func TestTruncateVisibleEllipsisFitsInMaxWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		maxWidth int
		expected int
	}{
		{"exact fit no truncation", "hello", 5, 5},
		{"truncated with ellipsis", "hello world", 8, 8},
		{"long string narrow width", "abcdefghij", 4, 4},
		{"width 1 truncation", "abcdef", 1, 1},
		{"ansi styled truncation", "\x1b[1mhello world\x1b[0m", 6, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := truncateVisible(tt.input, tt.maxWidth)
			assert.Equal(t, tt.expected, visibleLen(result))
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
		result := sortPlaylists(items, sortNone)
		assert.Equal(t, "Chill", result[0].Name)
	})

	t.Run("sortAsc", func(t *testing.T) {
		t.Parallel()
		result := sortPlaylists(items, sortAsc)
		assert.Equal(t, "Ambient", result[0].Name)
		assert.Equal(t, "Beats", result[1].Name)
		assert.Equal(t, "Chill", result[2].Name)
	})

	t.Run("sortDesc", func(t *testing.T) {
		t.Parallel()
		result := sortPlaylists(items, sortDesc)
		assert.Equal(t, "Chill", result[0].Name)
		assert.Equal(t, "Beats", result[1].Name)
		assert.Equal(t, "Ambient", result[2].Name)
	})

	t.Run("does not mutate original", func(t *testing.T) {
		t.Parallel()
		_ = sortPlaylists(items, sortAsc)
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
		result := sortAlbums(items, sortAsc)
		assert.Equal(t, "Alpha", result[0].Name)
		assert.Equal(t, "A", result[1].AlbumArtist)
		assert.Equal(t, "B", result[2].AlbumArtist)
	})

	t.Run("sortDesc", func(t *testing.T) {
		t.Parallel()
		result := sortAlbums(items, sortDesc)
		assert.Equal(t, "Zen", result[0].Name)
		assert.Equal(t, "B", result[0].AlbumArtist)
	})
}

func TestVisiblePlaylistsWithSearch(t *testing.T) {
	t.Parallel()

	b := &browseState{
		Screen: browsePlaylists,
		Playlists: []music.Playlist{
			{Name: "Rock Hits", ParentID: ""},
			{Name: "Jazz Vibes", ParentID: ""},
			{Name: "Rock Classics", ParentID: ""},
		},
		SearchQuery: "rock",
	}

	result := visiblePlaylists(b)
	assert.Len(t, result, 2)
	assert.Equal(t, "Rock Hits", result[0].Name)
	assert.Equal(t, "Rock Classics", result[1].Name)
}

func TestVisiblePlaylistsWithSortAndSearch(t *testing.T) {
	t.Parallel()

	b := &browseState{
		Screen: browsePlaylists,
		Playlists: []music.Playlist{
			{Name: "Rock Hits", ParentID: ""},
			{Name: "Jazz Vibes", ParentID: ""},
			{Name: "Rock Classics", ParentID: ""},
		},
		SearchQuery: "rock",
		Sort:        sortAsc,
	}

	result := visiblePlaylists(b)
	assert.Len(t, result, 2)
	assert.Equal(t, "Rock Classics", result[0].Name)
	assert.Equal(t, "Rock Hits", result[1].Name)
}

func TestClearSearch(t *testing.T) {
	t.Parallel()

	b := &browseState{
		SearchQuery:  "test",
		SearchActive: true,
	}
	clearSearch(b)
	assert.Empty(t, b.SearchQuery)
	assert.False(t, b.SearchActive)
}

func TestBrowseSelectedPlaylist(t *testing.T) {
	t.Parallel()

	b := &browseState{
		Screen: browsePlaylists,
		Playlists: []music.Playlist{
			{Name: "Alpha", ParentID: ""},
			{Name: "Beta", ParentID: ""},
		},
		PlaylistsView: listState{Cursor: 1},
	}

	p, ok := browseSelectedPlaylist(b)
	assert.True(t, ok)
	assert.Equal(t, "Beta", p.Name)
}

func TestBrowseSelectedAlbum(t *testing.T) {
	t.Parallel()

	b := &browseState{
		Screen: browseAlbums,
		Albums: []music.Album{
			{Name: "Discovery", AlbumArtist: "Daft Punk"},
			{Name: "RAM", AlbumArtist: "Daft Punk"},
		},
		AlbumsView: listState{Cursor: 0},
	}

	a, ok := browseSelectedAlbum(b)
	assert.True(t, ok)
	assert.Equal(t, "Discovery", a.Name)
}

func TestDrawHintsSingleLine(t *testing.T) {
	t.Parallel()

	var lines []string
	draw := func(row, col int, s string) {
		lines = append(lines, s)
	}
	items := []string{"[j|k] move", "[enter] select", "[x] close"}
	drawHints(items, 60, 0, 0, draw)

	assert.Len(t, lines, 2) // content line + blank clear line
	assert.Contains(t, lines[0], "[j|k] move")
	assert.Contains(t, lines[0], "[enter] select")
	assert.Contains(t, lines[0], "[x] close")
}

func TestDrawHintsTwoLines(t *testing.T) {
	t.Parallel()

	var lines []string
	draw := func(row, col int, s string) {
		lines = append(lines, s)
	}
	items := []string{"[j|k] move", "[/] search", "[a|d] sort", "[enter] select", "[h] back", "[x] close"}
	drawHints(items, 30, 0, 0, draw)

	assert.Len(t, lines, 2)
	// First line should NOT contain all items.
	assert.NotContains(t, lines[0], "[x] close")
	// Second line should contain the overflow items.
	assert.Contains(t, lines[1], "[x] close")
}

func TestItemCountWithSearch(t *testing.T) {
	t.Parallel()

	b := &browseState{
		Screen: browseAlbums,
		Albums: []music.Album{
			{Name: "Discovery"},
			{Name: "RAM"},
			{Name: "Homework"},
		},
		SearchQuery: "dis",
	}

	assert.Equal(t, 1, b.itemCount())
}
