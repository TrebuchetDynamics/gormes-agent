package installtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstall_DryRunTermuxPublishesOnlyToPrefixBin(t *testing.T) {
	sb, err := os.MkdirTemp("/tmp", "gormes-install-termux-")
	if err != nil {
		t.Fatalf("create Termux install fixture root: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(sb) })
	fixture := newTermuxDryRunFixture(sb)
	out := runInstallDryRun(t, fixture.env(nil), "--verbose")

	wantPublished := filepath.Join(fixture.Prefix, "bin", "gormes")
	wantManaged := filepath.Join(fixture.InstallHome, "bin", "gormes")
	for _, want := range []string{
		"termux: yes",
		"release_arch: android-arm64",
		"published_binary: " + wantPublished,
		"managed_binary: " + wantManaged,
		"install_system_service: skipped (Termux runtime;",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("Termux dry-run output missing %q:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{
		filepath.Join(fixture.Home, ".local", "bin", "gormes"),
		"/home/xel",
		"workspace-mineru",
		"workspace-gormes",
		"release_arch: linux-arm64",
	} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("Termux dry-run output contains forbidden path/marker %q:\n%s", forbidden, out)
		}
	}
	for _, path := range []string{wantPublished, wantManaged} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("dry-run should not write %q; stat err=%v", path, err)
		}
	}
}
