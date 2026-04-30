# Homebrew formula fixture for Gormes — a Go-native Hermes port. The static
# binary contract makes this formula intentionally narrower than upstream
# `hermes-agent.rb`: no Python virtualenv, no pip resources, no Node, no
# Playwright. Only `bin.install "gormes"` and an offline doctor smoke.
#
# Release-asset script (fixture, embedded as a comment so the row's
# write_scope stays minimal — `packaging/homebrew/gormes-agent.rb` only):
#
#   #!/bin/sh
#   # release_assets.sh — emits per-platform Gormes release archives that
#   # feed the formula's url + sha256 fields. Static Go binary only — no
#   # Python source-archive packaging, no Python-wheel build, no PyPI flow.
#   set -eu
#   GORMES_VERSION="${GORMES_VERSION:?set e.g. 0.2.0-scout}"
#   for target in darwin-arm64 darwin-amd64 linux-amd64 linux-arm64; do
#     goos="${target%-*}"; goarch="${target##*-}"
#     out="dist/gormes-${GORMES_VERSION}-${target}"
#     CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
#       go build -trimpath -ldflags="-s -w" -o "$out" ./cmd/gormes
#     tar -C dist -czf "${out}.tar.gz" "$(basename "$out")"
#     shasum -a 256 "${out}.tar.gz"
#   done
#   # Expected artifact names:
#   #   gormes-${GORMES_VERSION}-darwin-arm64.tar.gz
#   #   gormes-${GORMES_VERSION}-darwin-amd64.tar.gz
#   #   gormes-${GORMES_VERSION}-linux-amd64.tar.gz
#   #   gormes-${GORMES_VERSION}-linux-arm64.tar.gz
class GormesAgent < Formula
  desc "Self-improving AI agent that creates skills from experience (Go-native Hermes port)"
  homepage "https://gormes.ai"
  url "https://github.com/TrebuchetDynamics/gormes-agent/releases/download/v0.2.0-scout/gormes-0.2.0-scout-darwin-arm64.tar.gz"
  version "0.2.0-scout"
  sha256 "0000000000000000000000000000000000000000000000000000000000000000"
  license "MIT"

  def install
    bin.install "gormes"
  end

  test do
    assert_match "gormes", shell_output("#{bin}/gormes version")
    assert_match "gormes", shell_output("#{bin}/gormes doctor --offline")
  end
end
