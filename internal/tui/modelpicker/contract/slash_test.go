package contract

import "testing"

func TestSlashArgument(t *testing.T) {
	if got := SlashArgument("/model openai/gpt-4.1"); got != "openai/gpt-4.1" {
		t.Fatalf("SlashArgument = %q, want openai/gpt-4.1", got)
	}
	if got := SlashArgument("/model"); got != "" {
		t.Fatalf("SlashArgument without arg = %q, want empty", got)
	}
}
