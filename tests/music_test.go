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
