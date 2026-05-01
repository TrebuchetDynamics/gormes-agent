# Gormes — Go-native Hermes port. No Python, no virtualenv, no runtime drift.
# Builds from source via standard `go build`. The static binary contract means
# this formula is intentionally simpler than the upstream `hermes-agent.rb`.
#
# Source build mirrors the Makefile contract:
#   VERSION=<version> go build -trimpath -ldflags="-s -w -X main.Version=<version>" ./cmd/gormes
#
# To update version for a new release:
#   1. Bump the `version` field below
#   2. The sha256 will fail until corrected — Homebrew will print the expected hash
class Gormes < Formula
  desc "Self-improving AI agent runtime in Go — Hermes-in-Go, no Python backend"
  homepage "https://gormes.ai"
  license "MIT"
  version "0.2.0-scout"

  on_macos do
    on_intel do
      url "file://#{ENV["HOMEBREW_CACHE"]}/gormes-#{version}-darwin-amd64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
    on_arm do
      url "file://#{ENV["HOMEBREW_CACHE"]}/gormes-#{version}-darwin-arm64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  on_linux do
    on_x86_64 do
      url "file://#{ENV["HOMEBREW_CACHE"]}/gormes-#{version}-linux-amd64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
    on_arm64 do
      url "file://#{ENV["HOMEBREW_CACHE"]}/gormes-#{version}-linux-arm64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  def install
    bin.install "gormes"
  end

  test do
    assert_match "gormes", shell_output("#{bin}/gormes version")
    assert_match "gormes", shell_output("#{bin}/gormes doctor --offline")
  end
end
