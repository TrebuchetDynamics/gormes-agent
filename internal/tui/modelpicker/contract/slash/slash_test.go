package slash

import "testing"

func TestSlashArgument(t *testing.T) {
	if got := Argument("/model openai/gpt-4.1"); got != "openai/gpt-4.1" {
		t.Fatalf("SlashArgument = %q, want openai/gpt-4.1", got)
	}
	if got := Argument("/model"); got != "" {
		t.Fatalf("SlashArgument without arg = %q, want empty", got)
	}
}
