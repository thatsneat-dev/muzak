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
	"time"

	"github.com/thatsneat-dev/muzak/internal/model"
)

// nowPlayingScript is the embedded AppleScript that queries Music.app for current track info.
//
//go:embed now_playing.applescript
var nowPlayingScript string

// queueScript is the embedded AppleScript that fetches upcoming queue tracks.
//
//go:embed queue.applescript
var queueScript string

// queueResponse wraps the JSON array of upcoming tracks from the queue AppleScript.
type queueResponse struct {
	Tracks []model.QueueTrack `json:"tracks"`
}

// rawResponse is the raw JSON structure returned by the now-playing AppleScript.
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
func NowPlaying(ctx context.Context, artworkPath string) (*model.TrackInfo, error) {
	cmd := exec.CommandContext(ctx, "osascript", "-", artworkPath)
	cmd.Stdin = strings.NewReader(nowPlayingScript)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("osascript: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return ParseResponse(stdout.Bytes())
}

// ParseResponse unmarshals the JSON output from the AppleScript
// and returns a TrackInfo, or nil when playback is stopped.
func ParseResponse(data []byte) (*model.TrackInfo, error) {
	var raw rawResponse
	if err := json.Unmarshal(bytes.TrimSpace(data), &raw); err != nil {
		return nil, fmt.Errorf("parsing track info: %w", err)
	}

	if raw.State == "stopped" || raw.State == "" {
		return nil, nil
	}

	return &model.TrackInfo{
		Artist:   raw.Artist,
		Album:    raw.Album,
		Name:     raw.Name,
		Duration: raw.Duration,
		Position: raw.Position,
		Playing:  raw.State == "playing",
	}, nil
}

// runCommand executes a single-line AppleScript, capturing stderr for
// actionable error messages on failure.
func runCommand(ctx context.Context, script string) error {
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("osascript: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return nil
}

// PlayPause toggles playback in Music.app.
func PlayPause(ctx context.Context) error {
	return runCommand(ctx, `tell application "Music" to playpause`)
}

// NextTrack skips to the next track in Music.app.
func NextTrack(ctx context.Context) error {
	return runCommand(ctx, `tell application "Music" to next track`)
}

// PreviousTrack goes to the previous track in Music.app.
func PreviousTrack(ctx context.Context) error {
	return runCommand(ctx, `tell application "Music" to previous track`)
}

// RestartTrack seeks to the beginning of the current track in Music.app.
func RestartTrack(ctx context.Context) error {
	return runCommand(ctx, `tell application "Music" to set player position to 0`)
}

// Queue returns the upcoming tracks in the current playlist (up to 10).
func Queue(ctx context.Context) ([]model.QueueTrack, error) {
	cmd := exec.CommandContext(ctx, "osascript", "-")
	cmd.Stdin = strings.NewReader(queueScript)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("osascript: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	var resp queueResponse
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &resp); err != nil {
		return nil, fmt.Errorf("parsing queue info: %w", err)
	}

	return resp.Tracks, nil
}

// ShuffleEnabled returns whether shuffle mode is currently on.
func ShuffleEnabled(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "osascript", "-e",
		`tell application "Music" to return (shuffle enabled) as text`)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("osascript: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return strings.TrimSpace(stdout.String()) == "true", nil
}

// ToggleShuffle toggles shuffle mode in Music.app and returns the new state.
func ToggleShuffle(ctx context.Context) (bool, error) {
	if err := runCommand(ctx, `tell application "Music" to set shuffle enabled to not shuffle enabled`); err != nil {
		return false, err
	}
	time.Sleep(200 * time.Millisecond)
	return ShuffleEnabled(ctx)
}

// RepeatMode returns the current repeat mode: "off", "one", or "all".
func RepeatMode(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "osascript", "-e",
		`tell application "Music" to return song repeat as text`)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("osascript: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return strings.TrimSpace(stdout.String()), nil
}

// CycleRepeat advances the repeat mode (off → all → one → off) and returns the new mode.
func CycleRepeat(ctx context.Context) (string, error) {
	mode, err := RepeatMode(ctx)
	if err != nil {
		return "", err
	}
	var next string
	switch mode {
	case "off":
		next = "all"
	case "all":
		next = "one"
	default:
		next = "off"
	}
	if err := runCommand(ctx, `tell application "Music" to set song repeat to `+next); err != nil {
		return "", err
	}
	time.Sleep(200 * time.Millisecond)
	return RepeatMode(ctx)
}
