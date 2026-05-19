package main

import (
	"testing"

	"github.com/TrebuchetDynamics/goncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
)

func TestNewChannelGonchoServiceInstallsHermesDialecticCaller(t *testing.T) {
	store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(t.Context())

	svc := newChannelGonchoService(store.DB(), goncho.Config{WorkspaceID: "default", ObserverPeerID: "gormes"}, nil, hermes.NewMockClient(), "gpt-test")
	if svc.DialecticCaller() == nil {
		t.Fatal("DialecticCaller = nil, want native provider-backed caller installed")
	}
}

func TestNewChannelGonchoServiceLeavesFallbackWhenClientMissing(t *testing.T) {
	store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(t.Context())

	svc := newChannelGonchoService(store.DB(), goncho.Config{WorkspaceID: "default", ObserverPeerID: "gormes"}, nil, nil, "gpt-test")
	if svc.DialecticCaller() != nil {
		t.Fatal("DialecticCaller installed without client; deterministic fallback should remain available")
	}
}
