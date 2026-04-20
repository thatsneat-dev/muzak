# Release Notes: v1.1.0

**Summary:** Queue view, shuffle/repeat, catalog search, browse refinements, major refactor, and CI

## Overview

Second release of `muzak` — adds queue display, shuffle and repeat controls,
Apple Music catalog search and streaming, browse modal refinements (sorting,
fuzzy search, emoji support), a full codebase refactor into internal packages,
and GitHub Actions CI/CD pipelines.

## Features

### Queue View

- **Up Next display**: View upcoming tracks from the current playlist with `q`
- **Scrollable modal**: Vim-style `j`/`k` scrolling, displays up to 5 tracks
- **Loading spinner**: Braille dot animation while fetching queue data

### Shuffle & Repeat

- **Shuffle toggle**: `s` to toggle shuffle on/off with visual feedback
- **Repeat cycling**: `r` to cycle through off → all → one → off
- **Status icons**: Nerd Font icons highlighted in yellow when active

### Catalog Search & Streaming

- **Search the Apple Music catalog**: New "Search Catalog" option in the browse
  modal for discovering music beyond your library
- **iTunes Search API**: Public catalog search with no authentication required
- **Stream playback**: Play catalog tracks via MediaPlayer framework without
  stealing window focus
- **Artwork support**: Catalog artwork converted from JPEG to PNG for Kitty
  protocol compatibility

### Browse Modal Enhancements

- **Sorting**: `a` for ascending, `d` for descending (case-insensitive)
- **Fuzzy search**: `/` to activate subsequence matching (e.g., "bt" matches
  "Best Of Taylor"), `Enter` to confirm, `Esc` to clear
- **Emoji and wide character support**: Integrated `go-runewidth` for correct
  terminal cell width calculations, fixing modal alignment issues
- **Folder navigation**: Sorting and search apply correctly within playlist
  folders
- **Nerd Font icons**: Contextual icons for playlists, library, and folders in
  modal titles
- **Dynamic hints**: Keybinding hints rendered in brackets, wrapping to two
  lines when the modal is narrow
- **Close key**: Changed from `q` to `x` to avoid conflict with quit

### Terminal Handling Improvements

- **Alternate screen buffer**: Uses `\x1b[?1049h` for clean display without
  cluttering terminal history
- **SIGWINCH support**: Dynamic recalculation of layout, artwork, and overlays
  on terminal resize
- **Scrollback clearing**: Clears both visible screen and scrollback buffer on
  startup
- **Text truncation**: ANSI-aware truncation with ellipsis for long track names
  and metadata
- **Progress bar scaling**: Playback bar adapts to available terminal width

## Keybindings

| Key     | Action                                    |
| ------- | ----------------------------------------- |
| `space` | Toggle play/pause                         |
| `h`     | Restart track (or previous if < 3s in)    |
| `l`     | Next track                                |
| `k`     | Volume up                                 |
| `j`     | Volume down                               |
| `b`     | Browse playlists and library              |
| `q`     | Show queue / quit (context-sensitive)     |
| `s`     | Toggle shuffle                            |
| `r`     | Cycle repeat mode                         |
| `a`     | Sort ascending (in browse)                |
| `d`     | Sort descending (in browse)               |
| `/`     | Search (in browse)                        |
| `x`     | Close modal                               |

## Codebase

### Refactor

Reduced `cmd/muzak/main.go` from ~1,124 lines to 187 by extracting logic into
internal packages:

- **`internal/ui`**: Terminal rendering primitives, layout, text helpers, icons,
  Kitty graphics, and drawing functions
- **`internal/browse`**: Browse modal state machine, fuzzy matching, sorting,
  folder navigation, and rendering
- **`internal/catalog`**: iTunes Search API client and artwork processing
- **`cmd/muzak/app.go`**: Runtime state and event handling methods
- **`cmd/muzak/keys.go`**: Key dispatch split into now-playing, queue, and
  browse handlers

### Linting & Formatting

- Resolved all 19 `golangci-lint` issues (18 unchecked error returns, 1 unused
  variable)
- Added `goimports` formatting alongside `gofumpt` and `alejandra`

### Nix & Tooling

- **Go 1.26.2**: Pinned via `overrideAttrs` in `flake.nix` to address
  CVEs in `crypto/x509` and `crypto/tls` (GO-2026-4947, GO-2026-4946,
  GO-2026-4870, GO-2026-4866)
- **DevShells**: Added `bash` and `zsh` shell variants; added `govulncheck` and
  `gotestsum` to dev tooling
- **Flake apps**: Version bumping via `nix run .#patch|minor|major`
- **Source filtering**: Added `fileset` to `flake.nix` for reproducible builds

### CI/CD

- **CI workflow**: Runs on PRs to main — flake check, formatting, linting,
  vulnerability scan, Nix build, and tests with JUnit reporting
- **Release workflow**: Runs on PR merge — tags the version, creates a GitHub
  release with release notes from `docs/`
- **Version bump check**: CI verifies the `VERSION` file has changed from main
- **Release notes check**: CI verifies `docs/release-notes-v{version}.md`
  exists

## Bug Fixes

- Fixed now-playing detection for streaming/catalog tracks by falling back to
  Music.app's `current track` when `MPMusicPlayerController` returns
  `missing value`
- Fixed UI overflow from long hint strings causing terminal wrapping in catalog
  search view
- Fixed modal border misalignment caused by emoji and wide characters
- Fixed `SIGWINCH` redraw to correctly re-render overlays after resize
