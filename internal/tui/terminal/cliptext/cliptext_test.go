package cliptext

import "testing"

func TestIsUsableRejectsEmptyAndBinaryTerminalClipboardPayloads(t *testing.T) {
	for _, text := range []string{"", "   \n\t", "hello\x00world", string([]byte{0xff, 0xfe})} {
		if IsUsable(text) {
			t.Fatalf("IsUsable(%q) = true, want false", text)
		}
	}
	if !IsUsable("hello\nclipboard") {
		t.Fatalf("IsUsable(text) = false, want true")
	}
}
