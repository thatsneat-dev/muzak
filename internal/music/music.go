// Package music provides access to Apple Music's now-playing state via AppleScript.
package music

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

//go:embed now_playing.applescript
var nowPlayingScript string

// TrackInfo holds metadata about the currently playing track.
type TrackInfo struct {
	Artist   string  `json:"artist"`
	Album    string  `json:"album"`
	Name     string  `json:"name"`
	Duration float64 `json:"duration"`
	Position float64 `json:"position"`
	Playing  bool
}

type rawResponse struct {
	State    string  `json:"state"`
	Artist   string  `json:"artist"`
	Album    string  `json:"album"`
	Name     string  `json:"name"`
	Duration float64 `json:"duration"`
	Position float64 `json:"position"`
}

// NowPlaying runs the embedded AppleScript to get current track info.
// artworkPath is where the script will write artwork PNG data if available.
// Returns nil if no track is currently playing.
func NowPlaying(ctx context.Context, artworkPath string) (*TrackInfo, error) {
	cmd := exec.CommandContext(ctx, "osascript", "-", artworkPath)
	cmd.Stdin = strings.NewReader(nowPlayingScript)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("osascript: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	var raw rawResponse
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &raw); err != nil {
		return nil, fmt.Errorf("parsing track info: %w", err)
	}

	if raw.State == "stopped" || raw.State == "" {
		return nil, nil
	}

	return &TrackInfo{
		Artist:   raw.Artist,
		Album:    raw.Album,
		Name:     raw.Name,
		Duration: raw.Duration,
		Position: raw.Position,
		Playing:  raw.State == "playing",
	}, nil
}

// PlayPause toggles playback in Music.app.
func PlayPause(ctx context.Context) error {
	return exec.CommandContext(ctx, "osascript", "-e", `tell application "Music" to playpause`).Run()
}

// NextTrack skips to the next track in Music.app.
func NextTrack(ctx context.Context) error {
	return exec.CommandContext(ctx, "osascript", "-e", `tell application "Music" to next track`).Run()
}

// PreviousTrack goes to the previous track in Music.app.
func PreviousTrack(ctx context.Context) error {
	return exec.CommandContext(ctx, "osascript", "-e", `tell application "Music" to previous track`).Run()
}

// RestartTrack seeks to the beginning of the current track in Music.app.
func RestartTrack(ctx context.Context) error {
	return exec.CommandContext(ctx, "osascript", "-e", `tell application "Music" to set player position to 0`).Run()
}
