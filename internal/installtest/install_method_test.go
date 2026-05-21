// Package installtest — see iso_bin_dir_test.go for the dry-run-as-public
// contract testing rationale shared with this file.
//
// These tests cover the install_method decision: install.sh now defaults to
// fetching pre-built release binaries from GitHub Releases instead of cloning
// + building from source. The decision is exposed via the dry-run plan as
// `install_method: binary-fetch|source-build (<reason>)` so the test contract
// is exactly what the operator sees.
package installtest

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestInstall_DryRunDefaultPlatform_PrefersBinaryFetch proves the new default:
// on a supported platform with no override flags, the dry-run plan reports
// install_method as binary-fetch. This is the user-facing fix for the
// "shouldn't we use release binaries?" gap from the 2026-05-07 fresh-fedora
// install report.
func TestInstall_DryRunDefaultPlatform_PrefersBinaryFetch(t *testing.T) {
	if !supportedHostForBinaryFetch() {
		t.Skipf("host platform %s/%s has no published release asset; binary-fetch test does not apply", runtime.GOOS, runtime.GOARCH)
	}
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"GORMES_INSTALL_HOME":    filepath.Join(sb, "home"),
		"GORMES_SKIP_SETUP":      "1",
		"GORMES_RESTART_GATEWAY": "never",
	})

	if !strings.Contains(out, "install_method: binary-fetch") {
		t.Fatalf("dry-run plan should report install_method as binary-fetch by default\non supported platforms; got:\n%s", out)
	}
	if !strings.Contains(out, "no Go toolchain or git clone needed") {
		t.Fatalf("dry-run plan should explain WHY binary-fetch is the right default\n(no Go toolchain, no git clone); got:\n%s", out)
	}
	// Plan must NOT advertise a source build when binary-fetch is the chosen path.
	if strings.Contains(out, "source: managed git checkout") {
		t.Fatalf("dry-run plan must not advertise managed git checkout when\ninstall_method is binary-fetch; got:\n%s", out)
	}
}

// TestInstall_DryRunFromSourceFlag_ForcesSourceBuild proves the opt-out:
// operators can still force source-build for hermetic verification, locked-down
// hosts that cannot reach github.com release assets, or pre-release branches.
func TestInstall_DryRunFromSourceFlag_ForcesSourceBuild(t *testing.T) {
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"GORMES_INSTALL_HOME":    filepath.Join(sb, "home"),
		"GORMES_SKIP_SETUP":      "1",
		"GORMES_RESTART_GATEWAY": "never",
	}, "--from-source")

	if !strings.Contains(out, "install_method: source-build") {
		t.Fatalf("dry-run plan should report install_method as source-build\nwhen --from-source is set; got:\n%s", out)
	}
	if !strings.Contains(out, "--from-source flag set") {
		t.Fatalf("dry-run plan should name the override that triggered source-build\nso operators can audit the decision; got:\n%s", out)
	}
	if !strings.Contains(out, "source: managed git checkout") {
		t.Fatalf("source-build path must still advertise the managed git checkout\nso operators know what will be cloned; got:\n%s", out)
	}
}

// TestInstall_DryRunFromSourceEnvVar_ForcesSourceBuild proves the env-var
// equivalent of --from-source so unattended installs (cron, CI, IaC) can opt
// out without touching argv.
func TestInstall_DryRunFromSourceEnvVar_ForcesSourceBuild(t *testing.T) {
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"GORMES_INSTALL_HOME":        filepath.Join(sb, "home"),
		"GORMES_SKIP_SETUP":          "1",
		"GORMES_RESTART_GATEWAY":     "never",
		"GORMES_INSTALL_FROM_SOURCE": "1",
	})

	if !strings.Contains(out, "install_method: source-build") {
		t.Fatalf("dry-run plan should report install_method as source-build\nwhen GORMES_INSTALL_FROM_SOURCE=1; got:\n%s", out)
	}
}

// TestInstall_DryRunNonDefaultBranch_FallsBackToSourceBuild proves the
// branch-safety fence: release assets are only published from main, so any
// non-main branch must fall back to source-build automatically.
func TestInstall_DryRunNonDefaultBranch_FallsBackToSourceBuild(t *testing.T) {
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"GORMES_INSTALL_HOME":    filepath.Join(sb, "home"),
		"GORMES_BRANCH":          "development",
		"GORMES_SKIP_SETUP":      "1",
		"GORMES_RESTART_GATEWAY": "never",
	})

	if !strings.Contains(out, "install_method: source-build") {
		t.Fatalf("dry-run plan should report install_method as source-build\nwhen --branch is non-default (release binaries only published from main); got:\n%s", out)
	}
	if !strings.Contains(out, "release binaries are only published from main") {
		t.Fatalf("dry-run plan should explain the branch-safety reason; got:\n%s", out)
	}
}

// TestInstall_DryRunVerboseDefault_SurfacesReleaseArchAndApi proves the
// verbose plan includes operator-auditable details: which release-asset arch
// the binary-fetch will request and which GitHub API endpoint will be hit.
func TestInstall_DryRunVerboseDefault_SurfacesReleaseArchAndApi(t *testing.T) {
	if !supportedHostForBinaryFetch() {
		t.Skipf("host platform %s/%s has no published release asset; verbose binary-fetch test does not apply", runtime.GOOS, runtime.GOARCH)
	}
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"GORMES_INSTALL_HOME":    filepath.Join(sb, "home"),
		"GORMES_SKIP_SETUP":      "1",
		"GORMES_RESTART_GATEWAY": "never",
		"GORMES_INSTALL_VERBOSE": "1",
	}, "--verbose")

	if !strings.Contains(out, "source_mode: github-releases") {
		t.Fatalf("verbose plan should declare source_mode as github-releases\nwhen install_method is binary-fetch; got:\n%s", out)
	}
	if !strings.Contains(out, "release_arch:") {
		t.Fatalf("verbose plan should declare release_arch (e.g. linux-amd64)\nso operators know which asset will be downloaded; got:\n%s", out)
	}
	if !strings.Contains(out, "release_api:") {
		t.Fatalf("verbose plan should declare release_api endpoint\nso operators can audit the network call; got:\n%s", out)
	}
}

// TestInstall_DryRunTermuxArm64_PrefersAndroidReleaseAsset proves Termux arm64
// is not treated as ordinary Linux arm64 for binary-fetch installs. The release
// workflow publishes a GOOS=android/GOARCH=arm64 artifact for this host shape.
func TestInstall_DryRunTermuxArm64_PrefersAndroidReleaseAsset(t *testing.T) {
	fixture := newTermuxDryRunFixture(t.TempDir())
	fixture.Prefix = "/data/data/com.termux/files/usr"
	out := runInstallDryRun(t, fixture.env(nil), "--verbose")

	if !strings.Contains(out, "install_method: binary-fetch") {
		t.Fatalf("Termux arm64 dry-run should prefer binary-fetch; got:\n%s", out)
	}
	if !strings.Contains(out, "release_arch: android-arm64") {
		t.Fatalf("Termux arm64 dry-run should select android-arm64 release asset; got:\n%s", out)
	}
	if strings.Contains(out, "release_arch: linux-arm64") {
		t.Fatalf("Termux arm64 dry-run must not select linux-arm64 release asset; got:\n%s", out)
	}
}

// TestInstall_DryRunLinuxArm64_StillUsesLinuxReleaseAsset is the regression
// fence for ordinary Linux arm64 hosts: the Termux android-arm64 special case
// must not steal non-Termux Linux installs.
func TestInstall_DryRunLinuxArm64_StillUsesLinuxReleaseAsset(t *testing.T) {
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"GORMES_INSTALL_HOME":         filepath.Join(sb, "home"),
		"GORMES_SKIP_SETUP":           "1",
		"GORMES_RESTART_GATEWAY":      "never",
		"GORMES_INSTALL_TEST_UNAME_M": "aarch64",
		"UNAME":                       "Linux",
	}, "--verbose")

	if !strings.Contains(out, "install_method: binary-fetch") {
		t.Fatalf("Linux arm64 dry-run should prefer binary-fetch; got:\n%s", out)
	}
	if !strings.Contains(out, "release_arch: linux-arm64") {
		t.Fatalf("Linux arm64 dry-run should select linux-arm64 release asset; got:\n%s", out)
	}
	if strings.Contains(out, "release_arch: android-arm64") {
		t.Fatalf("Linux arm64 dry-run must not select android-arm64 release asset; got:\n%s", out)
	}
}

// supportedHostForBinaryFetch returns true when the host runtime.GOOS/GOARCH
// matches one of the published release-asset slugs. The test runner is the
// host the install would actually run on, so this gates platform-dependent
// assertions.
func supportedHostForBinaryFetch() bool {
	switch runtime.GOOS {
	case "linux", "darwin", "android":
		switch runtime.GOARCH {
		case "amd64", "arm64":
			return true
		}
	}
	return false
}
