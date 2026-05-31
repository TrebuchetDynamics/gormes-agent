package autoaccept

import "testing"

func TestAutoAcceptParserBoolValues(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		wantAccept bool
		wantCode   AutoAcceptEvidence
	}{
		{name: "true accepts", value: true, wantAccept: true, wantCode: AutoAcceptAcceptedByConfig},
		{name: "false rejects", value: false, wantAccept: false, wantCode: AutoAcceptRejectedDefault},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveAutoAccept(AutoAcceptInputs{ConfigValue: tt.value})
			assertAutoAcceptDecision(t, got, tt.wantAccept, tt.wantCode)
		})
	}
}

func TestAutoAcceptParserStringTruthTable(t *testing.T) {
	tests := []struct {
		value      string
		wantAccept bool
		wantCode   AutoAcceptEvidence
	}{
		{value: "1", wantAccept: true, wantCode: AutoAcceptAcceptedByConfig},
		{value: "true", wantAccept: true, wantCode: AutoAcceptAcceptedByConfig},
		{value: "yes", wantAccept: true, wantCode: AutoAcceptAcceptedByConfig},
		{value: "on", wantAccept: true, wantCode: AutoAcceptAcceptedByConfig},
		{value: " TRUE ", wantAccept: true, wantCode: AutoAcceptAcceptedByConfig},
		{value: "\tyEs\n", wantAccept: true, wantCode: AutoAcceptAcceptedByConfig},
		{value: "false", wantAccept: false, wantCode: AutoAcceptInvalid},
		{value: "no", wantAccept: false, wantCode: AutoAcceptInvalid},
		{value: "0", wantAccept: false, wantCode: AutoAcceptInvalid},
		{value: "off", wantAccept: false, wantCode: AutoAcceptInvalid},
		{value: "", wantAccept: false, wantCode: AutoAcceptInvalid},
		{value: "always", wantAccept: false, wantCode: AutoAcceptInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got := ResolveAutoAccept(AutoAcceptInputs{ConfigValue: tt.value})
			assertAutoAcceptDecision(t, got, tt.wantAccept, tt.wantCode)
		})
	}
}

func TestAutoAcceptParserNonBoolScalarsReject(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		wantCode AutoAcceptEvidence
	}{
		{name: "nil", value: nil, wantCode: AutoAcceptRejectedDefault},
		{name: "integer one", value: 1, wantCode: AutoAcceptInvalid},
		{name: "slice", value: []string{"true"}, wantCode: AutoAcceptInvalid},
		{name: "map", value: map[string]bool{"accept": true}, wantCode: AutoAcceptInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveAutoAccept(AutoAcceptInputs{ConfigValue: tt.value})
			assertAutoAcceptDecision(t, got, false, tt.wantCode)
		})
	}
}

func TestAutoAcceptParserCLIOverride(t *testing.T) {
	got := ResolveAutoAccept(AutoAcceptInputs{CLIAccept: true, ConfigValue: "false"})
	assertAutoAcceptDecision(t, got, true, AutoAcceptAcceptedByCLI)
}

func TestAutoAcceptParserEnvEvidence(t *testing.T) {
	envValue := "ON"
	got := ResolveAutoAccept(AutoAcceptInputs{EnvValue: &envValue, ConfigValue: true})
	assertAutoAcceptDecision(t, got, true, AutoAcceptAcceptedByEnv)
}

func assertAutoAcceptDecision(t *testing.T, got AutoAcceptDecision, wantAccept bool, wantCode AutoAcceptEvidence) {
	t.Helper()
	if got.Accept != wantAccept {
		t.Fatalf("Accept = %v, want %v; decision=%+v", got.Accept, wantAccept, got)
	}
	if got.Evidence != wantCode {
		t.Fatalf("Evidence = %q, want %q; decision=%+v", got.Evidence, wantCode, got)
	}
}
