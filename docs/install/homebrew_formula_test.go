package install_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestHomebrewFormulaContract proves the Gormes Homebrew formula fixture
// ports the upstream Hermes Homebrew/release packaging expectations into a
// Go-native artifact path: a single static binary with a stable version
// output, an explicit release-asset URL + checksum, a binary install layout
// that lands the static gormes executable in `bin`, and a doctor smoke
// command that proves the installed binary is reachable. The release-asset
// script lives as an embedded fixture comment block in the same formula
// file so the row's write_scope stays minimal.
//
// The test inspects file content only — it must never invoke `brew install`,
// `brew audit`, network downloads, or any live tap mutation.
func TestHomebrewFormulaContract(t *testing.T) {
	formula := readRepoFile(t, "packaging/homebrew/gormes-agent.rb")

	tests := []struct {
		name     string
		body     string
		wantAll  []string
		wantNone []string
	}{
		{
			// Acceptance #1: Formula fixtures prove class name, version, URL,
			// checksum, binary install path, and doctor smoke command.
			name: "formula_declares_class_version_url_checksum_install_test",
			body: formula,
			wantAll: []string{
				"class GormesAgent < Formula",
				"desc \"",
				"homepage \"https://gormes.ai\"",
				"url \"https://github.com/TrebuchetDynamics/gormes-agent/releases/download/v",
				"sha256 \"",
				"version \"",
				"license \"MIT\"",
				"def install",
				"bin.install \"gormes\"",
				"test do",
				"#{bin}/gormes version",
				"#{bin}/gormes doctor --offline",
				"assert_match \"gormes\"",
			},
			wantNone: []string{
				// Gormes ships a single static Go binary; the formula must
				// not pull in the Hermes Python virtualenv pipeline or any
				// Python/Node runtime dependencies.
				"include Language::Python::Virtualenv",
				"depends_on \"python",
				"virtualenv_create",
				"pip_install",
				"pypi_packages",
				"depends_on \"node",
				"depends_on \"uv\"",
				"playwright",
			},
		},
		{
			// Acceptance #1 reinforce: the formula's binary install layout
			// must place the static gormes executable in `bin`, not under a
			// libexec virtualenv shim, and the doctor smoke must run offline.
			name: "formula_binary_layout_is_static_go_binary",
			body: formula,
			wantAll: []string{
				"bin.install \"gormes\"",
				"shell_output(\"#{bin}/gormes version\")",
				"shell_output(\"#{bin}/gormes doctor --offline\")",
			},
			wantNone: []string{
				"libexec/\"bin\"",
				"write_env_script",
				"HERMES_BUNDLED_SKILLS",
				"HERMES_OPTIONAL_SKILLS",
				"HERMES_MANAGED",
			},
		},
		{
			// Acceptance #2: Release-script fixture (embedded in the formula
			// as a comment block) proves Gormes artifact names and checksums
			// feed the formula without Hermes Python packaging paths — no
			// sdist, no wheels, no PyPI flow.
			name: "release_assets_emit_static_binary_archive_and_sha256",
			body: formula,
			wantAll: []string{
				"#!/bin/sh",
				"set -eu",
				"GORMES_VERSION=",
				"gormes-${GORMES_VERSION}-darwin-arm64.tar.gz",
				"gormes-${GORMES_VERSION}-darwin-amd64.tar.gz",
				"gormes-${GORMES_VERSION}-linux-amd64.tar.gz",
				"gormes-${GORMES_VERSION}-linux-arm64.tar.gz",
				"shasum -a 256",
				"./cmd/gormes",
				"go build",
			},
			wantNone: []string{
				"hermes_agent-",
				"python -m build",
				"twine upload",
				"pip wheel",
				"setup.py",
				"pyproject.toml",
				"sdist",
				".whl",
			},
		},
		{
			// Acceptance #2 reinforce: the SHA256 line in the formula must
			// be produced by the same shasum tool the release-asset fixture
			// emits, so a future bump pipes one tool's output into the
			// other and never requires Hermes Python tooling.
			name: "release_assets_sha256_format_matches_formula",
			body: formula,
			wantAll: []string{
				"shasum -a 256",
			},
			wantNone: []string{
				"md5",
				"sha1sum",
			},
		},
		{
			// Acceptance #3: Nix/flake references remain separate row-backed
			// work — the Homebrew formula and embedded release-asset fixture
			// must not reference flake.nix, nix-build, or pkgs.* expressions.
			name: "homebrew_does_not_drag_in_nix_or_flake_inputs",
			body: formula,
			wantNone: []string{
				"flake.nix",
				"nix-build",
				"nixpkgs",
				"buildGoModule",
				"pkgs.callPackage",
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

	t.Run("formula_version_and_url_share_same_semver", func(t *testing.T) {
		urlVer := captureGroup(t, formula,
			`url "https://github\.com/TrebuchetDynamics/gormes-agent/releases/download/v([0-9][0-9A-Za-z.\-]+)/`)
		versionVer := captureGroup(t, formula, `(?m)^\s*version "([0-9][0-9A-Za-z.\-]+)"`)
		if urlVer == "" || versionVer == "" {
			t.Fatalf("could not capture url=%q version=%q from formula", urlVer, versionVer)
		}
		if urlVer != versionVer {
			t.Errorf("formula url version %q does not match declared version %q",
				urlVer, versionVer)
		}
	})

	t.Run("formula_sha256_is_64_char_hex", func(t *testing.T) {
		got := captureGroup(t, formula, `(?m)^\s*sha256 "([0-9a-f]+)"`)
		if got == "" {
			t.Fatalf("sha256 line missing or not a lowercase hex string")
		}
		if len(got) != 64 {
			t.Errorf("sha256 must be 64 hex chars, got %d (%q)", len(got), got)
		}
	})
}

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	// Tests under docs/install run with cwd at docs/install, so reach back
	// to the repo root for top-level fixtures like packaging/homebrew/.
	raw, err := os.ReadFile(filepath.Join("..", "..", rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(raw)
}

func captureGroup(t *testing.T, body, pattern string) string {
	t.Helper()
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(body)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}
