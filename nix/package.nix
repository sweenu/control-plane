{
  lib,
  buildGoModule,
}:

buildGoModule {
  pname = "relay-control-plane";
  version = "unstable-2026-02-11";

  src = lib.fileset.toSource {
    root = ./..;
    fileset = lib.fileset.intersection (lib.fileset.fromSource (lib.sources.cleanSource ./..)) (
      lib.fileset.unions [
        ./../go.mod
        ./../go.sum
        ./../main.go
        ./../tools.go
        ./../migrations
        ./../routes
        ./../cwt
        ./../templates
      ]
    );
  };

  vendorHash = "sha256-xWwubdF64vVkTC7/EYEIopsxXR7sqUlWA7I2KUtc4vY=";

  env.CGO_ENABLED = "0";
  ldflags = [
    "-s"
    "-w"
  ];

  postInstall = ''
    mkdir -p $out/share/relay-control-plane
    cp -r templates $out/share/relay-control-plane/
  '';

  meta = {
    description = "Control plane for the Relay.md network";
    homepage = "https://github.com/No-Instructions/relay-control-plane";
    mainProgram = "relay-control-plane";
  };
}
