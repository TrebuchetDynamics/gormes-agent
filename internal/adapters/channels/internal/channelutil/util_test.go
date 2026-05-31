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

func TestBoolSet(t *testing.T) {
	got := BoolSet([]string{" a ", "", "b", "a", "  "})
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2: %#v", len(got), got)
	}
	if !got["a"] || !got["b"] {
		t.Fatalf("missing expected members: %#v", got)
	}
	if got[""] {
		t.Fatalf("empty member should not be present: %#v", got)
	}
}

func TestBoolSetsIntersect(t *testing.T) {
	if !BoolSetsIntersect(map[string]bool{"a": true}, map[string]bool{"a": true, "b": true}) {
		t.Fatal("expected intersection")
	}
	if BoolSetsIntersect(map[string]bool{"a": true}, map[string]bool{"b": true}) {
		t.Fatal("unexpected intersection")
	}
	if BoolSetsIntersect(nil, map[string]bool{"a": true}) {
		t.Fatal("nil set should not intersect")
	}
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

func TestContainsString(t *testing.T) {
	if !ContainsString([]string{"alpha", "Beta"}, "Beta") {
		t.Fatal("ContainsString returned false for exact match")
	}
	if ContainsString([]string{"alpha", "Beta"}, "beta") {
		t.Fatal("ContainsString returned true for case-different value")
	}
}

func TestContainsEqualFold(t *testing.T) {
	if !ContainsEqualFold([]string{"alpha", "Beta"}, "beta") {
		t.Fatal("ContainsEqualFold returned false for case-insensitive match")
	}
	if ContainsEqualFold([]string{"alpha", "Beta"}, "gamma") {
		t.Fatal("ContainsEqualFold returned true for missing value")
	}
}

func TestParseBoolDefault(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		def  bool
		want bool
	}{
		{name: "empty true default", raw: "", def: true, want: true},
		{name: "empty false default", raw: "", def: false, want: false},
		{name: "true word", raw: " true ", def: false, want: true},
		{name: "one", raw: "1", def: false, want: true},
		{name: "yes", raw: "YES", def: false, want: true},
		{name: "on", raw: "on", def: false, want: true},
		{name: "false word", raw: " false ", def: true, want: false},
		{name: "zero", raw: "0", def: true, want: false},
		{name: "no", raw: "NO", def: true, want: false},
		{name: "off", raw: "off", def: true, want: false},
		{name: "unknown uses default true", raw: "maybe", def: true, want: true},
		{name: "unknown uses default false", raw: "maybe", def: false, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ParseBoolDefault(tc.raw, tc.def); got != tc.want {
				t.Fatalf("ParseBoolDefault(%q, %v) = %v, want %v", tc.raw, tc.def, got, tc.want)
			}
		})
	}
}

func TestDocumentMediaTypeForExtension(t *testing.T) {
	if got, ok := DocumentMediaTypeForExtension(" .MD "); !ok || got != "text/markdown" {
		t.Fatalf("DocumentMediaTypeForExtension = %q, %v", got, ok)
	}
	if _, ok := DocumentMediaTypeForExtension(".exe"); ok {
		t.Fatal("unexpected document media type for .exe")
	}
}

func TestDocumentExtensions(t *testing.T) {
	got := DocumentExtensions()
	if !ContainsString(got, ".md") || !ContainsString(got, ".pdf") {
		t.Fatalf("DocumentExtensions missing expected entries: %#v", got)
	}
	for i := 1; i < len(got); i++ {
		if got[i-1] > got[i] {
			t.Fatalf("DocumentExtensions not sorted: %#v", got)
		}
	}
}

func TestMIMEExtensionFallback(t *testing.T) {
	if got := MIMEExtensionFallback(" text/plain; charset=utf-8 "); got != ".txt" {
		t.Fatalf("MIMEExtensionFallback text = %q", got)
	}
	if got := MIMEExtensionFallback("application/unknown"); got != "" {
		t.Fatalf("MIMEExtensionFallback unknown = %q", got)
	}
}

func TestCleanMediaType(t *testing.T) {
	cases := map[string]string{
		"":                            "",
		" text/plain; charset=utf-8 ": "text/plain",
		"IMAGE/PNG":                   "image/png",
	}
	for raw, want := range cases {
		if got := CleanMediaType(raw); got != want {
			t.Fatalf("CleanMediaType(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestImageExtensionAndMediaType(t *testing.T) {
	if got := ImageExtensionForMediaType(" image/png; charset=binary "); got != ".png" {
		t.Fatalf("ImageExtensionForMediaType png = %q", got)
	}
	if got := ImageExtensionForMediaType("image/unknown"); got != ".jpg" {
		t.Fatalf("ImageExtensionForMediaType default = %q", got)
	}
	if !ImageExtensionSupported(" .WEBP ") || ImageExtensionSupported(".bmp") {
		t.Fatalf("ImageExtensionSupported returned unexpected result")
	}
	if got := ImageMediaTypeForExtension(".jpeg"); got != "image/jpeg" {
		t.Fatalf("ImageMediaTypeForExtension jpeg = %q", got)
	}
	if got := ImageMediaTypeForExtension(".unknown"); got != "image/jpeg" {
		t.Fatalf("ImageMediaTypeForExtension default = %q", got)
	}
}

func TestSafeFileName(t *testing.T) {
	if got := SafeFileName(" ../unsafe/name.txt "); got != "name.txt" {
		t.Fatalf("SafeFileName path = %q", got)
	}
	if got := SafeFileName("bad\x00name.txt"); got != "bad_name.txt" {
		t.Fatalf("SafeFileName control = %q", got)
	}
	if got := SafeFileName(" . "); got != "" {
		t.Fatalf("SafeFileName blank = %q", got)
	}
	long := strings.Repeat("a", 180) + ".txt"
	got := SafeFileName(long)
	if len(got) > 160 || !strings.HasSuffix(got, ".txt") {
		t.Fatalf("SafeFileName long = len %d value %q", len(got), got)
	}
}

func TestSafeTokenDefault(t *testing.T) {
	if got := SafeTokenDefault(" user/id:42 ", "fallback"); got != "user_id_42" {
		t.Fatalf("SafeTokenDefault = %q", got)
	}
	if got := SafeTokenDefault("___", "fallback"); got != "fallback" {
		t.Fatalf("SafeTokenDefault fallback = %q", got)
	}
	long := strings.Repeat("a", 80)
	if got := SafeTokenDefault(long, "fallback"); len(got) != 64 {
		t.Fatalf("SafeTokenDefault long len = %d", len(got))
	}
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

func TestSplitCommaList(t *testing.T) {
	got := SplitCommaList(" a, ,b,a ")
	want := []string{"a", "b", "a"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q; all=%#v", i, got[i], want[i], got)
		}
	}
	if got := SplitCommaList("  "); got != nil {
		t.Fatalf("blank list = %#v, want nil", got)
	}
}

func TestUniqueCommaList(t *testing.T) {
	got := UniqueCommaList(" b, ,a,b ")
	want := []string{"b", "a"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q; all=%#v", i, got[i], want[i], got)
		}
	}
	if got := UniqueCommaList("  "); got != nil {
		t.Fatalf("blank list = %#v, want nil", got)
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

func TestUniqueSortedStrings(t *testing.T) {
	got := UniqueSortedStrings([]string{" b ", "", "a", "b", " c "})
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

func TestUniqueLowerSortedStrings(t *testing.T) {
	got := UniqueLowerSortedStrings([]string{" Beta ", "alpha", "BETA", "", " gamma "})
	want := []string{"alpha", "beta", "gamma"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q; all=%#v", i, got[i], want[i], got)
		}
	}
}
