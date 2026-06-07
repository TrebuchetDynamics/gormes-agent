//go:build !slim

package commandtemplate

import (
	"runtime"
	"strings"
	"testing"
)

func TestRenderQuotesPlaceholdersByShellContext(t *testing.T) {
	got := Render(`copy --in {input_path} --out "{output_path}" --voice "{voice}" --literal {{keep}} --unknown {missing}`, map[string]string{
		"input_path":  "/tmp/input text.txt",
		"output_path": "/tmp/voice clip.ogg",
		"voice":       "bob's $voice",
	})

	if runtime.GOOS == "windows" {
		if !strings.Contains(got, `/tmp/input text.txt`) {
			t.Fatalf("rendered command = %q, want windows unquoted placeholder text", got)
		}
	} else if !strings.Contains(got, `--in '/tmp/input text.txt'`) {
		t.Fatalf("rendered command = %q, want shell-quoted input path", got)
	}
	if !strings.Contains(got, `--out "/tmp/voice clip.ogg"`) {
		t.Fatalf("rendered command = %q, want double-quoted output path preserved", got)
	}
	if !strings.Contains(got, `--voice "bob's \$voice"`) {
		t.Fatalf("rendered command = %q, want double-quote context escaping", got)
	}
	if !strings.Contains(got, `--literal {keep} --unknown {missing}`) {
		t.Fatalf("rendered command = %q, want escaped braces and unknown placeholders preserved", got)
	}
}
