# Release Notes: v1.2.0

**Summary:** Add `--version` / `-v` flag for printing the program version

## Overview

Patch release adding a `--version` (`-v`) command-line flag that prints the
build-injected version string and exits, making it easier to confirm which
build is installed when debugging.

## Features

### Version Flag

- **`--version` / `-v`**: Prints the version string set at build time via
  `-ldflags "-X main.version=..."` (e.g. `1.2.0-abc123`) and exits without
  entering the TUI
- Handled before terminal raw-mode setup, so it works cleanly in any
  environment
