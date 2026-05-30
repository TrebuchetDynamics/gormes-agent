package channelutil

import (
	"testing"
)

func TestToSet(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		result := ToSet(nil)
		if len(result) != 0 {
			t.Fatalf("expected empty set, got %d entries", len(result))
		}
	})

	t.Run("skips empty strings", func(t *testing.T) {
		result := ToSet([]string{"a", "", "b", "  "})
		if len(result) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(result))
		}
		if _, ok := result["a"]; !ok {
			t.Fatal("missing 'a'")
		}
		if _, ok := result["b"]; !ok {
			t.Fatal("missing 'b'")
		}
	})

	t.Run("trims whitespace", func(t *testing.T) {
		result := ToSet([]string{"  hello ", "world  "})
		if _, ok := result["hello"]; !ok {
			t.Fatal("missing 'hello'")
		}
		if _, ok := result["world"]; !ok {
			t.Fatal("missing 'world'")
		}
	})

	t.Run("deduplicates", func(t *testing.T) {
		result := ToSet([]string{"x", "x", "y"})
		if len(result) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(result))
		}
	})
}

func TestStripLeadingMentions(t *testing.T) {
	t.Run("no mentions", func(t *testing.T) {
		got := StripLeadingMentions("hello world")
		if got != "hello world" {
			t.Fatalf("expected 'hello world', got %q", got)
		}
	})

	t.Run("strips leading mentions", func(t *testing.T) {
		got := StripLeadingMentions("@bot hello world")
		if got != "hello world" {
			t.Fatalf("expected 'hello world', got %q", got)
		}
	})

	t.Run("strips multiple leading mentions", func(t *testing.T) {
		got := StripLeadingMentions("@bot @user hello world")
		if got != "hello world" {
			t.Fatalf("expected 'hello world', got %q", got)
		}
	})

	t.Run("mention in middle preserved", func(t *testing.T) {
		got := StripLeadingMentions("hello @bot world")
		if got != "hello @bot world" {
			t.Fatalf("expected 'hello @bot world', got %q", got)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		got := StripLeadingMentions("")
		if got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})

	t.Run("only mentions", func(t *testing.T) {
		got := StripLeadingMentions("@bot @user")
		if got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})
}

func TestNormalizedPolicy(t *testing.T) {
	t.Run("empty returns open", func(t *testing.T) {
		if got := NormalizedPolicy(""); got != "open" {
			t.Fatalf("expected 'open', got %q", got)
		}
	})

	t.Run("lowercases", func(t *testing.T) {
		if got := NormalizedPolicy("ALLOWLIST"); got != "allowlist" {
			t.Fatalf("expected 'allowlist', got %q", got)
		}
	})

	t.Run("trims whitespace", func(t *testing.T) {
		if got := NormalizedPolicy("  disabled  "); got != "disabled" {
			t.Fatalf("expected 'disabled', got %q", got)
		}
	})

	t.Run("passes through known values", func(t *testing.T) {
		for _, v := range []string{"open", "disabled", "allowlist"} {
			if got := NormalizedPolicy(v); got != v {
				t.Fatalf("expected %q, got %q", v, got)
			}
		}
	})
}

func TestFirstNonEmpty(t *testing.T) {
	t.Run("returns first non-empty trimmed", func(t *testing.T) {
		got := FirstNonEmpty("", "  ", "hello", "world")
		if got != "hello" {
			t.Fatalf("expected 'hello', got %q", got)
		}
	})

	t.Run("all empty returns empty", func(t *testing.T) {
		got := FirstNonEmpty("", "  ")
		if got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})

	t.Run("no args returns empty", func(t *testing.T) {
		got := FirstNonEmpty()
		if got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})

	t.Run("trims result", func(t *testing.T) {
		got := FirstNonEmpty("  spaced  ")
		if got != "spaced" {
			t.Fatalf("expected 'spaced', got %q", got)
		}
	})

	t.Run("first non-empty wins even if later is longer", func(t *testing.T) {
		got := FirstNonEmpty("", "a", "bb")
		if got != "a" {
			t.Fatalf("expected 'a', got %q", got)
		}
	})
}
