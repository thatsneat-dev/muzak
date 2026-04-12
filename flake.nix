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
      in {
        packages.muzak = pkgs.buildGoModule {
          pname = "muzak";
          inherit version;
          src = ./.;

          vendorHash = null;

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

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            just
            gofumpt
            gotools
            golangci-lint
            alejandra
            statix
            deadnix
          ];
        };

        apps.muzak = {
          type = "app";
          program = "${self'.packages.muzak}/bin/muzak";
          meta.description = "Apple Music now-playing terminal widget";
        };
      };
    };
}
