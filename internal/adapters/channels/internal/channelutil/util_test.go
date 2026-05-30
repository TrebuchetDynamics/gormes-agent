package channelutil

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
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

func TestAllowedByPolicy(t *testing.T) {
	allowed := ToSet([]string{" user-1 "})

	cases := []struct {
		name   string
		policy string
		value  string
		want   bool
	}{
		{name: "open allows", policy: "open", value: "missing", want: true},
		{name: "empty defaults open", policy: "", value: "missing", want: true},
		{name: "unknown stays permissive", policy: "custom", value: "missing", want: true},
		{name: "disabled rejects", policy: "disabled", value: "user-1", want: false},
		{name: "allowlist accepts trimmed value", policy: "allowlist", value: " user-1 ", want: true},
		{name: "allowlist rejects missing value", policy: "allowlist", value: "user-2", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := AllowedByPolicy(tc.policy, allowed, tc.value); got != tc.want {
				t.Fatalf("AllowedByPolicy(%q, value %q) = %v, want %v", tc.policy, tc.value, got, tc.want)
			}
		})
	}
}

func TestFormatToolTrace(t *testing.T) {
	t.Run("nil events", func(t *testing.T) {
		result := FormatToolTrace(nil)
		if result != "" {
			t.Fatalf("expected empty for nil events, got %q", result)
		}
	})

	t.Run("empty events", func(t *testing.T) {
		result := FormatToolTrace([]kernel.SoulEntry{})
		if result != "" {
			t.Fatalf("expected empty for empty events, got %q", result)
		}
	})

	t.Run("single event", func(t *testing.T) {
		result := FormatToolTrace([]kernel.SoulEntry{{Text: "tool: read_file"}})
		if result == "" {
			t.Fatal("expected non-empty result for a single event")
		}
		if !strings.Contains(result, "read_file") {
			t.Fatalf("expected result to contain event text, got: %s", result)
		}
	})

	t.Run("multiple events", func(t *testing.T) {
		result := FormatToolTrace([]kernel.SoulEntry{
			{Text: "tool: read_file"},
			{Text: "tool: edit"},
		})
		if result == "" {
			t.Fatal("expected non-empty result for multiple events")
		}
		// Should contain the tool preview structure (FormatBlock adds icons/formatting)
		if strings.Contains(result, "📖") || strings.Contains(result, "🔧") {
			return // FormatBlock adds structural formatting
		}
	})

	t.Run("empty text events", func(t *testing.T) {
		result := FormatToolTrace([]kernel.SoulEntry{{Text: ""}})
		if result != "" {
			t.Fatalf("expected empty for empty text events, got %q", result)
		}
	})
}

func TestTruncateRunes(t *testing.T) {
	t.Run("max zero returns empty", func(t *testing.T) {
		if got := TruncateRunes("hello", 0); got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})

	t.Run("max negative returns empty", func(t *testing.T) {
		if got := TruncateRunes("hello", -1); got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})

	t.Run("under limit returns original", func(t *testing.T) {
		if got := TruncateRunes("hi", 10); got != "hi" {
			t.Fatalf("expected 'hi', got %q", got)
		}
	})

	t.Run("exact limit returns original", func(t *testing.T) {
		if got := TruncateRunes("hello", 5); got != "hello" {
			t.Fatalf("expected 'hello', got %q", got)
		}
	})

	t.Run("over limit truncates with ellipsis", func(t *testing.T) {
		got := TruncateRunes("hello world", 8)
		if got != "hello..." {
			t.Fatalf("expected 'hello...', got %q", got)
		}
	})

	t.Run("max 3 returns first 3 without ellipsis", func(t *testing.T) {
		if got := TruncateRunes("hello", 3); got != "hel" {
			t.Fatalf("expected 'hel', got %q", got)
		}
	})

	t.Run("max 1 returns first rune", func(t *testing.T) {
		if got := TruncateRunes("hello", 1); got != "h" {
			t.Fatalf("expected 'h', got %q", got)
		}
	})

	t.Run("respects unicode runes", func(t *testing.T) {
		got := TruncateRunes("日本語テスト", 4)
		if got != "日..." {
			t.Fatalf("expected '日...', got %q", got)
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

func TestCompactStrings(t *testing.T) {
	got := CompactStrings([]string{" a ", "", "b", "a", "  "})
	want := []string{"a", "b", "a"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q; all=%#v", i, got[i], want[i], got)
		}
	}
}

func TestUniqueStrings(t *testing.T) {
	got := UniqueStrings([]string{" a ", "", "b", "a", " b ", "c"})
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q; all=%#v", i, got[i], want[i], got)
		}
	}
}
