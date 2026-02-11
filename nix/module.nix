{
  config,
  lib,
  pkgs,
  ...
}:

let
  cfg = config.services.relay-control-plane;
  stateDir = "/var/lib/relay-control-plane";
in
{
  options.services.relay-control-plane = {
    enable = lib.mkEnableOption "Relay.md control plane (PocketBase)";

    package = lib.mkPackageOption pkgs "relay-control-plane" { };

    host = lib.mkOption {
      type = lib.types.str;
      default = "0.0.0.0";
      description = "Address to bind the HTTP server to.";
    };

    port = lib.mkOption {
      type = lib.types.port;
      default = 8090;
      description = "Port for the PocketBase HTTP server.";
    };

    environment = lib.mkOption {
      type = lib.types.attrsOf lib.types.str;
      default = { };
      description = ''
        Environment variables to set for the relay-control-plane service.
        Note: Do not use this for secrets; use {option}`environmentFile` instead.
      '';
    };

    environmentFile = lib.mkOption {
      type = lib.types.nullOr lib.types.path;
      default = null;
      description = ''
        Path to a file containing environment variables such as
        `RELAY_HMAC_KEY`, `RELAY_HMAC_KEY_ID`, and `RELAY_ISSUER`.
        This file is not added to the Nix store.
      '';
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.services.relay-control-plane = {
      description = "Relay.md Control Plane";
      wantedBy = [ "multi-user.target" ];
      after = [ "network.target" ];

      inherit (cfg) environment;

      preStart = ''
        ln -sfn ${cfg.package}/share/relay-control-plane/templates ${stateDir}/templates
      '';

      serviceConfig = {
        ExecStart = lib.concatStringsSep " " [
          (lib.getExe cfg.package)
          "serve"
          "--http=${cfg.host}:${toString cfg.port}"
          "--dir=${stateDir}/pb_data"
        ];
        WorkingDirectory = stateDir;
        Restart = "on-failure";

        DynamicUser = true;
        StateDirectory = "relay-control-plane";

        EnvironmentFile = lib.mkIf (cfg.environmentFile != null) cfg.environmentFile;

        # Hardening
        CapabilityBoundingSet = [ "" ];
        LockPersonality = true;
        NoNewPrivileges = true;
        PrivateDevices = true;
        PrivateTmp = true;
        ProcSubset = "pid";
        ProtectClock = true;
        ProtectControlGroups = true;
        ProtectHome = true;
        ProtectHostname = true;
        ProtectKernelLogs = true;
        ProtectKernelModules = true;
        ProtectKernelTunables = true;
        ProtectProc = "invisible";
        ProtectSystem = "strict";
        RestrictAddressFamilies = [
          "AF_INET"
          "AF_INET6"
          "AF_UNIX"
        ];
        RestrictNamespaces = true;
        RestrictRealtime = true;
        RestrictSUIDSGID = true;
        SystemCallArchitectures = "native";
        SystemCallFilter = [
          "@system-service"
          "~@privileged"
        ];
        UMask = "0077";
        AmbientCapabilities = [ "" ];
        DevicePolicy = "closed";
        MemoryDenyWriteExecute = true;
        RemoveIPC = true;
      };
    };
  };
}
