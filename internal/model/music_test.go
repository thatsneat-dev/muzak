package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thatsneat-dev/muzak/internal/model"
)

func TestPlaylistIsFolder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		specialKind string
		want        bool
	}{
		{"folder", "folder", true},
		{"none", "none", false},
		{"empty", "", false},
		{"Library", "Library", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := model.Playlist{SpecialKind: tt.specialKind}
			assert.Equal(t, tt.want, p.IsFolder())
		})
	}
}
