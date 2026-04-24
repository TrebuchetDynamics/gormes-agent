package tuigateway

import (
	"testing"
	"time"
)

func ptrDuration(d time.Duration) *time.Duration { return &d }

func TestFormatToolDuration(t *testing.T) {
	cases := []struct {
		name string
		in   *time.Duration
		want string
	}{
		{name: "nil returns empty", in: nil, want: ""},
		{name: "zero formats as 0.0s", in: ptrDuration(0), want: "0.0s"},
		{name: "sub-second one decimal", in: ptrDuration(500 * time.Millisecond), want: "0.5s"},
		{name: "just under ten seconds one decimal", in: ptrDuration(9900 * time.Millisecond), want: "9.9s"},
		{name: "ten seconds flat rounds to integer", in: ptrDuration(10 * time.Second), want: "10s"},
		{name: "mid-range rounds up", in: ptrDuration(10600 * time.Millisecond), want: "11s"},
		{name: "mid-range rounds at threshold", in: ptrDuration(45600 * time.Millisecond), want: "46s"},
		{name: "exactly one minute drops seconds field", in: ptrDuration(60 * time.Second), want: "1m"},
		{name: "one minute five seconds", in: ptrDuration(65 * time.Second), want: "1m 5s"},
		{name: "two minutes five seconds", in: ptrDuration(125 * time.Second), want: "2m 5s"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := FormatToolDuration(c.in)
			if got != c.want {
				t.Fatalf("FormatToolDuration(%v) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestCountList(t *testing.T) {
	t.Run("nil root returns not-ok", func(t *testing.T) {
		if n, ok := CountList(nil, "a"); ok || n != 0 {
			t.Fatalf("CountList(nil, a) = (%d, %v), want (0, false)", n, ok)
		}
	})

	t.Run("missing key returns not-ok", func(t *testing.T) {
		if n, ok := CountList(map[string]any{}, "missing"); ok || n != 0 {
			t.Fatalf("want (0, false), got (%d, %v)", n, ok)
		}
	})

	t.Run("leaf list returns length", func(t *testing.T) {
		obj := map[string]any{"items": []any{1, 2, 3}}
		n, ok := CountList(obj, "items")
		if !ok || n != 3 {
			t.Fatalf("want (3, true), got (%d, %v)", n, ok)
		}
	})

	t.Run("nested list returns length", func(t *testing.T) {
		obj := map[string]any{
			"data": map[string]any{"web": []any{"a", "b"}},
		}
		n, ok := CountList(obj, "data", "web")
		if !ok || n != 2 {
			t.Fatalf("want (2, true), got (%d, %v)", n, ok)
		}
	})

	t.Run("non-list leaf returns not-ok", func(t *testing.T) {
		obj := map[string]any{"items": "not-a-list"}
		if n, ok := CountList(obj, "items"); ok || n != 0 {
			t.Fatalf("want (0, false), got (%d, %v)", n, ok)
		}
	})

	t.Run("non-map mid-path returns not-ok", func(t *testing.T) {
		obj := map[string]any{"a": "scalar"}
		if n, ok := CountList(obj, "a", "b"); ok || n != 0 {
			t.Fatalf("want (0, false), got (%d, %v)", n, ok)
		}
	})

	t.Run("empty list returns zero-true", func(t *testing.T) {
		obj := map[string]any{"items": []any{}}
		n, ok := CountList(obj, "items")
		if !ok || n != 0 {
			t.Fatalf("want (0, true), got (%d, %v)", n, ok)
		}
	})
}

func TestToolSummary(t *testing.T) {
	cases := []struct {
		name     string
		tool     string
		result   string
		duration *time.Duration
		want     string
	}{
		{
			name:   "web_search singular",
			tool:   "web_search",
			result: `{"data":{"web":[{"url":"x"}]}}`,
			want:   "Did 1 search",
		},
		{
			name:   "web_search plural",
			tool:   "web_search",
			result: `{"data":{"web":[1,2,3]}}`,
			want:   "Did 3 searches",
		},
		{
			name:     "web_search plural with duration",
			tool:     "web_search",
			result:   `{"data":{"web":[1,2,3]}}`,
			duration: ptrDuration(2 * time.Second),
			want:     "Did 3 searches in 2.0s",
		},
		{
			name:   "web_extract singular from results",
			tool:   "web_extract",
			result: `{"results":[{"a":1}]}`,
			want:   "Extracted 1 page",
		},
		{
			name:     "web_extract plural with duration",
			tool:     "web_extract",
			result:   `{"results":[1,2]}`,
			duration: ptrDuration(3 * time.Second),
			want:     "Extracted 2 pages in 3.0s",
		},
		{
			name:   "web_extract falls back to data.results when top-level empty",
			tool:   "web_extract",
			result: `{"results":[],"data":{"results":[1,2,3,4]}}`,
			want:   "Extracted 4 pages",
		},
		{
			name:   "web_extract falls back to data.results when top-level missing",
			tool:   "web_extract",
			result: `{"data":{"results":[1]}}`,
			want:   "Extracted 1 page",
		},
		{
			name:     "unknown tool with duration yields Completed suffix",
			tool:     "mystery",
			result:   `{"ignored":true}`,
			duration: ptrDuration(5500 * time.Millisecond),
			want:     "Completed in 5.5s",
		},
		{
			name:   "unknown tool without duration yields empty",
			tool:   "mystery",
			result: `{"ignored":true}`,
			want:   "",
		},
		{
			name:     "invalid JSON with duration yields Completed suffix",
			tool:     "web_search",
			result:   `not-json`,
			duration: ptrDuration(time.Second),
			want:     "Completed in 1.0s",
		},
		{
			name:   "invalid JSON without duration yields empty",
			tool:   "web_search",
			result: `not-json`,
			want:   "",
		},
		{
			name:   "web_search missing nested data yields empty",
			tool:   "web_search",
			result: `{"data":{"other":[1,2]}}`,
			want:   "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ToolSummary(c.tool, c.result, c.duration)
			if got != c.want {
				t.Fatalf("ToolSummary(%q, %q, %v) = %q, want %q", c.tool, c.result, c.duration, got, c.want)
			}
		})
	}
}
