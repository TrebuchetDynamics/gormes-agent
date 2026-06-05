package guidance

import (
	"strings"
	"testing"
)

type guidanceSwitchCase struct {
	name               string
	guidance           string
	build              func(bool) GuidanceSwitchResult
	injectedEvidence   string
	suppressedEvidence string
}

func assertGuidanceSwitch(t *testing.T, tc guidanceSwitchCase) {
	t.Helper()

	injected := tc.build(true)
	if !injected.Injected {
		t.Fatalf("%s: expected injected when enabled", tc.name)
	}
	if injected.Guidance != tc.guidance {
		t.Fatalf("%s: expected full guidance text", tc.name)
	}
	if injected.Evidence != tc.injectedEvidence {
		t.Fatalf("%s: expected evidence=%s, got %s", tc.name, tc.injectedEvidence, injected.Evidence)
	}

	suppressed := tc.build(false)
	if suppressed.Injected {
		t.Fatalf("%s: expected not injected when disabled", tc.name)
	}
	if suppressed.Guidance != "" {
		t.Fatalf("%s: expected empty guidance when suppressed", tc.name)
	}
	if !strings.Contains(suppressed.Evidence, tc.suppressedEvidence) {
		t.Fatalf("%s: expected suppression evidence containing %q, got %s", tc.name, tc.suppressedEvidence, suppressed.Evidence)
	}

	again := tc.build(true)
	if injected.Guidance != again.Guidance || injected.Injected != again.Injected || injected.Evidence != again.Evidence {
		t.Fatalf("%s: expected deterministic guidance switch result", tc.name)
	}
}
