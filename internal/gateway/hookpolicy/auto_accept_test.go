package hookpolicy

import "testing"

func TestResolveAutoAcceptPrecedenceAndValues(t *testing.T) {
	env := "ON"
	tests := []struct {
		name       string
		input      AutoAcceptInputs
		wantAccept bool
		wantCode   AutoAcceptEvidence
	}{
		{name: "cli", input: AutoAcceptInputs{CLIAccept: true, EnvValue: ptr("off"), ConfigValue: false}, wantAccept: true, wantCode: AutoAcceptAcceptedByCLI},
		{name: "env", input: AutoAcceptInputs{EnvValue: &env, ConfigValue: false}, wantAccept: true, wantCode: AutoAcceptAcceptedByEnv},
		{name: "config true", input: AutoAcceptInputs{ConfigValue: " yes "}, wantAccept: true, wantCode: AutoAcceptAcceptedByConfig},
		{name: "config false", input: AutoAcceptInputs{ConfigValue: false}, wantAccept: false, wantCode: AutoAcceptRejectedDefault},
		{name: "invalid", input: AutoAcceptInputs{ConfigValue: "no"}, wantAccept: false, wantCode: AutoAcceptInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveAutoAccept(tt.input)
			if got.Accept != tt.wantAccept || got.Evidence != tt.wantCode {
				t.Fatalf("ResolveAutoAccept() = %+v, want accept=%v evidence=%q", got, tt.wantAccept, tt.wantCode)
			}
		})
	}
}

func ptr(s string) *string { return &s }
