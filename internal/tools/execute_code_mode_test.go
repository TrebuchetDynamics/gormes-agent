package tools

import "testing"

func TestExecuteCodeModeResolverConfigValue(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  ExecuteCodeMode
	}{
		{name: "project", value: "project", want: ExecuteCodeModeProject},
		{name: "strict", value: "strict", want: ExecuteCodeModeStrict},
		{name: "case insensitive", value: "STRICT", want: ExecuteCodeModeStrict},
		{name: "trim whitespace", value: "  project  ", want: ExecuteCodeModeProject},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveExecuteCodeMode(ExecuteCodeModeResolverInput{
				ConfigSet:   true,
				ConfigValue: tc.value,
				Default:     ExecuteCodeModeStrict,
			})
			if got.Mode != tc.want {
				t.Fatalf("Mode = %q, want %q", got.Mode, tc.want)
			}
			if len(got.Evidence) != 0 {
				t.Fatalf("Evidence = %#v, want none", got.Evidence)
			}
		})
	}
}

func TestExecuteCodeModeResolverDefaultFallback(t *testing.T) {
	tests := []struct {
		name        string
		input       ExecuteCodeModeResolverInput
		want        ExecuteCodeMode
		wantInvalid bool
	}{
		{
			name: "absent config uses default",
			input: ExecuteCodeModeResolverInput{
				ConfigSet: false,
				Default:   ExecuteCodeModeStrict,
			},
			want: ExecuteCodeModeStrict,
		},
		{
			name: "empty config uses default with evidence",
			input: ExecuteCodeModeResolverInput{
				ConfigSet:   true,
				ConfigValue: "",
				Default:     ExecuteCodeModeProject,
			},
			want:        ExecuteCodeModeProject,
			wantInvalid: true,
		},
		{
			name: "nil-like config uses default with evidence",
			input: ExecuteCodeModeResolverInput{
				ConfigSet:   true,
				ConfigValue: nil,
				Default:     ExecuteCodeModeStrict,
			},
			want:        ExecuteCodeModeStrict,
			wantInvalid: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveExecuteCodeMode(tc.input)
			if got.Mode != tc.want {
				t.Fatalf("Mode = %q, want %q", got.Mode, tc.want)
			}
			hasInvalid := hasExecuteCodeModeEvidence(got.Evidence, ExecuteCodeModeEvidenceInvalid)
			if hasInvalid != tc.wantInvalid {
				t.Fatalf("invalid evidence = %v, want %v; evidence=%#v", hasInvalid, tc.wantInvalid, got.Evidence)
			}
		})
	}
}

func TestExecuteCodeModeResolverInvalidValuesWarnAndDefault(t *testing.T) {
	got := ResolveExecuteCodeMode(ExecuteCodeModeResolverInput{
		ConfigSet:   true,
		ConfigValue: "sk-live-secret-value",
		Default:     ExecuteCodeModeStrict,
	})
	if got.Mode != ExecuteCodeModeStrict {
		t.Fatalf("Mode = %q, want %q", got.Mode, ExecuteCodeModeStrict)
	}
	if len(got.Evidence) != 1 {
		t.Fatalf("Evidence = %#v, want one invalid config evidence", got.Evidence)
	}
	ev := got.Evidence[0]
	if ev.Code != ExecuteCodeModeEvidenceInvalid {
		t.Fatalf("Evidence code = %q, want %q", ev.Code, ExecuteCodeModeEvidenceInvalid)
	}
	if ev.Source != "config" {
		t.Fatalf("Evidence source = %q, want config", ev.Source)
	}
	if ev.Message == "" {
		t.Fatalf("Evidence message is empty")
	}
	if containsString([]string{ev.Message}, "sk-live-secret-value") {
		t.Fatalf("Evidence leaks raw invalid value: %#v", ev)
	}

	got = ResolveExecuteCodeMode(ExecuteCodeModeResolverInput{
		ConfigSet:   true,
		ConfigValue: "banana",
		Default:     ExecuteCodeMode("not-a-mode"),
	})
	if got.Mode != ExecuteCodeModeProject {
		t.Fatalf("Mode = %q, want safe default %q", got.Mode, ExecuteCodeModeProject)
	}
	if !hasExecuteCodeModeEvidence(got.Evidence, ExecuteCodeModeEvidenceInvalid) {
		t.Fatalf("Evidence = %#v, want invalid evidence", got.Evidence)
	}
}

func TestExecuteCodeModeResolverFreezesTokenSet(t *testing.T) {
	got := ValidExecuteCodeModes()
	if len(got) != 2 {
		t.Fatalf("ValidExecuteCodeModes length = %d, want 2: %#v", len(got), got)
	}
	if !containsExecuteCodeMode(got, ExecuteCodeModeProject) || !containsExecuteCodeMode(got, ExecuteCodeModeStrict) {
		t.Fatalf("ValidExecuteCodeModes = %#v, want project and strict", got)
	}

	for _, rejected := range []ExecuteCodeMode{"", "off", "disabled", "shell", "project-mode"} {
		if IsValidExecuteCodeMode(rejected) {
			t.Fatalf("IsValidExecuteCodeMode(%q) = true, want false", rejected)
		}
	}
}

func hasExecuteCodeModeEvidence(evidence []ExecuteCodeModeEvidence, code string) bool {
	for _, ev := range evidence {
		if ev.Code == code {
			return true
		}
	}
	return false
}

func containsExecuteCodeMode(modes []ExecuteCodeMode, mode ExecuteCodeMode) bool {
	for _, got := range modes {
		if got == mode {
			return true
		}
	}
	return false
}
