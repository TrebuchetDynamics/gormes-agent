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
			name: "matrix room and opaque thread id without server delimiter",
			raw:  "matrix:!room:$thread",
			want: Target{Platform: "matrix", ChatID: "!room", ThreadID: "$thread", IsExplicit: true},
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

func TestParseTargetRejectsControlCharactersInOriginSource(t *testing.T) {
	for _, origin := range []OriginSource{
		{Platform: "telegram\nadmin", ChatID: "42"},
		{Platform: "telegram", ChatID: "42\nadmin"},
		{Platform: "telegram", ChatID: "42", ThreadID: "thread\nadmin"},
	} {
		if target, err := ParseTarget("origin", &origin); err == nil {
			t.Fatalf("ParseTarget(origin, %+v) = %+v, nil; want invalid origin error", origin, target)
		}
	}
}

func TestParseTargetRejectsControlCharacters(t *testing.T) {
	for _, raw := range []string{"telegram\nadmin:42", "telegram:42\nadmin", "telegram:42:thread\nadmin"} {
		t.Run(raw, func(t *testing.T) {
			if got, err := ParseTarget(raw, nil); err == nil {
				t.Fatalf("ParseTarget(%q) = %+v, nil; want control-character target rejected", raw, got)
			}
		})
	}
}

func TestParseTargetRejectsHiddenFormattingRunes(t *testing.T) {
	for _, raw := range []string{"tele\u200bgram:42", "telegram:42\u200badmin", "telegram:42:thread\u202eadmin"} {
		t.Run(raw, func(t *testing.T) {
			if got, err := ParseTarget(raw, nil); err == nil {
				t.Fatalf("ParseTarget(%q) = %+v, nil; want hidden-formatting target rejected", raw, got)
			}
		})
	}
}

func TestParseTargetOriginWithMissingChatFallsBackToLocal(t *testing.T) {
	got, err := ParseTarget("origin", &OriginSource{Platform: "telegram", ChatID: " ", ThreadID: "99"})
	if err != nil {
		t.Fatalf("ParseTarget(origin): %v", err)
	}
	want := Target{Platform: "local", IsOrigin: true}
	if got != want {
		t.Fatalf("ParseTarget(origin with missing chat) = %+v, want %+v", got, want)
	}
}

func TestParseTargetOriginWithEmptySourceFallsBackToLocal(t *testing.T) {
	got, err := ParseTarget("origin", &OriginSource{})
	if err != nil {
		t.Fatalf("ParseTarget(origin): %v", err)
	}
	want := Target{Platform: "local", IsOrigin: true}
	if got != want {
		t.Fatalf("ParseTarget(origin with empty source) = %+v, want %+v", got, want)
	}
}

func TestParseTargetNormalizesLengthEncodedIDs(t *testing.T) {
	got, err := ParseTarget("bluebubbles:7: chat x:9: thread x", nil)
	if err != nil {
		t.Fatalf("ParseTarget length-encoded IDs: %v", err)
	}
	want := Target{Platform: "bluebubbles", ChatID: "chat x", ThreadID: "thread x", IsExplicit: true}
	if got != want {
		t.Fatalf("ParseTarget length-encoded IDs = %+v, want normalized %+v", got, want)
	}
}

func TestParseTargetRejectsLengthEncodedTargetSplittingUTF8(t *testing.T) {
	if got, err := ParseTarget("bluebubbles:1:é:x", nil); err == nil {
		t.Fatalf("ParseTarget accepted length-encoded target with split UTF-8: %+v", got)
	}
}

func TestParseTargetRejectsSignedLengthEncodedSegments(t *testing.T) {
	for _, raw := range []string{
		"telegram:+5:a:b:c",
		"telegram:3:a:b:+3:t:u",
	} {
		t.Run(raw, func(t *testing.T) {
			if target, err := ParseTarget(raw, nil); err == nil {
				t.Fatalf("ParseTarget(%q) = %+v, nil; want signed length segment rejected", raw, target)
			}
		})
	}
}

func TestParseTargetRejectsLocalWithChatID(t *testing.T) {
	if got, err := ParseTarget("local:42", nil); err == nil {
		t.Fatalf("ParseTarget(local:42) = %+v, nil; want invalid local target", got)
	}
}

func TestTargetStringRoundTripsMatrixThreadWithoutEventPrefix(t *testing.T) {
	want := Target{Platform: "matrix", ChatID: "!room:matrix.org", ThreadID: "thread:1", IsExplicit: true}
	got, err := ParseTarget(want.String(), nil)
	if err != nil {
		t.Fatalf("ParseTarget(%q): %v", want.String(), err)
	}
	if got != want {
		t.Fatalf("ParseTarget(Target.String()) = %+v, want %+v", got, want)
	}
}

func TestTargetStringRoundTripsSimplexThreadedGroupTarget(t *testing.T) {
	want := Target{Platform: "simplex", ChatID: "group:ops", ThreadID: "thread:1", IsExplicit: true}
	got, err := ParseTarget(want.String(), nil)
	if err != nil {
		t.Fatalf("ParseTarget(%q): %v", want.String(), err)
	}
	if got != want {
		t.Fatalf("ParseTarget(Target.String()) = %+v, want %+v", got, want)
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
