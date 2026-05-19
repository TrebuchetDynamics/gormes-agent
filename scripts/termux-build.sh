#!/bin/bash
# Termux package build script for gormes-agent.
# Follows the termux-packages build.sh convention.
# Usage: run inside termux-packages/packages/gormes/ or with termux-create-package.

TERMUX_PKG_HOMEPAGE=https://github.com/TrebuchetDynamics/gormes-agent
TERMUX_PKG_DESCRIPTION="Go-native Hermes-compatible AI agent runtime"
TERMUX_PKG_LICENSE="MIT"
TERMUX_PKG_MAINTAINER="Trebuchet Dynamics @TrebuchetDynamics"
TERMUX_PKG_VERSION=0.2.17

# GitHub release asset URL for the android-arm64 prebuilt binary
TERMUX_PKG_SRCURL=https://github.com/TrebuchetDynamics/gormes-agent/releases/download/v${TERMUX_PKG_VERSION}/gormes-${TERMUX_PKG_VERSION}-android-arm64.tar.gz
TERMUX_PKG_SHA256=SKIP_CHECKSUM

# Runtime dependency: libc++ is required for Go binaries on Android
TERMUX_PKG_DEPENDS="libc++"

# We ship a prebuilt binary; skip source extraction
TERMUX_PKG_SKIP_SRC_EXTRACT=true

termux_step_make_install() {
	install -Dm700 "${TERMUX_PKG_SRCDIR}/gormes" "${TERMUX_PREFIX}/bin/gormes"
}

termux_step_install_license() {
	install -Dm600 "${TERMUX_PKG_SRCDIR}/LICENSE" "${TERMUX_PREFIX}/share/doc/gormes/LICENSE"
}
