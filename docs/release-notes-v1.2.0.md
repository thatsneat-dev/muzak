# Release Notes: v1.2.0

**Summary:** Add `--version` / `-v` flag, simplify Go toolchain pinning

## Overview

Minor release adding a `--version` (`-v`) command-line flag that prints the
build-injected version string and exits, making it easier to confirm which
build is installed when debugging. Also simplifies the Nix flake by dropping
the manual Go overlay now that nixpkgs ships a current Go.

## Features

### Version Flag

- **`--version` / `-v`**: Prints the version string set at build time via
  `-ldflags "-X main.version=..."` (e.g. `1.2.0-abc123`) and exits without
  entering the TUI
- Handled before terminal raw-mode setup, so it works cleanly in any
  environment

## Tooling

### Nix Flake

- **Removed Go 1.26.2 `overrideAttrs` overlay**: nixpkgs unstable now ships
  Go 1.26.2 directly, so the manual `fetchurl` override is no longer needed
- **Tracks latest stable Go**: switched from `pkgs.go_1_26` to `pkgs.go`
  (the stable alias), so `nix flake update` will pick up future Go releases
  automatically
- **Updated `flake.lock`**: bumped `nixpkgs` to the 2026-05-05 snapshot

