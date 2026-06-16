package snapshot

import (
	"strings"
	"testing"
)

func TestBuildShellWrapperLoadsSnapshotAndPreservesCommandOutput(t *testing.T) {
	wrapper, evidence := BuildShellWrapper(Config{Mode: Enabled, SnapshotPath: "/tmp/gormes snapshot/it's-live.sh"}, "echo visible")

	if evidence.Code != EvidenceLoaded {
		t.Fatalf("evidence.Code = %q, want %q", evidence.Code, EvidenceLoaded)
	}
	lines := strings.Split(wrapper, "\n")
	if len(lines) != 2 {
		t.Fatalf("wrapper lines = %#v, want source line plus command", lines)
	}
	wantSource := "source '/tmp/gormes snapshot/it'\"'\"'s-live.sh' >/dev/null 2>&1 || true"
	if lines[0] != wantSource {
		t.Fatalf("source line = %q, want %q", lines[0], wantSource)
	}
	if lines[1] != "echo visible" {
		t.Fatalf("command line = %q, want command verbatim", lines[1])
	}
}

func TestBuildShellWrapperSkipsMissingSnapshot(t *testing.T) {
	wrapper, evidence := BuildShellWrapper(Config{Mode: Enabled}, "echo visible")

	if wrapper != "echo visible" {
		t.Fatalf("wrapper = %q, want command verbatim", wrapper)
	}
	if evidence.Code != EvidencePathMissing {
		t.Fatalf("evidence.Code = %q, want %q", evidence.Code, EvidencePathMissing)
	}
}
