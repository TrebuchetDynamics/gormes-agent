package delivery

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

func TestParseTarget_Invalid(t *testing.T) {
	for _, raw := range []string{"", " ", "telegram:", ":42", "telegram::42", "telegram:42:"} {
		t.Run(raw, func(t *testing.T) {
			if _, err := ParseTarget(raw, nil); err == nil {
				t.Fatalf("ParseTarget(%q) error = nil, want non-nil", raw)
			}
		})
	}
}
