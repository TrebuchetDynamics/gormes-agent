package gateway

import (
	"context"
	"testing"
)

func TestHookAutoAcceptParser_BoolValues(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		wantAccept bool
		wantCode   hookAutoAcceptEvidence
	}{
		{
			name:       "true accepts",
			value:      true,
			wantAccept: true,
			wantCode:   hookAutoAcceptAcceptedByConfig,
		},
		{
			name:       "false rejects",
			value:      false,
			wantAccept: false,
			wantCode:   hookAutoAcceptRejectedDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveHookAutoAccept(hookAutoAcceptInputs{
				ConfigValue: tt.value,
			})

			assertHookAutoAcceptDecision(t, got, tt.wantAccept, tt.wantCode)
		})
	}
}

func TestHookAutoAcceptParser_StringTruthTable(t *testing.T) {
	tests := []struct {
		value      string
		wantAccept bool
		wantCode   hookAutoAcceptEvidence
	}{
		{value: "1", wantAccept: true, wantCode: hookAutoAcceptAcceptedByConfig},
		{value: "true", wantAccept: true, wantCode: hookAutoAcceptAcceptedByConfig},
		{value: "yes", wantAccept: true, wantCode: hookAutoAcceptAcceptedByConfig},
		{value: "on", wantAccept: true, wantCode: hookAutoAcceptAcceptedByConfig},
		{value: " TRUE ", wantAccept: true, wantCode: hookAutoAcceptAcceptedByConfig},
		{value: "\tyEs\n", wantAccept: true, wantCode: hookAutoAcceptAcceptedByConfig},
		{value: "false", wantAccept: false, wantCode: hookAutoAcceptInvalid},
		{value: "no", wantAccept: false, wantCode: hookAutoAcceptInvalid},
		{value: "0", wantAccept: false, wantCode: hookAutoAcceptInvalid},
		{value: "off", wantAccept: false, wantCode: hookAutoAcceptInvalid},
		{value: "", wantAccept: false, wantCode: hookAutoAcceptInvalid},
		{value: "always", wantAccept: false, wantCode: hookAutoAcceptInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got := resolveHookAutoAccept(hookAutoAcceptInputs{
				ConfigValue: tt.value,
			})

			assertHookAutoAcceptDecision(t, got, tt.wantAccept, tt.wantCode)
		})
	}
}

func TestHookAutoAcceptParser_NonBoolScalarsReject(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		wantCode hookAutoAcceptEvidence
	}{
		{name: "nil", value: nil, wantCode: hookAutoAcceptRejectedDefault},
		{name: "integer one", value: 1, wantCode: hookAutoAcceptInvalid},
		{name: "slice", value: []string{"true"}, wantCode: hookAutoAcceptInvalid},
		{name: "map", value: map[string]bool{"accept": true}, wantCode: hookAutoAcceptInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveHookAutoAccept(hookAutoAcceptInputs{
				ConfigValue: tt.value,
			})

			assertHookAutoAcceptDecision(t, got, false, tt.wantCode)
		})
	}
}

func TestHookAutoAcceptParser_CLIOverride(t *testing.T) {
	got := resolveHookAutoAccept(hookAutoAcceptInputs{
		CLIAccept:   true,
		ConfigValue: "false",
	})

	assertHookAutoAcceptDecision(t, got, true, hookAutoAcceptAcceptedByCLI)
}

func TestHookAutoAcceptParser_EnvEvidence(t *testing.T) {
	envValue := "ON"

	got := resolveHookAutoAccept(hookAutoAcceptInputs{
		EnvValue:    &envValue,
		ConfigValue: true,
	})

	assertHookAutoAcceptDecision(t, got, true, hookAutoAcceptAcceptedByEnv)
}

func TestGatewayApprovalChoiceParser(t *testing.T) {
	tests := []struct {
		value string
		want  ApprovalChoice
		ok    bool
	}{
		{value: "once", want: ApprovalChoiceOnce, ok: true},
		{value: " SESSION ", want: ApprovalChoiceSession, ok: true},
		{value: "always", want: ApprovalChoiceAlways, ok: true},
		{value: "deny", want: ApprovalChoiceDeny, ok: true},
		{value: "approved", ok: false},
		{value: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got, ok := ParseApprovalChoice(tt.value)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("choice = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGatewayApprovalResolverFunc(t *testing.T) {
	var got ApprovalResolution
	resolver := ApprovalResolverFunc(func(_ context.Context, res ApprovalResolution) error {
		got = res
		return nil
	})

	err := resolver.ResolveGatewayApproval(context.Background(), ApprovalResolution{
		SessionKey: "slack:C123:sess-1",
		Choice:     ApprovalChoiceOnce,
		Platform:   "slack",
		ChatID:     "C123",
		MessageID:  "1711111111.000100",
		ActorID:    "U42",
	})
	if err != nil {
		t.Fatalf("ResolveGatewayApproval: %v", err)
	}
	if got.SessionKey != "slack:C123:sess-1" || got.Choice != ApprovalChoiceOnce || got.Platform != "slack" {
		t.Fatalf("resolution = %+v, want Slack once resolution", got)
	}
}

func assertHookAutoAcceptDecision(t *testing.T, got hookAutoAcceptDecision, wantAccept bool, wantCode hookAutoAcceptEvidence) {
	t.Helper()

	if got.Accept != wantAccept {
		t.Fatalf("Accept = %v, want %v; decision=%+v", got.Accept, wantAccept, got)
	}
	if got.Evidence != wantCode {
		t.Fatalf("Evidence = %q, want %q; decision=%+v", got.Evidence, wantCode, got)
	}
}
