package setupchoice

import "testing"

func TestNormalizeAnswerResolvesDefaultOrdinalLabelAndAlias(t *testing.T) {
	options := []Choice{
		{ID: "local", Label: "Local loopback only", Aliases: []string{"loopback"}},
		{ID: "tailscale", Label: "Tailscale VPN"},
	}
	for _, tt := range []struct {
		name   string
		answer string
		want   string
	}{
		{name: "blank default", answer: "", want: "tailscale"},
		{name: "ordinal", answer: "1", want: "local"},
		{name: "label", answer: "Tailscale VPN", want: "tailscale"},
		{name: "alias", answer: "loopback", want: "local"},
		{name: "unknown normalizes", answer: "Custom-Thing", want: "custom_thing"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeAnswer(tt.answer, options, "tailscale"); got != tt.want {
				t.Fatalf("NormalizeAnswer(%q) = %q, want %q", tt.answer, got, tt.want)
			}
		})
	}
}

func TestYesNoParsesDefaultsAndAliases(t *testing.T) {
	for _, tt := range []struct {
		value       string
		defaultBool bool
		want        bool
		wantOK      bool
	}{
		{value: "", defaultBool: true, want: true, wantOK: true},
		{value: "off", want: false, wantOK: true},
		{value: "Y", want: true, wantOK: true},
		{value: "maybe", want: false, wantOK: false},
	} {
		got, ok := YesNo(tt.value, tt.defaultBool)
		if got != tt.want || ok != tt.wantOK {
			t.Fatalf("YesNo(%q,%v) = %v,%v; want %v,%v", tt.value, tt.defaultBool, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestStripInputNoiseRemovesANSIAndControlBytes(t *testing.T) {
	if got := StripInputNoise("\x1b[31mred\x1b[0m\n"); got != "red" {
		t.Fatalf("StripInputNoise = %q, want red", got)
	}
}
