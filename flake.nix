{
  description = "Smart session manager for the terminal";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        pname = "sesh";
        version = "dev";

        sesh = pkgs.buildGoModule rec {
          inherit pname version;

          src = pkgs.lib.cleanSource ./.;

          vendorHash = "sha256-9wJmseb2WDbQAbwlxfYzDFmeuZeQRkCcvGhAyBnBj1I=";

          ldflags = [
            "-s"
            "-w"
            "-X main.version=${version}"
          ];

          # Disable CGO for static binary
          env.CGO_ENABLED = "0";

          # Build flags for optimization
          tags = [ "netgo" ];

          # Skip tests for now due to missing mock dependencies
          doCheck = false;

          meta = with pkgs.lib; {
            description = "Smart session manager for the terminal";
            homepage = "https://github.com/joshmedeski/sesh";
            license = licenses.mit;
            mainProgram = "sesh";
          };
        };

      in
      {
        packages = {
          default = sesh;
        };

        apps = {
          default = flake-utils.lib.mkApp {
            drv = sesh;
            name = "sesh";
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            go-tools
            git
            gnumake
            go-mockery
          ];

          shellHook = ''
            echo "🚀 Sesh development environment"
            echo "Go version: $(go version)"
            echo "Available commands:"
            echo "  go build          - Build the binary"
            echo "  go test ./...     - Run tests"
            echo "  go run .          - Run sesh directly"
            echo "  make              - Build using Makefile"
          '';
        };

        # Formatter for the flake
        formatter = pkgs.nixpkgs-fmt;
      }
    );
}
