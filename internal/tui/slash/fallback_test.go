package slash

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

func TestFallbackForInput(t *testing.T) {
	got := FallbackForInput("/not-a-command")
	if !got.Handled || !strings.Contains(got.Status, "unknown command /not-a-command") {
		t.Fatalf("FallbackForInput unknown = %+v", got)
	}
	got = FallbackForInput("")
	if got.Handled || got.Status != "" {
		t.Fatalf("FallbackForInput empty = %+v, want empty", got)
	}
}

func TestKnownUnhandledStatusAndAmbiguous(t *testing.T) {
	status := KnownUnhandledStatus("cp", cli.CommandPolicy{Name: "copy", Surface: cli.CommandSurfaceGateway})
	if !strings.Contains(status, "/cp -> /copy") || !strings.Contains(status, "gateway support") {
		t.Fatalf("KnownUnhandledStatus = %q", status)
	}
	ambiguous := AmbiguousNameStatus([]string{"a", "b", "c", "d", "e", "f", "g"})
	if ambiguous != "ambiguous command: a, b, c, d, e, f, ..." {
		t.Fatalf("AmbiguousNameStatus = %q", ambiguous)
	}
}
