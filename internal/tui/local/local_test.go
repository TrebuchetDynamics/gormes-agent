package local

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

type sessionSeed struct {
	id      string
	title   string
	role    string
	content string
	ts      int64
	chatID  string
	turnKey string
}

func TestNewSessionBranchFuncWithIDForksTranscriptAndResumesKernelSession(t *testing.T) {
	seedSessionDB(t, []sessionSeed{
		{id: "sess-parent", role: "user", content: "persisted question", ts: 100},
		{id: "sess-parent", role: "assistant", content: "persisted answer", ts: 101},
	})
	boltMap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		t.Fatalf("OpenBolt: %v", err)
	}
	defer boltMap.Close()

	var resumedSession string
	var resumedHistory []llm.Message
	branch := NewSessionBranchFuncWithID(context.Background(), boltMap, func(sessionID string, history []llm.Message) error {
		resumedSession = sessionID
		resumedHistory = append([]llm.Message(nil), history...)
		return nil
	}, func() string { return "sess-child" })

	result, err := branch(context.Background(), tui.BranchRequest{
		ParentSessionID: "sess-parent",
		Title:           "branch title",
		History: []llm.Message{
			{Role: "user", Content: "visible question"},
			{Role: "assistant", Content: "visible answer"},
		},
	})
	if err != nil {
		t.Fatalf("SessionBranch: %v", err)
	}
	if result.SessionID != "sess-child" || result.ParentSessionID != "sess-parent" || result.Title != "branch title" || result.TranscriptCopied != 2 {
		t.Fatalf("BranchResult = %+v, want child/parent/title/copied evidence", result)
	}
	if resumedSession != "sess-child" {
		t.Fatalf("resumed session = %q, want sess-child", resumedSession)
	}
	if len(resumedHistory) != 2 || resumedHistory[0].Content != "visible question" || resumedHistory[1].Content != "visible answer" {
		t.Fatalf("resumed history = %+v, want visible frame history", resumedHistory)
	}
	assertTurnCount(t, "sess-child", 2)

	meta, ok, err := boltMap.GetMetadata(context.Background(), "sess-child")
	if err != nil {
		t.Fatalf("GetMetadata: %v", err)
	}
	if !ok || meta.ParentSessionID != "sess-parent" || meta.LineageKind != session.LineageKindFork {
		t.Fatalf("metadata = %+v ok=%v, want fork child metadata", meta, ok)
	}
}

func TestNewSessionBranchFuncWithIDFallsBackToCopiedTranscriptWhenVisibleHistoryEmpty(t *testing.T) {
	seedSessionDB(t, []sessionSeed{
		{id: "sess-parent", role: "user", content: "persisted question", ts: 100},
		{id: "sess-parent", role: "assistant", content: "persisted answer", ts: 101},
	})
	boltMap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		t.Fatalf("OpenBolt: %v", err)
	}
	defer boltMap.Close()

	var resumedHistory []llm.Message
	branch := NewSessionBranchFuncWithID(context.Background(), boltMap, func(_ string, history []llm.Message) error {
		resumedHistory = append([]llm.Message(nil), history...)
		return nil
	}, func() string { return "sess-child" })

	if _, err := branch(context.Background(), tui.BranchRequest{ParentSessionID: "sess-parent"}); err != nil {
		t.Fatalf("SessionBranch: %v", err)
	}
	if len(resumedHistory) != 2 || resumedHistory[0].Content != "persisted question" || resumedHistory[1].Content != "persisted answer" {
		t.Fatalf("resumed history = %+v, want copied persisted transcript", resumedHistory)
	}
}

func TestNewSessionDirectoryFuncListsMirrorSource(t *testing.T) {
	seedSessionDB(t, []sessionSeed{
		{id: "sess-alpha", title: "Alpha Work", role: "user", content: "preview alpha", ts: 100},
		{id: "sess-beta", title: "Beta Work", role: "user", content: "preview beta", ts: 200, turnKey: "beta-user"},
		{id: "sess-beta", title: "Beta Work", role: "user", content: "preview beta", ts: 200, chatID: "user"},
	})
	writeSessionMirrorIndex(t, "sess-beta", "telegram")

	entries, err := NewSessionDirectoryFunc(context.Background())(1)
	if err != nil {
		t.Fatalf("SessionDirectory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("SessionDirectory returned %d entries, want limit 1: %+v", len(entries), entries)
	}
	if entries[0].ID != "sess-beta" || entries[0].Title != "Beta Work" || entries[0].Preview != "preview beta" || entries[0].Source != "telegram" || entries[0].MessageCount != 1 {
		t.Fatalf("SessionDirectory entry = %+v, want newest deduped Telegram Beta Work entry", entries[0])
	}
}

func TestNewSessionResumeFuncResolvesPrefixAndLoadsHistory(t *testing.T) {
	seedSessionDB(t, []sessionSeed{
		{id: "sess-alpha", role: "user", content: "alpha question", ts: 100},
		{id: "sess-alpha", role: "assistant", content: "alpha answer", ts: 101},
	})

	var resumedSession string
	resume := NewSessionResumeFunc(context.Background(), func(sessionID string, _ []llm.Message) error {
		resumedSession = sessionID
		return nil
	})
	result, err := resume(context.Background(), "sess-al")
	if err != nil {
		t.Fatalf("SessionResume: %v", err)
	}
	if result.SessionID != "sess-alpha" || resumedSession != "sess-alpha" {
		t.Fatalf("SessionResume session = result:%q callback:%q, want sess-alpha", result.SessionID, resumedSession)
	}
	if len(result.History) != 2 || result.History[0].Content != "alpha question" || result.History[1].Content != "alpha answer" {
		t.Fatalf("SessionResume History = %+v, want replayed alpha transcript", result.History)
	}
}

func TestSessionTreeAdaptersReadLabelAndRestore(t *testing.T) {
	seedSessionDB(t, []sessionSeed{
		{id: "sess-root", title: "Root Plan", role: "user", content: "root prompt", ts: 100},
		{id: "sess-root", role: "assistant", content: "root answer", ts: 101},
		{id: "sess-fork", role: "user", content: "fork prompt", ts: 120},
	})
	boltMap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		t.Fatalf("OpenBolt: %v", err)
	}
	defer boltMap.Close()
	ctx := context.Background()
	if err := boltMap.PutMetadata(ctx, session.Metadata{SessionID: "sess-root", Title: "Root Plan", LineageKind: session.LineageKindPrimary, CreatedAt: 100, UpdatedAt: 110, Labels: []string{"pinned"}}); err != nil {
		t.Fatalf("put root metadata: %v", err)
	}
	if err := boltMap.PutMetadata(ctx, session.Metadata{SessionID: "sess-fork", Title: "Branch", ParentSessionID: "sess-root", LineageKind: session.LineageKindFork, CreatedAt: 120, UpdatedAt: 130}); err != nil {
		t.Fatalf("put fork metadata: %v", err)
	}

	treeResult, err := NewSessionTreeFunc(context.Background(), boltMap)(context.Background(), tui.SessionTreeRequest{Filter: "all-equivalent", ActiveSessionID: "sess-fork"})
	if err != nil {
		t.Fatalf("SessionTree: %v", err)
	}
	labelResult, err := NewSessionTreeLabelFunc(context.Background(), boltMap)(context.Background(), tui.SessionTreeLabelRequest{SessionID: "sess-root", Action: "set", Label: "review"})
	if err != nil {
		t.Fatalf("SessionTreeLabel: %v", err)
	}
	restoreResult, err := NewSessionTreeRestoreFunc(context.Background())(context.Background(), tui.SessionTreeRestoreRequest{SessionID: "sess-root", MessageID: 1})
	if err != nil {
		t.Fatalf("SessionTreeRestore: %v", err)
	}

	if gotIDs := treeIDs(treeResult.Entries); !reflect.DeepEqual(gotIDs, []string{"sess-root", "sess-fork"}) {
		t.Fatalf("tree IDs = %v, want root/fork", gotIDs)
	}
	if !treeResult.Entries[1].Active || treeResult.Entries[1].ParentID != "sess-root" || treeResult.Entries[1].LineageKind != session.LineageKindFork {
		t.Fatalf("fork entry = %+v, want active fork child", treeResult.Entries[1])
	}
	if len(treeResult.Entries[0].Labels) == 0 || treeResult.Entries[0].Labels[0] != "pinned" {
		t.Fatalf("root labels = %v, want pinned", treeResult.Entries[0].Labels)
	}
	if roles := treeMessageRoles(treeResult.Entries[0].Messages); !reflect.DeepEqual(roles, []string{"user", "assistant"}) {
		t.Fatalf("root message roles = %v, want all-equivalent user+assistant", roles)
	}
	if labelResult.SessionID != "sess-root" || !reflect.DeepEqual(labelResult.Labels, []string{"pinned", "review"}) {
		t.Fatalf("label result = %+v, want pinned+review", labelResult)
	}
	if !restoreResult.Editable || restoreResult.Text != "root prompt" || restoreResult.Evidence != "" {
		t.Fatalf("restore result = %+v, want editable root prompt", restoreResult)
	}
}

func TestSessionTitleFuncPersistsManualTitle(t *testing.T) {
	setupLocalTestHome(t)
	boltMap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		t.Fatalf("OpenBolt: %v", err)
	}
	defer boltMap.Close()

	titleFn := NewSessionTitleFunc(context.Background(), boltMap)
	setRes, err := titleFn("sess-tui-title", "Operator Title")
	if err != nil {
		t.Fatalf("SessionTitle set: %v", err)
	}
	getRes, err := titleFn("sess-tui-title", "")
	if err != nil {
		t.Fatalf("SessionTitle get: %v", err)
	}
	if setRes.Title != "Operator Title" || getRes.Title != "Operator Title" {
		t.Fatalf("SessionTitle set/get = %+v/%+v, want Operator Title", setRes, getRes)
	}
	meta, ok, err := boltMap.GetMetadata(context.Background(), "sess-tui-title")
	if err != nil {
		t.Fatalf("GetMetadata: %v", err)
	}
	if !ok || meta.Title != "Operator Title" || !meta.TitleManuallySet {
		t.Fatalf("metadata = %+v ok=%v, want manual Operator Title", meta, ok)
	}
}

func TestConfigureToolsReportsUnknownMissingAndDisablePersistence(t *testing.T) {
	setupLocalTestHome(t)
	writeSetupToolsFixtureConfig(t, `
platform_toolsets = { cli = ["terminal"] }
`)
	configure := NewToolsConfigureFunc()

	result, err := configure(tui.ToolsConfigureRequest{Action: "enable", Names: []string{"web", "github:create_issue", "not-a-toolset"}, SessionID: "sess-tools"})
	if err != nil {
		t.Fatalf("enable ToolsConfigure: %v", err)
	}
	if !containsString(result.Changed, "web") || !containsString(result.MissingServers, "github") || !containsString(result.Unknown, "not-a-toolset") || !result.Reset {
		t.Fatalf("enable result = %+v, want changed web, missing github, unknown not-a-toolset, reset", result)
	}
	got := readCLIPlatformToolsets(t)
	if !containsString(got, "web") || containsString(got, "not_a_toolset") || containsString(got, "github") {
		t.Fatalf("persisted after enable = %v, want web only from accepted additions", got)
	}

	result, err = configure(tui.ToolsConfigureRequest{Action: "disable", Names: []string{"terminal"}, SessionID: "sess-tools"})
	if err != nil {
		t.Fatalf("disable ToolsConfigure: %v", err)
	}
	if !containsString(result.Changed, "terminal") || !result.Reset {
		t.Fatalf("disable result = %+v, want changed terminal with reset", result)
	}
	got = readCLIPlatformToolsets(t)
	if containsString(got, "terminal") || !containsString(got, "web") {
		t.Fatalf("persisted after disable = %v, want terminal removed and web preserved", got)
	}
}

func TestSkinConfigFuncGetsSetsAndRejectsUnknown(t *testing.T) {
	setupLocalTestHome(t)
	if err := config.WriteTOMLValue(config.ConfigPath(), "tui.theme", "ares"); err != nil {
		t.Fatalf("write tui.theme: %v", err)
	}
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	configure := NewSkinConfigFunc(cfg)
	status, err := configure(tui.SkinConfigRequest{SessionID: "sess-skin"})
	if err != nil {
		t.Fatalf("SkinConfig status: %v", err)
	}
	changed, err := configure(tui.SkinConfigRequest{Name: "mono", SessionID: "sess-skin"})
	if err != nil {
		t.Fatalf("SkinConfig set: %v", err)
	}
	if _, err := configure(tui.SkinConfigRequest{Name: "zeus", SessionID: "sess-skin"}); err == nil {
		t.Fatal("SkinConfig invalid skin error = nil, want unknown skin")
	}
	if status.Name != "ares" || changed.Name != "mono" {
		t.Fatalf("SkinConfig status/set = %+v/%+v, want ares/mono", status, changed)
	}
	reloaded, err := config.Load(nil)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if reloaded.TUI.Theme != "mono" {
		t.Fatalf("persisted tui.theme = %q, want mono", reloaded.TUI.Theme)
	}
}

func TestAccountUsageFuncReportsUnsupportedProvider(t *testing.T) {
	accountUsage := NewAccountUsageFunc(config.Config{Hermes: config.HermesCfg{Provider: "custom-provider"}})
	snapshot, err := accountUsage(context.Background())
	if err != nil {
		t.Fatalf("AccountUsage: %v", err)
	}
	if snapshot.Provider != "custom-provider" {
		t.Fatalf("AccountUsage provider = %q, want custom-provider", snapshot.Provider)
	}
	if snapshot.Unavailable == nil || snapshot.Unavailable.Reason != llm.AccountUsageReasonUnsupportedProvider {
		t.Fatalf("AccountUsage unavailable = %+v, want unsupported provider evidence", snapshot.Unavailable)
	}
}

func TestVoiceRequirementsDetailsShowsConfiguredSTT(t *testing.T) {
	details := VoiceRequirementsDetails(config.Config{
		Runtime: config.RuntimeCfg{TTSProvider: "edge"},
		STT: config.STTCfg{
			Provider: "local",
			Local:    config.STTLocalCfg{Model: "tiny.en", Language: "en"},
		},
	})
	for _, want := range []string{"STT: configured (local)", "model tiny.en", "language en", "TTS: configured (edge)"} {
		if !containsStringInText(details, want) {
			t.Fatalf("details missing %q:\n%s", want, details)
		}
	}
	if containsStringInText(details, "STT: not configured") {
		t.Fatalf("details should not report configured STT as absent:\n%s", details)
	}
}

func setupLocalTestHome(t *testing.T) string {
	t.Helper()
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dataHome, "config"))
	home := filepath.Join(dataHome, "gormes")
	t.Setenv("GORMES_HOME", home)
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatalf("mkdir GORMES_HOME: %v", err)
	}
	return home
}

func seedSessionDB(t *testing.T, seeds []sessionSeed) {
	t.Helper()
	setupLocalTestHome(t)
	store, err := memory.OpenSqlite(config.MemoryDBPath(), 8, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(context.Background())

	for _, seed := range seeds {
		meta := ""
		if seed.title != "" {
			meta = `{"title":"` + seed.title + `"}`
		}
		if _, err := store.DB().Exec(
			`INSERT INTO turns(session_id, role, content, ts_unix, chat_id, meta_json, turn_key) VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''))`,
			seed.id, seed.role, seed.content, seed.ts, seed.chatID, meta, seed.turnKey,
		); err != nil {
			t.Fatalf("seed session %s: %v", seed.id, err)
		}
	}
}

func assertTurnCount(t *testing.T, sessionID string, want int) {
	t.Helper()
	store, err := memory.OpenSqlite(config.MemoryDBPath(), 8, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(context.Background())
	var got int
	if err := store.DB().QueryRow(`SELECT COUNT(*) FROM turns WHERE session_id = ?`, sessionID).Scan(&got); err != nil {
		t.Fatalf("count turns for %s: %v", sessionID, err)
	}
	if got != want {
		t.Fatalf("turn count for %s = %d, want %d", sessionID, got, want)
	}
}

func writeSessionMirrorIndex(t *testing.T, sessionID, source string) {
	t.Helper()
	path := config.SessionIndexMirrorPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create session mirror dir: %v", err)
	}
	body := "# Auto-generated session index\nsessions:\n  " + source + ":42: " + sessionID + "\nlineage:\n  " + sessionID + ":\n    lineage_kind: primary\n    lineage_status: ok\n    source: " + source + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write session mirror: %v", err)
	}
}

func writeSetupToolsFixtureConfig(t *testing.T, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(config.ConfigPath()), 0o700); err != nil {
		t.Fatalf("mkdir config home: %v", err)
	}
	if err := os.WriteFile(config.ConfigPath(), []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func readCLIPlatformToolsets(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var doc map[string]any
	if err := toml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse config: %v\n%s", err, string(data))
	}
	platformToolsets, _ := doc["platform_toolsets"].(map[string]any)
	raw, _ := platformToolsets["cli"].([]any)
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		out = append(out, value.(string))
	}
	return out
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsStringInText(text, want string) bool {
	return strings.Contains(text, want)
}

func treeIDs(entries []tui.SessionTreeEntry) []string {
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	return ids
}

func treeMessageRoles(messages []tui.SessionTreeMessage) []string {
	roles := make([]string, 0, len(messages))
	for _, msg := range messages {
		roles = append(roles, msg.Role)
	}
	return roles
}
