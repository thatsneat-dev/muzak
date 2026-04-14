package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thatsneat-dev/muzak/internal/music"
)

func TestParseResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantNil  bool
		wantErr  bool
		expected *music.TrackInfo
	}{
		{
			name:  "playing track",
			input: `{"state":"playing","artist":"Daft Punk","album":"Discovery","name":"One More Time","duration":320.5,"position":45.2}`,
			expected: &music.TrackInfo{
				Artist:   "Daft Punk",
				Album:    "Discovery",
				Name:     "One More Time",
				Duration: 320.5,
				Position: 45.2,
				Playing:  true,
			},
		},
		{
			name:  "paused track",
			input: `{"state":"paused","artist":"Daft Punk","album":"Discovery","name":"One More Time","duration":320.5,"position":45.2}`,
			expected: &music.TrackInfo{
				Artist:   "Daft Punk",
				Album:    "Discovery",
				Name:     "One More Time",
				Duration: 320.5,
				Position: 45.2,
				Playing:  false,
			},
		},
		{
			name:    "stopped returns nil",
			input:   `{"state":"stopped","artist":"","album":"","name":"","duration":0,"position":0}`,
			wantNil: true,
		},
		{
			name:    "empty state returns nil",
			input:   `{"state":"","artist":"","album":"","name":"","duration":0,"position":0}`,
			wantNil: true,
		},
		{
			name:  "whitespace around JSON",
			input: "  \n" + `{"state":"playing","artist":"A","album":"B","name":"C","duration":100,"position":10}` + "  \n",
			expected: &music.TrackInfo{
				Artist:   "A",
				Album:    "B",
				Name:     "C",
				Duration: 100,
				Position: 10,
				Playing:  true,
			},
		},
		{
			name:    "invalid JSON",
			input:   `not json`,
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   ``,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			info, err := music.ParseResponse([]byte(tt.input))

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.wantNil {
				assert.Nil(t, info)
				return
			}

			require.NotNil(t, info)
			assert.Equal(t, tt.expected, info)
		})
	}
}

func TestParsePlaylists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantErr  bool
		expected []music.Playlist
	}{
		{
			name:  "valid array with multiple playlists",
			input: `[{"name":"Library","persistentID":"ABC123","specialKind":"Library","trackCount":100},{"name":"Favorites","persistentID":"DEF456","specialKind":"","trackCount":25}]`,
			expected: []music.Playlist{
				{Name: "Library", PersistentID: "ABC123", SpecialKind: "Library", TrackCount: 100},
				{Name: "Favorites", PersistentID: "DEF456", SpecialKind: "", TrackCount: 25},
			},
		},
		{
			name:  "playlists with folders and parentID",
			input: `[{"name":"Rock","persistentID":"F01","specialKind":"folder","trackCount":0,"parentID":""},{"name":"Classic Rock","persistentID":"P01","specialKind":"none","trackCount":50,"parentID":"F01"}]`,
			expected: []music.Playlist{
				{Name: "Rock", PersistentID: "F01", SpecialKind: "folder", TrackCount: 0, ParentID: ""},
				{Name: "Classic Rock", PersistentID: "P01", SpecialKind: "none", TrackCount: 50, ParentID: "F01"},
			},
		},
		{
			name:     "empty array",
			input:    `[]`,
			expected: []music.Playlist{},
		},
		{
			name:  "whitespace around JSON",
			input: "  \n" + `[{"name":"Library","persistentID":"ABC123","specialKind":"Library","trackCount":100}]` + "  \n",
			expected: []music.Playlist{
				{Name: "Library", PersistentID: "ABC123", SpecialKind: "Library", TrackCount: 100},
			},
		},
		{
			name:    "invalid JSON",
			input:   `not json`,
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   ``,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := music.ParsePlaylists([]byte(tt.input))

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseAlbums(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantErr  bool
		expected []music.Album
	}{
		{
			name:  "valid array with multiple albums",
			input: `[{"name":"Discovery","albumArtist":"Daft Punk","trackCount":14},{"name":"Random Access Memories","albumArtist":"Daft Punk","trackCount":13}]`,
			expected: []music.Album{
				{Name: "Discovery", AlbumArtist: "Daft Punk", TrackCount: 14},
				{Name: "Random Access Memories", AlbumArtist: "Daft Punk", TrackCount: 13},
			},
		},
		{
			name:     "empty array",
			input:    `[]`,
			expected: []music.Album{},
		},
		{
			name:  "whitespace around JSON",
			input: "  \n" + `[{"name":"Discovery","albumArtist":"Daft Punk","trackCount":14}]` + "  \n",
			expected: []music.Album{
				{Name: "Discovery", AlbumArtist: "Daft Punk", TrackCount: 14},
			},
		},
		{
			name:    "invalid JSON",
			input:   `not json`,
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   ``,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := music.ParseAlbums([]byte(tt.input))

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseAlbumTracks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantErr  bool
		expected []music.AlbumTrack
	}{
		{
			name:  "valid array with multiple tracks",
			input: `[{"name":"One More Time","persistentID":"AAA111","discNumber":1,"trackNumber":1,"duration":320.5},{"name":"Veridis Quo","persistentID":"BBB222","discNumber":2,"trackNumber":3,"duration":345.8}]`,
			expected: []music.AlbumTrack{
				{Name: "One More Time", PersistentID: "AAA111", DiscNumber: 1, TrackNumber: 1, Duration: 320.5},
				{Name: "Veridis Quo", PersistentID: "BBB222", DiscNumber: 2, TrackNumber: 3, Duration: 345.8},
			},
		},
		{
			name:     "empty array",
			input:    `[]`,
			expected: []music.AlbumTrack{},
		},
		{
			name:  "whitespace around JSON",
			input: "  \n" + `[{"name":"One More Time","persistentID":"AAA111","discNumber":1,"trackNumber":1,"duration":320.5}]` + "  \n",
			expected: []music.AlbumTrack{
				{Name: "One More Time", PersistentID: "AAA111", DiscNumber: 1, TrackNumber: 1, Duration: 320.5},
			},
		},
		{
			name:    "invalid JSON",
			input:   `not json`,
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   ``,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := music.ParseAlbumTracks([]byte(tt.input))

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
