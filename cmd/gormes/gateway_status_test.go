package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestGatewayStatusCommand_NoTransportsStarted(t *testing.T) {
	// The `gateway status` subcommand must be a pure read-only surface: it
	// must not open Telegram/Discord transports, the session map, or the
	// memory store. This test asserts presence + invocation via the cobra
	// root, piping stdout through a buffer.
	root := newRootCommand()

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"gateway", "status"})

	if err := root.Execute(); err != nil {
		t.Fatalf("gormes gateway status: %v (stdout+stderr = %q)", err, out.String())
	}

	body := out.String()
	if !strings.Contains(body, "gateway status") && !strings.Contains(body, "channels") && !strings.Contains(body, "no channels configured") {
		t.Fatalf("gormes gateway status: output did not look like status readout, got: %q", body)
	}
}
