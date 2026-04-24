package tuigateway

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

// FormatToolDuration renders a human-readable duration for tool progress UI.
// It mirrors the upstream `_fmt_tool_duration` helper in
// `tui_gateway/server.py` (lines 614-622): a nil input returns the empty
// string (the caller did not record a start time), durations under ten
// seconds render with a single decimal ("0.5s", "9.9s"), durations under a
// minute round to the nearest whole second, and durations at or above a
// minute split into "Xm Ys" with the seconds field omitted when zero.
func FormatToolDuration(d *time.Duration) string {
	if d == nil {
		return ""
	}
	seconds := d.Seconds()
	if seconds < 10 {
		return fmt.Sprintf("%.1fs", seconds)
	}
	rounded := int(math.Round(seconds))
	if rounded < 60 {
		return fmt.Sprintf("%ds", rounded)
	}
	mins := rounded / 60
	secs := rounded % 60
	if secs == 0 {
		return fmt.Sprintf("%dm", mins)
	}
	return fmt.Sprintf("%dm %ds", mins, secs)
}

// CountList walks the nested map at v along path and returns the length of
// the terminal list. It mirrors `_count_list` in `tui_gateway/server.py`
// (lines 625-631): any non-map encountered mid-path, a missing key, or a
// non-list leaf yields (0, false); an empty list yields (0, true) so callers
// can distinguish "known zero" from "not a list" when cascading alternatives.
func CountList(v any, path ...string) (int, bool) {
	cur := v
	for _, key := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return 0, false
		}
		cur = m[key]
	}
	list, ok := cur.([]any)
	if !ok {
		return 0, false
	}
	return len(list), true
}

// ToolSummary produces the short progress line rendered after a tool
// completes. It mirrors `_tool_summary` in `tui_gateway/server.py` (lines
// 634-654): web_search results (data.web list) render as
// "Did N search(es)"; web_extract results (results or data.results list)
// render as "Extracted N page(s)"; other tools (or decode failures) fall
// back to "Completed" only when a duration is available. A trailing
// " in <FormatToolDuration>" is appended whenever the duration is known.
// An empty string means "no summary worth rendering"—matching the Python
// `None` return used to suppress the progress emission entirely.
func ToolSummary(name, result string, d *time.Duration) string {
	var data any
	_ = json.Unmarshal([]byte(result), &data)

	dur := FormatToolDuration(d)
	suffix := ""
	if dur != "" {
		suffix = " in " + dur
	}

	text := ""
	switch name {
	case "web_search":
		if n, ok := CountList(data, "data", "web"); ok {
			text = fmt.Sprintf("Did %d %s", n, pluralize(n, "search", "searches"))
		}
	case "web_extract":
		if n, ok := firstCount(data, []string{"results"}, []string{"data", "results"}); ok {
			text = fmt.Sprintf("Extracted %d %s", n, pluralize(n, "page", "pages"))
		}
	}

	if text == "" && dur == "" {
		return ""
	}
	if text == "" {
		text = "Completed"
	}
	return text + suffix
}

// firstCount mirrors the Python `_count_list(x, ...) or _count_list(y, ...)`
// idiom used for web_extract: the first non-zero list length wins; when the
// first path produces a zero-length list, the second path is tried and its
// result is returned verbatim (matching Python's `or` returning the right
// operand as-is, so an empty fallback list still reports zero rather than
// silently dropping the summary).
func firstCount(v any, primary, fallback []string) (int, bool) {
	if n, ok := CountList(v, primary...); ok && n != 0 {
		return n, true
	}
	return CountList(v, fallback...)
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
