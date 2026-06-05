package install_test

import (
	"strings"
	"testing"
)

// TestNixFlakeContract proves the Gormes Nix fixture is a Go-native packaging
// contract. It inspects text only; it must never invoke nix, hit a binary
// cache, download dependencies, or require provider credentials.
func TestNixFlakeContract(t *testing.T) {
	flake := readRepoFile(t, "packaging/nix/flake.nix")

	tests := []struct {
		name     string
		body     string
		wantAll  []string
		wantNone []string
	}{
		{
			name: "flake_exposes_go_binary_package_for_supported_systems",
			body: flake,
			wantAll: []string{
				"description = \"Gormes",
				"systems = [",
				"\"x86_64-linux\"",
				"\"aarch64-linux\"",
				"\"x86_64-darwin\"",
				"\"aarch64-darwin\"",
				"packages = forAllSystems (system:",
				"default = self.packages.${system}.gormes;",
				"gormes = pkgs.buildGoModule",
				"pname = \"gormes\";",
				"src = ../..;",
				"subPackages = [ \"cmd/gormes\" ];",
				"vendorHash = lib.fakeHash;",
				"env.CGO_ENABLED = \"0\";",
				"ldflags = [",
				"\"-X main.Version=${gormesVersion}\"",
				"\"-X main.GitCommit=${gitCommit}\"",
				"\"-X main.GitDirty=false\"",
				"\"-X main.BuildDate=${gormesBuildDate}\"",
				"meta.mainProgram = \"gormes\";",
			},
			wantNone: []string{
				"GOOS=''${pkgs.hostPlatform.system}",
				"GOARCH=''${pkgs.hostPlatform.arch}",
				"CGO_ENABLED = 0;",
				"vendorSha256",
			},
		},
		{
			name: "flake_keeps_dev_shell_formatter_and_python_stack_out",
			body: flake,
			wantAll: []string{
				"formatter = forAllSystems",
				"pkgs.nixfmt-standard",
				"devShells = forAllSystems (system:",
				"default = pkgs.mkShell",
				"go_1_25",
				"gopls",
				"go-tools",
			},
			wantNone: []string{
				"flake-parts",
				"pyproject-nix",
				"uv2nix",
				"pyproject-build-systems",
				"npm-lockfile-fix",
				"python3",
				"nodejs",
				"playwright",
				"uv pip",
			},
		},
		{
			name: "flake_exposes_minimal_nixos_service_module",
			body: flake,
			wantAll: []string{
				"nixosModules.default =",
				"cfg = config.services.gormes-agent;",
				"options.services.gormes-agent =",
				"enable = lib.mkEnableOption \"Gormes Agent gateway service\";",
				"package = lib.mkOption",
				"settings = lib.mkOption",
				"environment = lib.mkOption",
				"environmentFiles = lib.mkOption",
				"stateDir = lib.mkOption",
				"extraArgs = lib.mkOption",
				"config = lib.mkIf cfg.enable",
				"systemd.services.gormes-agent =",
				"wantedBy = [ \"multi-user.target\" ];",
				"GORMES_HOME = cfg.stateDir;",
				"ExecStart = \"${lib.getExe cfg.package} gateway ${lib.escapeShellArgs cfg.extraArgs}\";",
				"StateDirectory = \"gormes-agent\";",
				"EnvironmentFile = cfg.environmentFiles;",
			},
			wantNone: []string{
				"services.hermes-agent",
				"HERMES_HOME",
				"postgres",
				"redis",
				"docker-compose",
				"prometheus",
				"grafana",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, want := range tc.wantAll {
				if !strings.Contains(tc.body, want) {
					t.Errorf("missing required fragment %q", want)
				}
			}
			for _, banned := range tc.wantNone {
				if strings.Contains(tc.body, banned) {
					t.Errorf("forbidden fragment present: %q", banned)
				}
			}
		})
	}
}
