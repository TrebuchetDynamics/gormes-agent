package telegram

import "testing"

func TestTelegramCaptionMerge(t *testing.T) {
	t.Run("drops exact duplicates", func(t *testing.T) {
		got := telegramMergeCaption("Revenue", "Revenue")
		if got != "Revenue" {
			t.Fatalf("caption = %q, want exact duplicate to be dropped", got)
		}
	})

	t.Run("preserves substring captions", func(t *testing.T) {
		got := telegramMergeCaption("Meeting agenda", "Meeting")
		if got != "Meeting agenda\n\nMeeting" {
			t.Fatalf("caption = %q, want shorter substring preserved", got)
		}
	})

	t.Run("preserves longer captions containing existing text", func(t *testing.T) {
		got := telegramMergeCaption("Revenue", "Revenue and Profit")
		if got != "Revenue\n\nRevenue and Profit" {
			t.Fatalf("caption = %q, want longer substring preserved", got)
		}
	})

	t.Run("keeps three unique captions", func(t *testing.T) {
		got := telegramMergeCaption("", "A")
		got = telegramMergeCaption(got, "B")
		got = telegramMergeCaption(got, "C")
		if got != "A\n\nB\n\nC" {
			t.Fatalf("caption = %q, want all unique captions", got)
		}
	})
}
