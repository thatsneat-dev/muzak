package music

import (
	"context"
	_ "embed"
	"strconv"
)

//go:embed play_catalog.applescript
var playCatalogScript string

// PlayCatalogTrack plays an Apple Music catalog track by its store ID
// using MPMusicPlayerController. Does not steal focus.
func PlayCatalogTrack(ctx context.Context, trackID int) error {
	_, err := runScript(ctx, playCatalogScript, strconv.Itoa(trackID))
	return err
}
