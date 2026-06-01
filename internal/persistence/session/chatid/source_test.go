package chatid

import "testing"

func TestSourceFromTranscriptChatID(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty legacy cli", in: "", want: "cli"},
		{name: "unqualified legacy cli", in: "42", want: "cli"},
		{name: "qualified lower", in: "telegram:42", want: "telegram"},
		{name: "qualified trims lowercases", in: " Discord:channel ", want: "discord"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SourceFromTranscriptChatID(tt.in); got != tt.want {
				t.Fatalf("SourceFromTranscriptChatID(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
