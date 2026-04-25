version := `cat VERSION`
git_short := `git rev-parse --short=6 HEAD`
build_version := if env("CI", "") != "" { version } else { version + "-" + git_short }
binary := "muzak"
build_dir := "bin"

# List available recipes.
default:
    @just --list

# Show the current version.
version:
    @echo "{{version}}"

# Format all code (Go + Nix).
fmt:
    gofumpt -w .
    goimports -w .
    alejandra -q .

# Check formatting without modifying files.
fmt-check:
    @gofumpt -d . | diff /dev/null - >/dev/null 2>&1 || (echo "gofumpt: files need formatting" && gofumpt -d . && exit 1)
    @goimports -d . | diff /dev/null - >/dev/null 2>&1 || (echo "goimports: files need formatting" && goimports -d . && exit 1)
    @alejandra -c . >/dev/null 2>&1 || (echo "alejandra: files need formatting" && exit 1)

# Lint Go and Nix code.
lint:
    golangci-lint run --timeout=5m ./...
    statix check .
    deadnix .

# Run tests with race detection.
test:
    go test -race ./...

# Run tests with JUnit output for CI.
test-ci:
    gotestsum --format standard-verbose --junitfile test-results.xml -- -race ./...

# Run security vulnerability check.
vuln:
    govulncheck ./...

# Build the binary with version injection.
build:
    go build -ldflags "-X main.version={{build_version}}" -o {{build_dir}}/{{binary}} ./cmd/muzak

# Run the binary (build + exec).
run: build
    ./{{build_dir}}/{{binary}}

# Remove build artifacts.
clean:
    rm -rf {{build_dir}}

# Verify VERSION has been bumped from main.
version-check:
    #!/usr/bin/env bash
    set -euo pipefail
    git fetch origin main --depth=1 2>/dev/null
    main_version=$(git show origin/main:VERSION)
    pr_version=$(cat VERSION)
    if [ "$main_version" = "$pr_version" ]; then
        echo "VERSION has not been changed. Please run 'nix run .#patch|minor|major' before merging."
        exit 1
    fi
    echo "Version bumped: $main_version → $pr_version ✓"

# Verify release notes exist for the current version.
release-notes-check:
    #!/usr/bin/env bash
    set -euo pipefail
    version=$(cat VERSION)
    notes="docs/release-notes-v${version}.md"
    if [ ! -f "$notes" ]; then
        echo "Missing release notes at $notes"
        exit 1
    fi
    echo "Release notes found: $notes ✓"

# Run all checks (fmt, lint, test, vuln).
check: fmt-check lint test vuln

# Bump version (usage: just bump patch|minor|major).
bump type:
    nix run .#{{type}}
