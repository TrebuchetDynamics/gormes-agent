package trace

import "testing"

// Live regression 2026-05-10: an operator on Telegram saw tool-call previews
// rendered as `🔍 web_search: "...et docs websocket WSS Quickstart CLOB"`
// and `📄 web_extract: "...et.com/market-data/websocket/overview"`. Both
// hide the user-meaningful start (the search query's first words and the
// URL's domain) behind a leading "...". Search queries and URLs must keep
// their head; only file-path tool args (read_file/write_file/patch) keep
// the tail because the filename usually matters more than the directory
// chain.

func TestFormatPlain_PreservesUrlAndQueryHead(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "web_search keeps head of query",
			in:   "tool: web_search: polymarket docs websocket WSS Quickstart CLOB",
			want: `🔍 web_search: "polymarket docs websocket WSS Quickst..."`,
		},
		{
			name: "web_extract keeps domain at head of URL",
			in:   "tool: web_extract: https://docs.polymarket.com/market-data/websocket/overview",
			want: `📄 web_extract: "https://docs.polymarket.com/market-da..."`,
		},
		{
			name: "browser_navigate keeps domain at head of URL",
			in:   "tool: browser_navigate: https://docs.polymarket.com/market-data/websocket/overview",
			want: `🌐 browser_navigate: "https://docs.polymarket.com/market-da..."`,
		},
		{
			name: "read_file keeps tail (filename)",
			in:   "tool: read_file: internal/channels/telegram/document_cache.go",
			want: `📖 read_file: "...l/channels/telegram/document_cache.go"`,
		},
		{
			name: "write_file keeps tail (filename)",
			in:   "tool: write_file: internal/channels/telegram/document_cache.go",
			want: `🔧 write_file: "...l/channels/telegram/document_cache.go"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatPlain(tc.in)
			if got != tc.want {
				t.Fatalf("FormatPlain(%q)\n got: %s\nwant: %s", tc.in, got, tc.want)
			}
		})
	}
}
