package slack

import (
	"strings"
	"testing"
)

func TestAugmentInboundTextExtractsQuotesListsAndAttachments(t *testing.T) {
	got, evidence := augmentInboundText(
		"Can you summarize this?",
		sampleRichTextBlocks(),
		sampleAttachmentPreviews(),
	)

	if len(evidence) != 0 {
		t.Fatalf("evidence = %+v, want none", evidence)
	}

	want := strings.Join([]string{
		"Can you summarize this?",
		"> Quoted line",
		"- First bullet",
		"- Second bullet",
		"1. First ordered",
		"2. Second ordered",
		"```go",
		`fmt.Println("hi")`,
		"```",
		"",
		"Link preview: Spec",
		"https://example.com/spec",
		"The latest product spec preview",
		"Notion",
	}, "\n")
	if got != want {
		t.Fatalf("augmented text mismatch\n got:\n%s\nwant:\n%s", got, want)
	}
	if strings.Contains(got, "Thread copy") {
		t.Fatalf("augmented text contains skipped message unfurl: %q", got)
	}
}

func TestAugmentInboundTextSkipsPlainComposerDuplicate(t *testing.T) {
	got, evidence := augmentInboundText("hello world", []SlackBlock{
		{
			"type": "rich_text",
			"elements": []any{
				SlackBlock{
					"type": "rich_text_section",
					"elements": []any{
						SlackBlock{"type": "text", "text": "hello world"},
					},
				},
			},
		},
	}, nil)

	if len(evidence) != 0 {
		t.Fatalf("evidence = %+v, want none", evidence)
	}
	if got != "hello world" {
		t.Fatalf("augmented text = %q, want original text without duplicate", got)
	}
}

func TestAugmentInboundTextRecordsUnavailableEvidence(t *testing.T) {
	got, evidence := augmentInboundText("original text", []SlackBlock{
		{"type": "rich_text", "elements": "not a list"},
	}, []SlackAttachmentPreview{
		{"title": []string{"not", "a", "string"}},
	})

	if got != "original text" {
		t.Fatalf("augmented text = %q, want original text preserved", got)
	}
	if len(evidence) == 0 {
		t.Fatal("evidence is empty, want slack_rich_text_unavailable evidence")
	}
	for _, ev := range evidence {
		if ev.Code != "slack_rich_text_unavailable" {
			t.Fatalf("evidence code = %q, want slack_rich_text_unavailable", ev.Code)
		}
	}
}

func sampleRichTextBlocks() []SlackBlock {
	return []SlackBlock{
		{
			"type": "rich_text",
			"elements": []any{
				SlackBlock{
					"type": "rich_text_quote",
					"elements": []any{
						SlackBlock{
							"type": "rich_text_section",
							"elements": []any{
								SlackBlock{"type": "text", "text": "Quoted line"},
							},
						},
					},
				},
				SlackBlock{
					"type":  "rich_text_list",
					"style": "bullet",
					"elements": []any{
						SlackBlock{
							"type": "rich_text_section",
							"elements": []any{
								SlackBlock{"type": "text", "text": "First bullet"},
							},
						},
						SlackBlock{
							"type": "rich_text_section",
							"elements": []any{
								SlackBlock{"type": "text", "text": "Second bullet"},
							},
						},
					},
				},
				SlackBlock{
					"type":  "rich_text_list",
					"style": "ordered",
					"elements": []any{
						SlackBlock{
							"type": "rich_text_section",
							"elements": []any{
								SlackBlock{"type": "text", "text": "First ordered"},
							},
						},
						SlackBlock{
							"type": "rich_text_section",
							"elements": []any{
								SlackBlock{"type": "text", "text": "Second ordered"},
							},
						},
					},
				},
				SlackBlock{
					"type":     "rich_text_preformatted",
					"language": "go",
					"elements": []any{
						SlackBlock{"type": "text", "text": `fmt.Println("hi")`},
					},
				},
			},
		},
	}
}

func sampleAttachmentPreviews() []SlackAttachmentPreview {
	return []SlackAttachmentPreview{
		{
			"title":      "Spec",
			"title_link": "https://example.com/spec",
			"text":       "The latest product spec preview",
			"footer":     "Notion",
		},
		{
			"is_msg_unfurl": true,
			"title":         "Thread copy",
			"text":          "This should not be appended",
		},
	}
}
