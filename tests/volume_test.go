//go:build darwin && cgo

package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thatsneat-dev/muzak/internal/volume"
)

func TestGet(t *testing.T) {
	vol, err := volume.Get()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, vol, float32(0.0))
	assert.LessOrEqual(t, vol, float32(1.0))
}

func TestUpDown(t *testing.T) {
	// Save and restore the original volume so the test is non-destructive.
	original, err := volume.Get()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = volume.Set(original)
	})

	// Set to a known midpoint so Up/Down both have room.
	require.NoError(t, volume.Set(0.5))
	baseline, err := volume.Get()
	require.NoError(t, err)
	require.InDelta(t, 0.5, baseline, 0.05, "failed to set baseline volume")

	require.NoError(t, volume.Up())
	after, err := volume.Get()
	require.NoError(t, err)
	assert.Greater(t, after, baseline, "Up should increase volume")

	require.NoError(t, volume.Down())
	afterDown, err := volume.Get()
	require.NoError(t, err)
	assert.Less(t, afterDown, after, "Down should decrease volume")
}
