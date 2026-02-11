{
  description = "Control plane for the Relay.md network";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs =
    { nixpkgs, ... }:
    let
      forAllSystems = nixpkgs.lib.genAttrs nixpkgs.lib.systems.flakeExposed;
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          relay-control-plane = pkgs.callPackage ./nix/package.nix { };
          default = pkgs.callPackage ./nix/package.nix { };
        }
      );

      nixosModules = {
        relay-control-plane = import ./nix/module.nix;
        default = import ./nix/module.nix;
      };
    };
}
