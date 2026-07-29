{
  description = "kubectl-ondemand: Analyze why Karpenter nodes are on-demand";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs =
    { self, nixpkgs }:
    let
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems =
        f: nixpkgs.lib.genAttrs supportedSystems (system: f system nixpkgs.legacyPackages.${system});
    in
    {
      # Overlay that adds `kubectl-ondemand` to a nixpkgs instance,
      # for consumers who prefer composing it into their own package set.
      overlays.default = final: _prev: {
        kubectl-ondemand = final.callPackage ./nix/package.nix { };
      };

      # `nix build`, `nix build .#kubectl-ondemand`, and the input for
      # the NixOS / home-manager modules below.
      packages = forAllSystems (
        _system: pkgs: rec {
          kubectl-ondemand = pkgs.callPackage ./nix/package.nix { };
          default = kubectl-ondemand;
        }
      );

      # `nix run` -> `kubectl-ondemand` (also usable as `kubectl ondemand`).
      apps = forAllSystems (
        system: _pkgs: rec {
          kubectl-ondemand = {
            type = "app";
            program = nixpkgs.lib.getExe self.packages.${system}.kubectl-ondemand;
            meta.description = "Analyze why Karpenter nodes are on-demand";
          };
          default = kubectl-ondemand;
        }
      );

      # Matches the flox dev environment: Go 1.26 plus the release tooling.
      devShells = forAllSystems (
        _system: pkgs: {
          default = pkgs.mkShellNoCC {
            packages = [
              pkgs.go_1_26
              pkgs.golangci-lint
              pkgs.goreleaser
              pkgs.kubectl
            ];
          };
        }
      );

      formatter = forAllSystems (_system: pkgs: pkgs.nixfmt);

      # Declarative installation of the krew/kubectl plugin on NixOS or via
      # home-manager. Enabling either puts `kubectl-ondemand` on PATH,
      # which is all kubectl needs to expose `kubectl ondemand`.
      nixosModules.default = import ./nix/nixos-module.nix self;
      homeManagerModules.default = import ./nix/home-manager-module.nix self;
    };
}
