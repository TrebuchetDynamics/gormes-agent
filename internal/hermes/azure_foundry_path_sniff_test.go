package hermes

import "testing"

// TestClassifyAzurePath table-tests the pure URL inspector that mirrors
// hermes_cli/azure_detect.py:_looks_like_anthropic_path. Each subtest
// exercises a discrete classification rule with multiple URL variants;
// every variant in a subtest must classify identically.
func TestClassifyAzurePath(t *testing.T) {
	cases := []struct {
		name string
		urls []string
		want AzureTransport
	}{
		{
			name: "exact_anthropic",
			urls: []string{
				"https://example.azure.com/anthropic",
				"https://example.azure.com/anthropic/",
			},
			want: AzureTransportAnthropic,
		},
		{
			name: "trailing_anthropic",
			urls: []string{
				"https://x.openai.azure.com/openai/deployments/y/anthropic",
				"https://x.openai.azure.com/openai/deployments/y/anthropic/",
			},
			want: AzureTransportAnthropic,
		},
		{
			name: "nested_anthropic",
			urls: []string{
				"https://x/openai/anthropic/v1/messages",
				"https://x/anthropic/v1/messages",
				"https://x/openai/anthropic/",
			},
			want: AzureTransportAnthropic,
		},
		{
			name: "mixed_case",
			urls: []string{
				"https://x/AnthrOPic",
				"https://x/ANTHROPIC",
				"https://x/openai/Anthropic/v1/messages",
				"https://x/foo/AnThRoPiC/bar",
			},
			want: AzureTransportAnthropic,
		},
		{
			name: "bare_host_unknown",
			urls: []string{
				"https://x.openai.azure.com",
				"https://example.com",
				"https://example.com/",
			},
			want: AzureTransportUnknown,
		},
		{
			name: "openai_path_unknown",
			urls: []string{
				"https://x.openai.azure.com/openai/v1/chat/completions",
				"https://x.openai.azure.com/openai/deployments/foo",
				"https://x.openai.azure.com/openai/deployments/foo/chat/completions",
			},
			want: AzureTransportUnknown,
		},
		{
			name: "substring_false_positives",
			urls: []string{
				"https://x/anthropicx",
				"https://x/anthropic-mirror",
				"https://x/foo/anthropicx/bar",
				"https://x/preanthropic",
				"https://x/openai/anthropicv2/messages",
			},
			want: AzureTransportUnknown,
		},
		{
			name: "empty_and_malformed_unknown",
			urls: []string{
				"",
				"::garbage::",
				"http://%zz",
			},
			want: AzureTransportUnknown,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			for _, raw := range tc.urls {
				got := ClassifyAzurePath(raw)
				if got != tc.want {
					t.Errorf("ClassifyAzurePath(%q) = %q, want %q", raw, got, tc.want)
				}
			}
		})
	}
}
