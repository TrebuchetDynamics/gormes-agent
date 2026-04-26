package tuigateway

import "testing"

// TestNormalizeToolProgressMode_RecognizedValues asserts each of the four
// recognised modes round-trips verbatim.
func TestNormalizeToolProgressMode_RecognizedValues(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"off", "new", "all", "verbose"} {
		mode := mode
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeToolProgressMode(mode); got != mode {
				t.Errorf("NormalizeToolProgressMode(%q) = %q; want %q", mode, got, mode)
			}
		})
	}
}

// TestNormalizeToolProgressMode_BooleanFalseToOff mirrors the upstream
// `if raw is False: return "off"` branch in
// hermes-agent/tui_gateway/server.py:_load_tool_progress_mode.
func TestNormalizeToolProgressMode_BooleanFalseToOff(t *testing.T) {
	t.Parallel()
	if got := NormalizeToolProgressMode(false); got != "off" {
		t.Errorf("NormalizeToolProgressMode(false) = %q; want %q", got, "off")
	}
}

// TestNormalizeToolProgressMode_BooleanTrueToAll mirrors the upstream
// `if raw is True: return "all"` branch.
func TestNormalizeToolProgressMode_BooleanTrueToAll(t *testing.T) {
	t.Parallel()
	if got := NormalizeToolProgressMode(true); got != "all" {
		t.Errorf("NormalizeToolProgressMode(true) = %q; want %q", got, "all")
	}
}

// TestNormalizeToolProgressMode_LowercasesAndTrims accepts mixed-case and
// padded strings the way `str(raw).strip().lower()` does in the upstream
// helper.
func TestNormalizeToolProgressMode_LowercasesAndTrims(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		raw  any
		want string
	}{
		{name: "padded upper ALL", raw: " ALL ", want: "all"},
		{name: "title-case New", raw: "New", want: "new"},
		{name: "title-case Off", raw: "Off", want: "off"},
		{name: "tabbed verbose", raw: "\tVERBOSE\n", want: "verbose"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeToolProgressMode(tc.raw); got != tc.want {
				t.Errorf("NormalizeToolProgressMode(%#v) = %q; want %q", tc.raw, got, tc.want)
			}
		})
	}
}

// TestNormalizeToolProgressMode_UnknownDefaultsToAll covers the catch-all
// fallback in upstream:
//
//	mode = str(raw or "all").strip().lower()
//	return mode if mode in {"off","new","all","verbose"} else "all"
//
// `loud`, the empty string, and Go's nil all collapse to "all".
func TestNormalizeToolProgressMode_UnknownDefaultsToAll(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		raw  any
	}{
		{name: "unrecognised word", raw: "loud"},
		{name: "empty string", raw: ""},
		{name: "whitespace only", raw: "   "},
		{name: "nil interface", raw: nil},
		{name: "integer", raw: 7},
		{name: "float", raw: 1.5},
		{name: "slice", raw: []string{"all"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeToolProgressMode(tc.raw); got != "all" {
				t.Errorf("NormalizeToolProgressMode(%#v) = %q; want %q", tc.raw, got, "all")
			}
		})
	}
}
