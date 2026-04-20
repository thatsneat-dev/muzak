{
  description = "muzak - Apple Music now-playing terminal widget";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };
  };

  outputs = inputs @ {flake-parts, ...}:
    flake-parts.lib.mkFlake {inherit inputs;} {
      systems = [
        "aarch64-darwin"
        "x86_64-darwin"
      ];

      perSystem = {
        self',
        pkgs,
        ...
      }: let
        version = pkgs.lib.fileContents ./VERSION;
        go = pkgs.go_1_26.overrideAttrs (_: rec {
          version = "1.26.2";
          src = pkgs.fetchurl {
            url = "https://go.dev/dl/go${version}.src.tar.gz";
            hash = "sha256-LpHrtpR6lulDb7KzkmqIAu/mOm03Xf/sT4Kqnb1v1Ds=";
          };
        });
        buildGoModule = pkgs.buildGoModule.override {
          inherit go;
        };
      in {
        packages.default = self'.packages.muzak;
        packages.muzak = buildGoModule {
          pname = "muzak";
          inherit version;
          src = pkgs.lib.fileset.toSource {
            root = ./.;
            fileset = pkgs.lib.fileset.unions [
              ./cmd
              ./internal
              ./go.mod
              ./go.sum
              ./VERSION
            ];
          };

          vendorHash = "sha256-D5TnGKBhKrv+sP3kOEj92zfAa8jg76OhpQrpDxFXf0U=";

          subPackages = ["cmd/muzak"];

          ldflags = [
            "-X main.version=${version}"
          ];

          meta = {
            description = "Apple Music now-playing terminal widget";
            homepage = "https://github.com/thatsneat-dev/muzak";
            license = pkgs.lib.licenses.mit;
            mainProgram = "muzak";
            platforms = pkgs.lib.platforms.darwin;
          };
        };

        devShells = let
          devPackages =
            [
              go
            ]
            ++ (with pkgs; [
              just
              gofumpt
              gotools
              golangci-lint
              govulncheck
              gotestsum
              alejandra
              statix
              deadnix
            ]);
        in {
          default = pkgs.mkShell {packages = devPackages;};
          bash = pkgs.mkShell {packages = devPackages;};
          zsh = pkgs.mkShell {
            packages = devPackages;
            shellHook = ''
              exec zsh
            '';
          };
        };

        apps = let
          bumpApp = kind: {
            type = "app";
            program = toString (pkgs.writeShellScript "bump-${kind}" ''
              set -euo pipefail
              IFS='.' read -r major minor patch < VERSION
              case "${kind}" in
                patch) patch=$((patch + 1)) ;;
                minor) minor=$((minor + 1)); patch=0 ;;
                major) major=$((major + 1)); minor=0; patch=0 ;;
              esac
              echo "$major.$minor.$patch" > VERSION
              echo "v$major.$minor.$patch"
            '');
          };
        in {
          default = self'.apps.muzak;
          muzak = {
            type = "app";
            program = "${self'.packages.muzak}/bin/muzak";
            meta.description = "Apple Music now-playing terminal widget";
          };
          patch = bumpApp "patch";
          minor = bumpApp "minor";
          major = bumpApp "major";
        };
      };
    };
}
