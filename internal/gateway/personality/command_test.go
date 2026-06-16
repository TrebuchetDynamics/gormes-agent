package personality

import (
	"strings"
	"testing"
)

func TestParseArgRejectsMalformedSlashCommand(t *testing.T) {
	for _, raw := range []string{"/personality@bad-name pirate", "//personality pirate", "／personality@bot.name pirate"} {
		if got := ParseArg(raw); got != "" {
			t.Fatalf("ParseArg(%q) = %q, want malformed slash command rejected", raw, got)
		}
	}
}

func TestParseArg(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "empty", text: "", want: ""},
		{name: "bare", text: "/personality", want: ""},
		{name: "argument", text: "/personality pirate", want: "pirate"},
		{name: "bot mention", text: "/personality@GormesBot pirate", want: "pirate"},
		{name: "already payload", text: "pirate", want: "pirate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseArg(tt.text); got != tt.want {
				t.Fatalf("ParseArg(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

func TestRenderListSortsAndTruncates(t *testing.T) {
	got := RenderList("pirate", map[string]string{
		"zen":    "calm",
		"pirate": "ahoy there matey",
	}, 6)
	wantLines := []string{
		"**Personalities:**",
		"Active: **pirate**",
		"  • `/personality pirate` — ahoy t…",
		"  • `/personality zen` — calm",
		"",
		"Usage: `/personality <name>` or `/personality none` to clear",
	}
	want := strings.Join(wantLines, "\n")
	if got != want {
		t.Fatalf("RenderList() =\n%s\nwant\n%s", got, want)
	}
}

func TestRenderListClampsNegativeDescriptionLimit(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RenderList panicked with negative description limit: %v", r)
		}
	}()
	got := RenderList("", map[string]string{"pirate": "ahoy"}, -1)
	if !strings.Contains(got, "`/personality pirate` — …") {
		t.Fatalf("RenderList with negative desc limit =\n%s\nwant clamped ellipsis", got)
	}
}

func TestRenderListAndUnknownRedactAuthorizationValues(t *testing.T) {
	got := RenderList("authorization=Bearer plain-secret-token", map[string]string{
		"auth-mode": "uses authorization=Bearer plain-secret-token",
	}, 100)
	for _, forbidden := range []string{"plain-secret-token", "authorization", "Bearer", "bearer"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("RenderList leaked authorization value %q in:\n%s", forbidden, got)
		}
	}
	if !strings.Contains(got, "[redacted]") {
		t.Fatalf("RenderList missing redaction marker:\n%s", got)
	}

	unknown := RenderUnknown("authorization=Bearer plain-secret-token", map[string]string{"authorization=Bearer plain-secret-token": ""})
	for _, forbidden := range []string{"plain-secret-token", "authorization", "Bearer", "bearer"} {
		if strings.Contains(unknown, forbidden) {
			t.Fatalf("RenderUnknown leaked authorization value %q in %q", forbidden, unknown)
		}
	}
	if !strings.Contains(unknown, "[redacted]") {
		t.Fatalf("RenderUnknown missing redaction marker: %q", unknown)
	}
}

func TestRenderListAndUnknownRedactSecretLikeValues(t *testing.T) {
	got := RenderList("api_key=plain-secret-token", map[string]string{
		"secret=plain-secret-token": "password=plain-secret-token",
	}, 100)
	for _, forbidden := range []string{"plain-secret-token", "api_key", "secret=", "password="} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("RenderList leaked secret-like value %q in:\n%s", forbidden, got)
		}
	}
	if !strings.Contains(got, "[redacted]") {
		t.Fatalf("RenderList missing redaction marker:\n%s", got)
	}

	unknown := RenderUnknown("token=plain-secret-token", map[string]string{"api_key=plain-secret-token": ""})
	for _, forbidden := range []string{"plain-secret-token", "api_key", "token="} {
		if strings.Contains(unknown, forbidden) {
			t.Fatalf("RenderUnknown leaked secret-like value %q in %q", forbidden, unknown)
		}
	}
	if !strings.Contains(unknown, "[redacted]") {
		t.Fatalf("RenderUnknown missing redaction marker: %q", unknown)
	}
}

func TestRenderListAndUnknownSanitizeConfiguredNames(t *testing.T) {
	got := RenderList("pirate\n**Injected:**", map[string]string{
		"zen`mode":  "calm\nsecond line",
		"bad\nname": "desc",
	}, 100)
	for _, forbidden := range []string{"**Injected:**", "calm\nsecond", "bad\nname", "zen`mode"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("RenderList leaked unsanitized value %q in:\n%s", forbidden, got)
		}
	}
	for _, want := range []string{
		"Active: **pirate ''Injected:''**",
		"`/personality bad name` — desc",
		"`/personality zen'mode` — calm second line",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderList missing sanitized value %q in:\n%s", want, got)
		}
	}

	unknown := RenderUnknown("bad\nname", map[string]string{"zen`mode": "", "pirate\nship": ""})
	for _, forbidden := range []string{"bad\nname", "zen`mode", "pirate\nship"} {
		if strings.Contains(unknown, forbidden) {
			t.Fatalf("RenderUnknown leaked unsanitized value %q in %q", forbidden, unknown)
		}
	}
	if want := "Unknown personality \"bad name\". Available: pirate ship, zen'mode"; unknown != want {
		t.Fatalf("RenderUnknown() = %q, want %q", unknown, want)
	}
}

func TestRenderUnknownIncludesSortedHint(t *testing.T) {
	got := RenderUnknown("wizard", map[string]string{"zen": "", "pirate": ""})
	want := "Unknown personality \"wizard\". Available: pirate, zen"
	if got != want {
		t.Fatalf("RenderUnknown() = %q, want %q", got, want)
	}
}
