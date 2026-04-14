# Release Notes: v1.0.0

**Summary:** Initial release — live Apple Music now-playing widget for the terminal

## Overview

First release of `muzak`, a live Apple Music now-playing widget for terminals
that support the Kitty graphics protocol. Displays album artwork, track info, a
playback progress bar, and volume — all rendered inline with vim-style
keybindings. Built with cgo for direct CoreAudio volume control and AppleScript
for Music.app integration.

## Features

### Now Playing Display

- **Album artwork**: Rendered inline via the Kitty graphics protocol (Kitty,
  Ghostty, WezTerm, etc.)
- **Live playback position**: Text-based progress bar, updated every second
- **Track metadata**: Song title, artist, and album displayed alongside artwork
- **Visual feedback**: Control icons flash yellow on keypress

### Playback Controls

- **Play/pause**: Toggle with `space`
- **Track navigation**: `h` to restart or go to previous track (< 3s), `l` for
  next track
- **Non-blocking architecture**: Input is always responsive; polling runs in the
  background

### Volume Control

- **System volume via CoreAudio**: Direct cgo bindings for instant volume
  changes with no osascript overhead
- **Vertical volume bar**: Rendered with Nerd Font icons, controlled with
  `j`/`k`

### Browse Modal

- **Library browsing**: Browse and play playlists or albums from your library
  with `b`
- **Folder support**: Navigate playlist folders and nested playlists
- **Auto-launch**: Launches Apple Music on startup; opens browse modal if
  nothing is playing

### Terminal Handling

- **Alt screen**: Uses alternate screen buffer for clean display
- **SIGWINCH support**: Dynamic resizing on terminal window changes
- **Braille spinner**: Loading indicator while fetching track data

## Installation

- **Nix flake**: `nix run github:thatsneat-dev/muzak`
- **Go install**: `go install github.com/thatsneat-dev/muzak/cmd/muzak@latest`
- **From source**: `just build`

## Keybindings

| Key     | Action                                    |
| ------- | ----------------------------------------- |
| `space` | Toggle play/pause                         |
| `h`     | Restart track (or previous if < 3s in)    |
| `l`     | Next track                                |
| `k`     | Volume up                                 |
| `j`     | Volume down                               |
| `b`     | Browse playlists and library              |
| `q`     | Quit                                      |

## Requirements

- **macOS** (Apple Music via AppleScript + CoreAudio)
- **Terminal with Kitty graphics protocol support**
- **Nerd Font** (for control and volume icons)
- **Go 1.26+** (or Nix)

## Bug Fixes

- Fixed timer channel drain before `Reset` to prevent stale fires
- Poll errors now logged to stderr instead of silently dropped
- CoreAudio `OSStatus` errors properly propagated in volume package
- Stderr captured in playback control helpers
- Guard against missing value fields in AppleScript responses

## Documentation

- Added `README.md` with full feature list, codebase structure, installation,
  and development instructions
- Added `TODO.md` with project roadmap
