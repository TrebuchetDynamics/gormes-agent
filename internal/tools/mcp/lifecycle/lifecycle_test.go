package lifecycle

import "testing"

func TestServerEventShutdownWins(t *testing.T) {
	lifecycle := NewServer()
	lifecycle.SignalReconnect()

	if got := lifecycle.NextEvent(); got != EventReconnect {
		t.Fatalf("NextEvent = %q, want %q", got, EventReconnect)
	}
	if lifecycle.ReconnectPending() {
		t.Fatal("ReconnectPending = true after reconnect event was consumed")
	}

	lifecycle.SignalReconnect()
	lifecycle.SignalShutdown()
	if got := lifecycle.NextEvent(); got != EventShutdown {
		t.Fatalf("NextEvent with both events = %q, want %q", got, EventShutdown)
	}
	if lifecycle.ReconnectPending() {
		t.Fatal("ReconnectPending = true after shutdown won over reconnect")
	}
}
