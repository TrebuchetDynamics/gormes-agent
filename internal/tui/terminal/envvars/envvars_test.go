package envvars

import "testing"

func TestValueHasAndRemoteUseTerminalEnvSemantics(t *testing.T) {
	env := map[string]string{
		"TERM_PROGRAM":   "vscode",
		"SSH_CONNECTION": "local remote",
	}

	if got := Value(env, "TERM_PROGRAM"); got != "vscode" {
		t.Fatalf("Value() = %q; want vscode", got)
	}
	if !Has(env, "TERM_PROGRAM") {
		t.Fatal("Has() = false; want true")
	}
	if !IsRemote(env) {
		t.Fatal("IsRemote() = false; want true")
	}
}

func TestIsRemoteAcceptsSSHTTY(t *testing.T) {
	if !IsRemote(map[string]string{"SSH_TTY": "/dev/pts/1"}) {
		t.Fatal("IsRemote() = false; want true")
	}
	if IsRemote(nil) {
		t.Fatal("IsRemote(nil) = true; want false")
	}
}
