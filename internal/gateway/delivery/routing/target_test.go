package routing

import "testing"

func TestParseTarget_Valid(t *testing.T) {
	origin := &OriginSource{
		Platform: "telegram",
		ChatID:   "42",
		ThreadID: "99",
	}

	tests := []struct {
		name string
		raw  string
		want Target
	}{
		{
			name: "origin",
			raw:  "origin",
			want: Target{Platform: "telegram", ChatID: "42", ThreadID: "99", IsOrigin: true},
		},
		{
			name: "local",
			raw:  "local",
			want: Target{Platform: "local"},
		},
		{
			name: "platform home",
			raw:  "discord",
			want: Target{Platform: "discord"},
		},
		{
			name: "explicit chat",
			raw:  "telegram:-100123",
			want: Target{Platform: "telegram", ChatID: "-100123", IsExplicit: true},
		},
		{
			name: "explicit thread",
			raw:  "telegram:-100123:77",
			want: Target{Platform: "telegram", ChatID: "-100123", ThreadID: "77", IsExplicit: true},
		},
		{
			name: "matrix room id with server delimiter",
			raw:  "matrix:!room:matrix.org",
			want: Target{Platform: "matrix", ChatID: "!room:matrix.org", IsExplicit: true},
		},
		{
			name: "simplex group id with delimiter",
			raw:  "simplex:group:ops",
			want: Target{Platform: "simplex", ChatID: "group:ops", IsExplicit: true},
		},
		{
			name: "simplex group id with multiple delimiters",
			raw:  "simplex:group:ops:subteam:thread",
			want: Target{Platform: "simplex", ChatID: "group:ops:subteam:thread", IsExplicit: true},
		},
		{
			name: "matrix room id with homeserver port",
			raw:  "matrix:!room:matrix.org:8448",
			want: Target{Platform: "matrix", ChatID: "!room:matrix.org:8448", IsExplicit: true},
		},
		{
			name: "matrix room and thread ids with server delimiters",
			raw:  "matrix:!room:matrix.org:$thread:matrix.org",
			want: Target{Platform: "matrix", ChatID: "!room:matrix.org", ThreadID: "$thread:matrix.org", IsExplicit: true},
		},
		{
			name: "generic colon-bearing chat id uses length encoding",
			raw:  "bluebubbles:9:chat:guid",
			want: Target{Platform: "bluebubbles", ChatID: "chat:guid", IsExplicit: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTarget(tt.raw, origin)
			if err != nil {
				t.Fatalf("ParseTarget(%q) error = %v", tt.raw, err)
			}
			if got != tt.want {
				t.Fatalf("ParseTarget(%q) = %+v, want %+v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestParseTargetRejectsLengthEncodedTargetSplittingUTF8(t *testing.T) {
	if got, err := ParseTarget("bluebubbles:1:é:x", nil); err == nil {
		t.Fatalf("ParseTarget accepted length-encoded target with split UTF-8: %+v", got)
	}
}

func TestTargetStringRoundTripsColonBearingChatID(t *testing.T) {
	want := Target{Platform: "bluebubbles", ChatID: "chat:guid", IsExplicit: true}
	got, err := ParseTarget(want.String(), nil)
	if err != nil {
		t.Fatalf("ParseTarget(%q): %v", want.String(), err)
	}
	if got != want {
		t.Fatalf("ParseTarget(Target.String()) = %+v, want %+v", got, want)
	}
}

func TestParseTarget_Invalid(t *testing.T) {
	for _, raw := range []string{"", " ", "telegram:", ":42", "telegram::42", "telegram:42:", "matrix:!room:matrix.org:"} {
		t.Run(raw, func(t *testing.T) {
			if _, err := ParseTarget(raw, nil); err == nil {
				t.Fatalf("ParseTarget(%q) error = nil, want non-nil", raw)
			}
		})
	}
}
