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
    @alejandra -c . >/dev/null 2>&1 || (echo "alejandra: files need formatting" && exit 1)

# Lint Go and Nix code.
lint:
    golangci-lint run --timeout=5m ./...
    statix check .
    deadnix .

# Run tests with race detection.
test:
    go test -race ./...

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

# Run all checks (fmt, lint, test, vuln).
check: fmt-check lint test vuln

# Bump version (usage: just bump patch|minor|major).
bump type:
    #!/usr/bin/env bash
    set -euo pipefail
    IFS='.' read -r major minor patch < VERSION
    case "{{type}}" in
        patch) patch=$((patch + 1)) ;;
        minor) minor=$((minor + 1)); patch=0 ;;
        major) major=$((major + 1)); minor=0; patch=0 ;;
        *) echo "usage: just bump patch|minor|major" && exit 1 ;;
    esac
    echo "$major.$minor.$patch" > VERSION
    echo "v$major.$minor.$patch"
