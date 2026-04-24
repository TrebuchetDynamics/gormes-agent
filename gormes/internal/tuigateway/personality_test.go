package tuigateway

import (
	"errors"
	"strings"
	"testing"
)

func TestPersonalityRender(t *testing.T) {
	cases := []struct {
		name string
		in   Personality
		want string
	}{
		{
			name: "empty returns empty string",
			in:   Personality{},
			want: "",
		},
		{
			name: "only system prompt",
			in:   Personality{SystemPrompt: "You are a helpful assistant."},
			want: "You are a helpful assistant.",
		},
		{
			name: "prompt with tone",
			in:   Personality{SystemPrompt: "You are Hermes.", Tone: "warm"},
			want: "You are Hermes.\nTone: warm",
		},
		{
			name: "prompt with style",
			in:   Personality{SystemPrompt: "You are Hermes.", Style: "terse"},
			want: "You are Hermes.\nStyle: terse",
		},
		{
			name: "prompt with tone and style",
			in:   Personality{SystemPrompt: "You are Hermes.", Tone: "warm", Style: "terse"},
			want: "You are Hermes.\nTone: warm\nStyle: terse",
		},
		{
			name: "tone-only drops missing system prompt",
			in:   Personality{Tone: "warm"},
			want: "Tone: warm",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in.Render()
			if got != tc.want {
				t.Fatalf("Render() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPersonalitySetValidate_Unset(t *testing.T) {
	// Raw values that mean "no personality" must return empty (name, prompt) and nil error
	// regardless of the configured set, so operators can clear the setting with any of them.
	set := PersonalitySet{
		"hermes": Personality{SystemPrompt: "You are Hermes."},
	}
	for _, raw := range []string{"", "   ", "none", "default", "neutral", "None", "DEFAULT", "  Neutral  "} {
		t.Run("raw="+raw, func(t *testing.T) {
			name, prompt, err := set.Validate(raw)
			if err != nil {
				t.Fatalf("Validate(%q) err = %v, want nil", raw, err)
			}
			if name != "" {
				t.Fatalf("Validate(%q) name = %q, want empty", raw, name)
			}
			if prompt != "" {
				t.Fatalf("Validate(%q) prompt = %q, want empty", raw, prompt)
			}
		})
	}
}

func TestPersonalitySetValidate_Known(t *testing.T) {
	set := PersonalitySet{
		"hermes": Personality{SystemPrompt: "You are Hermes.", Tone: "warm"},
		"oracle": Personality{SystemPrompt: "You are the Oracle."},
	}

	name, prompt, err := set.Validate("  Hermes  ")
	if err != nil {
		t.Fatalf("Validate(Hermes) err = %v, want nil", err)
	}
	if name != "hermes" {
		t.Fatalf("Validate(Hermes) name = %q, want hermes", name)
	}
	want := "You are Hermes.\nTone: warm"
	if prompt != want {
		t.Fatalf("Validate(Hermes) prompt = %q, want %q", prompt, want)
	}
}

func TestPersonalitySetValidate_Unknown_ListsAvailable(t *testing.T) {
	set := PersonalitySet{
		"oracle": Personality{SystemPrompt: "You are the Oracle."},
		"hermes": Personality{SystemPrompt: "You are Hermes."},
		"coyote": Personality{SystemPrompt: "You are Coyote."},
	}

	_, _, err := set.Validate("gandalf")
	if err == nil {
		t.Fatalf("Validate(gandalf) err = nil, want non-nil")
	}
	if !errors.Is(err, ErrUnknownPersonality) {
		t.Fatalf("Validate(gandalf) err = %v, want wraps ErrUnknownPersonality", err)
	}
	// Mirrors the Python server.py error envelope so operators see a stable message.
	msg := err.Error()
	if !strings.Contains(msg, "Unknown personality: `gandalf`.") {
		t.Fatalf("err message %q missing raw-name echo", msg)
	}
	// Names must be sorted alphabetically so the error is deterministic for tests and humans.
	if !strings.Contains(msg, "Available: `none`, `coyote`, `hermes`, `oracle`") {
		t.Fatalf("err message %q missing sorted available list", msg)
	}
}

func TestPersonalitySetValidate_Unknown_EmptySet(t *testing.T) {
	set := PersonalitySet{}

	_, _, err := set.Validate("gandalf")
	if err == nil {
		t.Fatalf("Validate(gandalf) err = nil, want non-nil")
	}
	if !errors.Is(err, ErrUnknownPersonality) {
		t.Fatalf("Validate(gandalf) err = %v, want wraps ErrUnknownPersonality", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "Unknown personality: `gandalf`.") {
		t.Fatalf("err message %q missing raw-name echo", msg)
	}
	if !strings.Contains(msg, "No personalities configured.") {
		t.Fatalf("err message %q missing empty-set note, got: %s", msg, msg)
	}
	if strings.Contains(msg, "Available:") {
		t.Fatalf("err message %q must not list Available when set is empty", msg)
	}
}

func TestPersonalitySetValidate_NilSetTreatedAsEmpty(t *testing.T) {
	var set PersonalitySet
	_, _, err := set.Validate("gandalf")
	if err == nil {
		t.Fatalf("Validate(gandalf) on nil set err = nil, want non-nil")
	}
	if !errors.Is(err, ErrUnknownPersonality) {
		t.Fatalf("err = %v, want wraps ErrUnknownPersonality", err)
	}
	if !strings.Contains(err.Error(), "No personalities configured.") {
		t.Fatalf("nil set err must carry empty-set note, got: %s", err.Error())
	}
}
