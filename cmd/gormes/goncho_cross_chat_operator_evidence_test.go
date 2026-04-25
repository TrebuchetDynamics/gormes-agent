package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

func TestGonchoDoctorContextDryRunPrintsCrossChatOperatorEvidence(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dataHome, "config"))

	store, err := memory.OpenSqlite(config.MemoryDBPath(), 8, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(context.Background())

	smap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		t.Fatalf("OpenBolt: %v", err)
	}

	ctx := context.Background()
	for _, meta := range []session.Metadata{
		{SessionID: "sess-current", Source: "discord", ChatID: "chan-9", UserID: "user-juan"},
		{SessionID: "sess-telegram", Source: "telegram", ChatID: "42", UserID: "user-juan"},
	} {
		if err := smap.PutMetadata(ctx, meta); err != nil {
			t.Fatalf("PutMetadata(%s): %v", meta.SessionID, err)
		}
	}
	if err := smap.Close(); err != nil {
		t.Fatalf("Close session map: %v", err)
	}
	now := time.Now().Unix()
	if _, err := store.DB().ExecContext(ctx,
		`INSERT INTO turns(session_id, role, content, ts_unix, chat_id)
		 VALUES
		 ('sess-current', 'user', 'doctor dry-run current Discord note.', ?, 'discord:chan-9'),
		 ('sess-telegram', 'user', 'doctor dry-run widened Telegram note.', ?, 'telegram:42')`,
		now-20, now-10,
	); err != nil {
		t.Fatalf("seed turns: %v", err)
	}

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"goncho", "doctor",
		"--peer", "user-juan",
		"--session", "discord:chan-9",
		"--scope", "user",
		"--sources", "telegram",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v\nstderr=%s\nstdout=%s", err, stderr.String(), stdout.String())
	}

	out := stdout.String()
	for _, want := range []string{
		"Context dry-run",
		"scope_evidence: decision=allowed user_id=user-juan",
		"source_allowlist=telegram",
		"sessions_considered=1",
		"widened_sessions_considered=1",
		"search_results:",
		"source=turn origin_source=telegram session_key=sess-telegram",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q:\n%s", want, out)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}
