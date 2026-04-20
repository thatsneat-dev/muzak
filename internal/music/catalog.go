package music

import (
	"context"
	_ "embed"
	"strconv"
)

// playCatalogScript is the embedded AppleScript that plays a catalog track by store ID.
//
//go:embed play_catalog.applescript
var playCatalogScript string

// PlayCatalogTrack plays an Apple Music catalog track by its store ID
// using MPMusicPlayerController. Does not steal focus.
func PlayCatalogTrack(ctx context.Context, trackID int) error {
	_, err := runScript(ctx, playCatalogScript, strconv.Itoa(trackID))
	return err
}
