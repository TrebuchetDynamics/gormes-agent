{
  description = "Gormes - Go-native Hermes-in-Go agent runtime, no Python backend";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
      gormesVersion = "0.2.0";
      gormesBuildDate = "1970-01-01T00:00:00Z";
    in
    {
      packages = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          lib = pkgs.lib;
          gitCommit = self.rev or "unknown";
        in
        {
          default = self.packages.${system}.gormes;

          gormes = pkgs.buildGoModule {
            pname = "gormes";
            version = gormesVersion;
            src = ../..;

            vendorHash = lib.fakeHash;
            subPackages = [ "cmd/gormes" ];
            env.CGO_ENABLED = "0";

            ldflags = [
              "-s"
              "-w"
              "-X main.Version=${gormesVersion}"
              "-X main.GitCommit=${gitCommit}"
              "-X main.GitDirty=false"
              "-X main.BuildDate=${gormesBuildDate}"
            ];

            doCheck = true;
            checkPhase = ''
              runHook preCheck
              go test ./cmd/gormes -run TestVersionCommand_JSONIncludesBuildDateField -count=1
              runHook postCheck
            '';

            meta.description = "Go-native Hermes-in-Go agent runtime";
            meta.homepage = "https://gormes.ai";
            meta.license = lib.licenses.mit;
            meta.mainProgram = "gormes";
            meta.platforms = lib.platforms.linux ++ lib.platforms.darwin;
          };
        });

      formatter = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        pkgs.nixfmt-standard);

      devShells = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            packages = with pkgs; [
              go_1_25
              gopls
              go-tools
              gofumpt
              ripgrep
            ];

            shellHook = ''
              echo "Gormes dev shell"
              echo "Build: go build -trimpath ./cmd/gormes"
            '';
          };
        });

      nixosModules.default = { config, lib, pkgs, ... }:
        let
          cfg = config.services.gormes-agent;
        in
        {
          options.services.gormes-agent = {
            enable = lib.mkEnableOption "Gormes Agent gateway service";

            package = lib.mkOption {
              type = lib.types.package;
              default = self.packages.${pkgs.stdenv.hostPlatform.system}.default;
              description = "Gormes package to run.";
            };

            settings = lib.mkOption {
              type = lib.types.attrs;
              default = { };
              description = "Gormes settings reserved for generated config handoff.";
            };

            environment = lib.mkOption {
              type = lib.types.attrsOf lib.types.str;
              default = { };
              description = "Non-secret environment variables for the Gormes service.";
            };

            environmentFiles = lib.mkOption {
              type = lib.types.listOf lib.types.path;
              default = [ ];
              description = "Environment files containing secrets for the Gormes service.";
            };

            stateDir = lib.mkOption {
              type = lib.types.str;
              default = "/var/lib/gormes-agent";
              description = "Persistent GORMES_HOME directory for service state.";
            };

            extraArgs = lib.mkOption {
              type = lib.types.listOf lib.types.str;
              default = [ ];
              description = "Extra arguments appended after `gormes gateway`.";
            };
          };

          config = lib.mkIf cfg.enable {
            systemd.services.gormes-agent = {
              description = "Gormes Agent gateway";
              wantedBy = [ "multi-user.target" ];
              after = [ "network-online.target" ];
              wants = [ "network-online.target" ];

              environment = cfg.environment // {
                GORMES_HOME = cfg.stateDir;
              };

              serviceConfig = {
                Type = "simple";
                DynamicUser = true;
                StateDirectory = "gormes-agent";
                WorkingDirectory = cfg.stateDir;
                EnvironmentFile = cfg.environmentFiles;
                ExecStart = "${lib.getExe cfg.package} gateway ${lib.escapeShellArgs cfg.extraArgs}";
                Restart = "on-failure";
                RestartSec = "5s";
              };
            };
          };
        };
    };
}
