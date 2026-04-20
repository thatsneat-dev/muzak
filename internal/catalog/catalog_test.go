package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thatsneat-dev/muzak/internal/catalog"
)

func TestArtworkURL500(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"100x100 to 500x500",
			"https://is1-ssl.mzstatic.com/image/thumb/Music/v4/abc/100x100bb.jpg",
			"https://is1-ssl.mzstatic.com/image/thumb/Music/v4/abc/500x500bb.jpg",
		},
		{
			"200x200 to 500x500",
			"https://example.com/art/200x200bb.png",
			"https://example.com/art/500x500bb.png",
		},
		{
			"no dimension pattern unchanged",
			"https://example.com/art/image.jpg",
			"https://example.com/art/image.jpg",
		},
		{
			"empty string",
			"",
			"",
		},
		{
			"already 500x500",
			"https://example.com/500x500bb.jpg",
			"https://example.com/500x500bb.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, catalog.ArtworkURL500(tt.input))
		})
	}
}
