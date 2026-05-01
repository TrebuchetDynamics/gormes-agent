{
  description = "Gormes — Go-native Hermes-in-Go agent runtime, no Python backend";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      eachSystem = nixpkgs.lib.genAttrs [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
        "x86_64-darwin"
      ];
    in
    {
      packages = eachSystem (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          gormesVersion = "0.2.0-scout";
        in
        {
          default = self.packages.${system}.gormes;

          gormes = pkgs.buildGoModule {
            pname = "gormes";
            version = gormesVersion;

            src = ../..;

            vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";

            buildPhase = ''
              runHook preBuild
              CGO_ENABLED=0 GOOS=''${pkgs.hostPlatform.system} GOARCH=''${pkgs.hostPlatform.arch} \
                go build -trimpath -ldflags="-s -w -X main.Version=${gormesVersion}" -o gormes ./cmd/gormes
              runHook postBuild
            '';

            installPhase = ''
              runHook preInstall
              install -Dm755 gormes $out/bin/gormes
              runHook postInstall
            '';

            doCheck = false;

            meta = with pkgs.lib; {
              description = "Go-native Hermes-in-Go agent runtime";
              homepage = "https://gormes.ai";
              license = licenses.mit;
              maintainers = [ ];
              platforms = platforms.linux ++ platforms.darwin;
            };
          };
        }
      );

      formatter = eachSystem (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        pkgs.nixfmt-standard
      );

      devShells = eachSystem (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              go_1_23
              golangci-lint
              goimports
            ];

            shellHook = ''
              export VERSION=${self.rev or "dev"}
              echo "Gormes dev shell"
              echo "Build: go build -ldflags=\"-X main.Version=$VERSION\" ./cmd/gormes"
            '';
          };
        }
      );
    };
}
