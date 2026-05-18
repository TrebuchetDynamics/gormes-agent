package tui

import "testing"

func TestTUITextInputFastAppendShape(t *testing.T) {
	tests := []struct {
		name             string
		current          string
		cursor           int
		text             string
		columns          int
		currentLineWidth int
		want             bool
	}{
		{name: "ascii append fits", current: "hello", cursor: 5, text: "!", columns: 20, currentLineWidth: 5, want: true},
		{name: "cursor not at end", current: "hello", cursor: 3, text: "!", columns: 20, currentLineWidth: 5},
		{name: "empty current", current: "", cursor: 0, text: "a", columns: 20, currentLineWidth: 0},
		{name: "newline current", current: "hi\nthere", cursor: 8, text: "!", columns: 20, currentLineWidth: 5},
		{name: "wrap boundary", current: "hello", cursor: 5, text: "!", columns: 6, currentLineWidth: 5},
		{name: "ansi text", current: "hello", cursor: 5, text: "\x1b[31m", columns: 20, currentLineWidth: 5},
		{name: "tab text", current: "hello", cursor: 5, text: "\t", columns: 20, currentLineWidth: 5},
		{name: "latin1 text", current: "hello", cursor: 5, text: "é", columns: 20, currentLineWidth: 5},
		{name: "combining mark", current: "hello", cursor: 5, text: "e\u0301", columns: 20, currentLineWidth: 5},
		{name: "cjk text", current: "hello", cursor: 5, text: "界", columns: 20, currentLineWidth: 5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanFastAppendShape(tc.current, tc.cursor, tc.text, tc.columns, tc.currentLineWidth); got != tc.want {
				t.Fatalf("CanFastAppendShape() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTUITextInputFastBackspaceShape(t *testing.T) {
	tests := []struct {
		name    string
		current string
		cursor  int
		columns []int
		want    bool
	}{
		{name: "ascii delete", current: "hello", cursor: 5, columns: []int{20}, want: true},
		{name: "cursor not at end", current: "hello", cursor: 3, columns: []int{20}},
		{name: "empty", current: "", cursor: 0, columns: []int{20}},
		{name: "newline", current: "hi\nthere", cursor: 8, columns: []int{20}},
		{name: "cjk", current: "hi界", cursor: 5, columns: []int{20}},
		{name: "combining mark", current: "hie\u0301", cursor: 5, columns: []int{20}},
		{name: "emoji", current: "hi🙂", cursor: 6, columns: []int{20}},
		{name: "wrap boundary rejected", current: "hello ", cursor: 6, columns: []int{6}},
		{name: "double wrap boundary rejected", current: "abcdefghijkl", cursor: 12, columns: []int{6}},
		{name: "inside wrapped line accepted", current: "abcdefg", cursor: 7, columns: []int{6}, want: true},
		{name: "legacy omitted columns accepted", current: "hello ", cursor: 6, want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanFastBackspaceShape(tc.current, tc.cursor, tc.columns...); got != tc.want {
				t.Fatalf("CanFastBackspaceShape() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTUITextInputFastEchoTerminalSupport(t *testing.T) {
	if SupportsFastEchoTerminal(map[string]string{"TERM_PROGRAM": "Apple_Terminal"}) {
		t.Fatal("SupportsFastEchoTerminal(Apple_Terminal) = true, want false")
	}
	if !SupportsFastEchoTerminal(map[string]string{"TERM_PROGRAM": "vscode"}) {
		t.Fatal("SupportsFastEchoTerminal(vscode) = false, want true")
	}
	if !SupportsFastEchoTerminal(nil) {
		t.Fatal("SupportsFastEchoTerminal(nil) = false, want true")
	}
}
