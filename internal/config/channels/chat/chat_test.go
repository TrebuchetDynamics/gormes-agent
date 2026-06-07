package chat

import "testing"

type flexibleBoolToken string

func (t flexibleBoolToken) String() string { return string(t) }

func TestDiscordFlexibleBoolValuesShareStringParsing(t *testing.T) {
	cases := []struct {
		name         string
		requireValue any
		autoValue    any
		defaultValue bool
		wantRequire  bool
		wantAuto     bool
	}{
		{
			name:         "trimmed string tokens",
			requireValue: " off ",
			autoValue:    " YES ",
			defaultValue: true,
			wantRequire:  false,
			wantAuto:     true,
		},
		{
			name:         "fmt stringer tokens",
			requireValue: flexibleBoolToken(" no "),
			autoValue:    flexibleBoolToken(" 1 "),
			defaultValue: true,
			wantRequire:  false,
			wantAuto:     true,
		},
		{
			name:         "invalid tokens keep default",
			requireValue: "sometimes",
			autoValue:    flexibleBoolToken("later"),
			defaultValue: false,
			wantRequire:  false,
			wantAuto:     false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DiscordCfg{RequireMention: tc.requireValue, AutoThread: tc.autoValue}
			if got := cfg.RequireMentionValue(tc.defaultValue); got != tc.wantRequire {
				t.Fatalf("RequireMentionValue() = %t, want %t", got, tc.wantRequire)
			}
			if got := cfg.AutoThreadValue(tc.defaultValue); got != tc.wantAuto {
				t.Fatalf("AutoThreadValue() = %t, want %t", got, tc.wantAuto)
			}
		})
	}
}
