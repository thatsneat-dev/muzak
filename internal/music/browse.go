package music

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/thatsneat-dev/muzak/internal/model"
)

// listPlaylistsScript is the embedded AppleScript that lists all user playlists.
//
//go:embed list_playlists.applescript
var listPlaylistsScript string

// listAlbumsScript is the embedded AppleScript that lists all library albums.
//
//go:embed list_albums.applescript
var listAlbumsScript string

// listAlbumTracksScript is the embedded AppleScript that lists tracks for a specific album.
//
//go:embed list_album_tracks.applescript
var listAlbumTracksScript string

// playPlaylistScript is the embedded AppleScript that starts playback of a playlist.
//
//go:embed play_playlist.applescript
var playPlaylistScript string

// playAlbumScript is the embedded AppleScript that starts playback of an album.
//
//go:embed play_album.applescript
var playAlbumScript string

// runScript executes a multi-line AppleScript from stdin with optional args,
// returning the captured stdout bytes.
func runScript(ctx context.Context, script string, args ...string) ([]byte, error) {
	cmdArgs := []string{"-"}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.CommandContext(ctx, "osascript", cmdArgs...)
	cmd.Stdin = strings.NewReader(script)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("osascript: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return bytes.TrimSpace(stdout.Bytes()), nil
}

// Launch starts Music.app in the background without stealing focus.
func Launch(ctx context.Context) error {
	return runCommand(ctx, `tell application "Music" to launch`)
}

// ListPlaylists returns all user and smart playlists.
func ListPlaylists(ctx context.Context) ([]model.Playlist, error) {
	out, err := runScript(ctx, listPlaylistsScript)
	if err != nil {
		return nil, err
	}
	return ParsePlaylists(out)
}

// ListAlbums returns all unique albums in the user's library.
func ListAlbums(ctx context.Context) ([]model.Album, error) {
	out, err := runScript(ctx, listAlbumsScript)
	if err != nil {
		return nil, err
	}
	return ParseAlbums(out)
}

// ListAlbumTracks returns tracks for a specific album.
func ListAlbumTracks(ctx context.Context, albumName, albumArtist string) ([]model.AlbumTrack, error) {
	out, err := runScript(ctx, listAlbumTracksScript, albumName, albumArtist)
	if err != nil {
		return nil, err
	}
	return ParseAlbumTracks(out)
}

// PlayPlaylist starts playback of a playlist by its persistent ID.
func PlayPlaylist(ctx context.Context, persistentID string) error {
	_, err := runScript(ctx, playPlaylistScript, persistentID)
	return err
}

// PlayAlbum starts playback of an album using a temporary playlist.
func PlayAlbum(ctx context.Context, albumName, albumArtist string) error {
	_, err := runScript(ctx, playAlbumScript, albumName, albumArtist)
	return err
}

// ParsePlaylists unmarshals JSON output into a slice of Playlist.
func ParsePlaylists(data []byte) ([]model.Playlist, error) {
	var playlists []model.Playlist
	if err := json.Unmarshal(bytes.TrimSpace(data), &playlists); err != nil {
		return nil, fmt.Errorf("parsing playlists: %w", err)
	}
	return playlists, nil
}

// ParseAlbums unmarshals JSON output into a slice of Album.
func ParseAlbums(data []byte) ([]model.Album, error) {
	var albums []model.Album
	if err := json.Unmarshal(bytes.TrimSpace(data), &albums); err != nil {
		return nil, fmt.Errorf("parsing albums: %w", err)
	}
	return albums, nil
}

// ParseAlbumTracks unmarshals JSON output into a slice of AlbumTrack.
func ParseAlbumTracks(data []byte) ([]model.AlbumTrack, error) {
	var tracks []model.AlbumTrack
	if err := json.Unmarshal(bytes.TrimSpace(data), &tracks); err != nil {
		return nil, fmt.Errorf("parsing album tracks: %w", err)
	}
	return tracks, nil
}
