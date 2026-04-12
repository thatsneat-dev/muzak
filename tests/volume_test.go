//go:build darwin && cgo

package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thatsneat-dev/muzak/internal/volume"
)

func TestGet(t *testing.T) {
	vol := volume.Get()
	assert.GreaterOrEqual(t, vol, float32(0.0))
	assert.LessOrEqual(t, vol, float32(1.0))
}

func TestUpDown(t *testing.T) {
	// Save and restore the original volume so the test is non-destructive.
	original := volume.Get()
	t.Cleanup(func() {
		volume.Set(original)
	})

	// Set to a known midpoint so Up/Down both have room.
	volume.Set(0.5)
	baseline := volume.Get()
	require.InDelta(t, 0.5, baseline, 0.05, "failed to set baseline volume")

	volume.Up()
	after := volume.Get()
	assert.Greater(t, after, baseline, "Up should increase volume")

	volume.Down()
	afterDown := volume.Get()
	assert.Less(t, afterDown, after, "Down should decrease volume")
}
