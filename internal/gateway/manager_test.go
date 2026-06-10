package gateway

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

func TestManager_RegisterChannel(t *testing.T) {
	m := NewManager(ManagerConfig{}, nil, slog.Default())

	tg := newFakeChannel("telegram")
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register telegram: %v", err)
	}

	dc := newFakeChannel("discord")
	if err := m.Register(dc); err != nil {
		t.Fatalf("Register discord: %v", err)
	}

	if got := m.ChannelCount(); got != 2 {
		t.Errorf("ChannelCount() = %d, want 2", got)
	}
}

func TestManager_RegisterDuplicateName(t *testing.T) {
	m := NewManager(ManagerConfig{}, nil, slog.Default())

	if err := m.Register(newFakeChannel("telegram")); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	err := m.Register(newFakeChannel("telegram"))
	if err == nil {
		t.Fatal("expected duplicate-name error, got nil")
	}
}

func TestManager_RegisterEmptyName(t *testing.T) {
	m := NewManager(ManagerConfig{}, nil, slog.Default())
	if err := m.Register(newFakeChannel("")); err == nil {
		t.Fatal("expected empty-name error, got nil")
	}
}

func TestManager_ConfigMapsAreSnapshotted(t *testing.T) {
	allowedChats := map[string]string{"telegram": "42"}
	allowedUsers := map[string]map[string]bool{"telegram": {"u1": true}}
	m := NewManager(ManagerConfig{
		AllowedChats: allowedChats,
		AllowedUsers: allowedUsers,
	}, nil, slog.Default())

	allowedChats["telegram"] = "99"
	allowedUsers["telegram"]["u2"] = true
	allowedUsers["slack"] = map[string]bool{"u3": true}

	if !m.allowed(InboundEvent{Platform: "telegram", ChatID: "42"}) {
		t.Fatal("manager lost original allowed chat after caller mutated config map")
	}
	if m.allowed(InboundEvent{Platform: "telegram", ChatID: "99"}) {
		t.Fatal("manager observed caller mutation to allowed chat map")
	}
	if !m.allowed(InboundEvent{Platform: "telegram", ChatID: "unlisted", UserID: "u1"}) {
		t.Fatal("manager lost original allowed user after caller mutated nested map")
	}
	if m.allowed(InboundEvent{Platform: "telegram", ChatID: "unlisted", UserID: "u2"}) {
		t.Fatal("manager observed caller mutation to nested allowed users map")
	}
	if m.allowed(InboundEvent{Platform: "slack", ChatID: "unlisted", UserID: "u3"}) {
		t.Fatal("manager observed caller-added platform allowed users map")
	}
}

func TestManager_AllowedUsesPlatformUserAllowlist(t *testing.T) {
	m := NewManager(ManagerConfig{
		AllowedUsers: map[string]map[string]bool{
			"telegram": {"6586915095": true},
		},
	}, nil, slog.Default())

	if !m.allowed(InboundEvent{Platform: "telegram", ChatID: "-10042", UserID: "6586915095"}) {
		t.Fatal("allowed user was blocked")
	}
	if m.allowed(InboundEvent{Platform: "telegram", ChatID: "-10042", UserID: "111"}) {
		t.Fatal("unlisted user was allowed")
	}
}

func TestManager_GuestMentionBypassOnlyAdmitsTelegramDirectMention(t *testing.T) {
	m := NewManager(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "-200"},
	}, nil, slog.Default())

	if !m.allowed(InboundEvent{Platform: "telegram", ChatID: "-201", AllowlistBypassReason: AllowlistBypassTelegramGuestMention}) {
		t.Fatal("telegram guest mention bypass was blocked")
	}
	if m.allowed(InboundEvent{Platform: "telegram", ChatID: "-201"}) {
		t.Fatal("non-allowlisted telegram chat without guest mention bypass was allowed")
	}
	if m.allowed(InboundEvent{Platform: "slack", ChatID: "C-201", AllowlistBypassReason: AllowlistBypassTelegramGuestMention}) {
		t.Fatal("non-telegram platform used telegram guest mention bypass")
	}
}

type fakeKernel struct {
	mu        sync.Mutex
	submits   []kernel.PlatformEvent
	resets    int
	submitErr error
	resetErr  error
}

func (f *fakeKernel) Submit(e kernel.PlatformEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.submitErr != nil {
		return f.submitErr
	}
	f.submits = append(f.submits, e)
	return nil
}

func (f *fakeKernel) ResetSession() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.resetErr != nil {
		return f.resetErr
	}
	f.resets++
	return nil
}

func (f *fakeKernel) Render() <-chan kernel.RenderFrame {
	return nil
}

func (f *fakeKernel) submitsSnapshot() []kernel.PlatformEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	return cloneSlice(f.submits)
}

func stopManagerTestRun(t *testing.T, cancel context.CancelFunc, done <-chan struct{}) {
	t.Helper()
	cancel()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("manager test run did not stop")
	}
}

func TestManager_Inbound_AllowedChat_Submit(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = m.Run(ctx)
	}()
	defer stopManagerTestRun(t, cancel, done)

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "m",
		Kind: EventSubmit, Text: "hello",
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})
	got := fk.submitsSnapshot()[0]
	if got.Kind != kernel.PlatformEventSubmit || got.Text != "hello" {
		t.Errorf("kernel submit = %+v, want %+v", got, kernel.PlatformEvent{Kind: kernel.PlatformEventSubmit, Text: "hello"})
	}
}

func TestManager_ReloadCommandAppliesReloadableAllowlistWithoutRestart(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}
	store := NewRuntimeStatusStore(t.TempDir() + "/gateway_state.json")
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:  map[string]string{"telegram": "42"},
		RuntimeStatus: store,
		ReloadConfig: func(context.Context) (ManagerConfig, error) {
			return ManagerConfig{
				AllowedChats: map[string]string{"telegram": "99"},
			}, nil
		},
	}, fk, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = m.Run(ctx)
	}()
	defer stopManagerTestRun(t, cancel, done)

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "reload-1",
		Kind: EventReload,
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		sent := tg.sentSnapshot()
		return len(sent) == 1 && strings.Contains(sent[0].Text, "Config reloaded")
	})
	status, err := store.ReadRuntimeStatus(context.Background())
	if err != nil {
		t.Fatalf("ReadRuntimeStatus: %v", err)
	}
	if status.ConfigReload.Status != RuntimeConfigReloadApplied {
		t.Fatalf("config reload status = %+v, want applied", status.ConfigReload)
	}

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "99", UserID: "u", MsgID: "m-99",
		Kind: EventSubmit, Text: "after reload",
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})
	got := fk.submitsSnapshot()[0]
	if got.Text != "after reload" {
		t.Fatalf("kernel submit text = %q, want reloaded chat submit", got.Text)
	}
}

func TestManager_ReloadSnapshotsWhitelistConfig(t *testing.T) {
	reloadedWhitelists := map[string]WhitelistConfig{"telegram": {Enabled: true, IDs: []string{"42"}}}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "old"},
		ReloadConfig: func(context.Context) (ManagerConfig, error) {
			return ManagerConfig{
				AllowedUsers:          map[string]map[string]bool{"telegram": {"*": true}},
				AllowedChatWhitelists: reloadedWhitelists,
			}, nil
		},
	}, &fakeKernel{}, slog.Default())

	if err := m.Reload(context.Background()); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	reloadedWhitelists["telegram"] = WhitelistConfig{Enabled: true, IDs: []string{"99"}}

	if !m.allowed(InboundEvent{Platform: "telegram", ChatID: "42"}) {
		t.Fatal("manager lost reloaded whitelist after caller mutated returned map")
	}
	if m.allowed(InboundEvent{Platform: "telegram", ChatID: "99"}) {
		t.Fatal("manager observed caller mutation to reloaded whitelist map")
	}
}

func TestManager_ReloadCommandKeepsLastGoodConfigOnFailure(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}
	store := NewRuntimeStatusStore(t.TempDir() + "/gateway_state.json")
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:  map[string]string{"telegram": "42"},
		RuntimeStatus: store,
		ReloadConfig: func(context.Context) (ManagerConfig, error) {
			return ManagerConfig{}, errors.New("parse config.toml: api_key=plain-secret-token")
		},
	}, fk, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = m.Run(ctx)
	}()
	defer stopManagerTestRun(t, cancel, done)

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "reload-1",
		Kind: EventReload,
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		sent := tg.sentSnapshot()
		return len(sent) == 1 && strings.Contains(sent[0].Text, "Config reload failed")
	})
	if strings.Contains(tg.sentSnapshot()[0].Text, "plain-secret-token") {
		t.Fatalf("reload failure leaked secret: %q", tg.sentSnapshot()[0].Text)
	}

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "m-42",
		Kind: EventSubmit, Text: "last good",
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})
	if got := fk.submitsSnapshot()[0].Text; got != "last good" {
		t.Fatalf("submit text = %q, want last-good config to remain active", got)
	}

	status, err := store.ReadRuntimeStatus(context.Background())
	if err != nil {
		t.Fatalf("ReadRuntimeStatus: %v", err)
	}
	if status.ConfigReload.Status != RuntimeConfigReloadFailed {
		t.Fatalf("config reload status = %+v, want failed", status.ConfigReload)
	}
	if strings.Contains(status.ConfigReload.Error, "plain-secret-token") {
		t.Fatalf("runtime status leaked reload secret: %+v", status.ConfigReload)
	}
}

func TestManager_Inbound_VerboseDisabledSendsHermesGateGuidance(t *testing.T) {
	tg := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, &fakeKernel{}, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "m",
		Kind: EventVerbose, Text: "/verbose",
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		return len(tg.sentSnapshot()) == 1
	})
	got := tg.sentSnapshot()[0].Text
	assertContainsAll(t, got,
		"The `/verbose` command is not enabled for messaging platforms.",
		"Gormes `config.toml`",
		"[display]",
		"tool_progress_command = true",
	)
	if strings.Contains(got, "unavailable in this build") {
		t.Fatalf("/verbose disabled response = %q, want Gormes config guidance instead of unavailable text", got)
	}
}

func TestManager_Inbound_VerboseCyclesAndPersistsPerPlatform(t *testing.T) {
	tg := newFakeChannel("telegram")
	var persistedPlatform, persistedMode string
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:               map[string]string{"telegram": "42"},
		ToolProgressCommandEnabled: true,
		ToolProgressMode:           "all",
		ToolProgressModes:          map[string]string{"telegram": "off"},
		PersistToolProgressMode: func(platform, mode string) error {
			persistedPlatform = platform
			persistedMode = mode
			return nil
		},
	}, &fakeKernel{}, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "m",
		Kind: EventVerbose, Text: "/verbose",
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		return len(tg.sentSnapshot()) == 1
	})
	got := tg.sentSnapshot()[0].Text
	assertContainsAll(t, got,
		"⚙️ Tool progress: **NEW**",
		"saved for **telegram**",
	)
	if strings.Contains(got, "unavailable in this build") {
		t.Fatalf("/verbose enabled response = %q, want cycle result", got)
	}
	if persistedPlatform != "telegram" || persistedMode != "new" {
		t.Fatalf("persisted = (%q, %q), want (telegram, new)", persistedPlatform, persistedMode)
	}
	if got := m.toolProgressMode("telegram"); got != "new" {
		t.Fatalf("toolProgressMode(telegram) = %q, want runtime override new", got)
	}
}

func TestManager_Inbound_VerbosePersistErrorSanitizesOperatorReply(t *testing.T) {
	tg := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:               map[string]string{"telegram": "42"},
		ToolProgressCommandEnabled: true,
		ToolProgressModes:          map[string]string{"telegram": "off"},
		PersistToolProgressMode: func(string, string) error {
			return errors.New("save failed\n**Injected:** bearer plain-secret")
		},
	}, &fakeKernel{}, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "m", Kind: EventVerbose, Text: "/verbose"})

	waitFor(t, 200*time.Millisecond, func() bool { return len(tg.sentSnapshot()) == 1 })
	got := tg.sentSnapshot()[0].Text
	for _, forbidden := range []string{"plain-secret", "**Injected:**", "\n**"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("/verbose persist error leaked unsafe text %q in:\n%s", forbidden, got)
		}
	}
	if !strings.Contains(got, "could not save to config: [redacted]") {
		t.Fatalf("/verbose persist error missing redaction marker:\n%s", got)
	}
}

func TestManager_Inbound_ModelCommandSanitizesModelAndProvider(t *testing.T) {
	ch := newFakeChannel("slack")
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:           map[string]string{"slack": "42"},
		LiveTurnActiveModel:    func() string { return "bad`**model**" },
		LiveTurnActiveProvider: func() string { return "openai\n# injected" },
	}, &fakeKernel{}, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	ch.pushInbound(InboundEvent{Platform: "slack", ChatID: "42", UserID: "u", MsgID: "m", Kind: EventModel, Text: "/model"})

	waitFor(t, 200*time.Millisecond, func() bool { return len(ch.sentSnapshot()) == 1 })
	got := ch.sentSnapshot()[0].Text
	for _, forbidden := range []string{"bad`**model**", "# injected", "\n#"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("/model leaked unsafe field %q in:\n%s", forbidden, got)
		}
	}
	assertContainsAll(t, got, "🤖 **Model:** `bad'''model''`", "📡 **Provider:** `openai ＃ injected`")
}

func TestManager_Inbound_ProfileCommandSanitizesHomePath(t *testing.T) {
	t.Setenv("GORMES_HOME", "/tmp/gormes`**home**\n# injected")
	ch := newFakeChannel("slack")
	m := NewManagerWithSubmitter(ManagerConfig{AllowedChats: map[string]string{"slack": "42"}}, &fakeKernel{}, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	ch.pushInbound(InboundEvent{Platform: "slack", ChatID: "42", UserID: "u", MsgID: "m", Kind: EventProfile, Text: "/profile"})

	waitFor(t, 200*time.Millisecond, func() bool { return len(ch.sentSnapshot()) == 1 })
	got := ch.sentSnapshot()[0].Text
	for _, forbidden := range []string{"`**home**", "# injected", "\n#"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("/profile leaked unsafe home path %q in:\n%s", forbidden, got)
		}
	}
	assertContainsAll(t, got, "👤 **Profile:** `(default)`", "📂 **Home:** `/tmp/gormes'''home'' ＃ injected`")
}

func TestManager_Inbound_GatewayCommandSanitizesConnectedPlatformNames(t *testing.T) {
	ch := newFakeChannel("slack`**bot**\n# injected")
	m := NewManagerWithSubmitter(ManagerConfig{AllowedChats: map[string]string{"slack`**bot**\n# injected": "42"}}, &fakeKernel{}, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	ch.pushInbound(InboundEvent{Platform: ch.Name(), ChatID: "42", UserID: "u", MsgID: "m", Kind: EventGateway, Text: "/gateway"})

	waitFor(t, 200*time.Millisecond, func() bool { return len(ch.sentSnapshot()) == 1 })
	got := ch.sentSnapshot()[0].Text
	for _, forbidden := range []string{"`**bot**", "# injected", "\n#"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("/gateway leaked unsafe platform name %q in:\n%s", forbidden, got)
		}
	}
	assertContainsAll(t, got, "📡 **Connected Platforms:** slack'''bot'' ＃ injected")
}

func TestManager_ToolProgressModeUsesHermesPlatformDefaults(t *testing.T) {
	m := NewManagerWithSubmitter(ManagerConfig{}, &fakeKernel{}, slog.Default())
	for _, tc := range []struct {
		platform string
		want     string
	}{
		{platform: "telegram", want: "all"},
		{platform: "discord", want: "all"},
		{platform: "slack", want: "off"},
		{platform: "mattermost", want: "new"},
		{platform: "signal", want: "off"},
		{platform: "email", want: "off"},
		{platform: "unknown", want: "all"},
	} {
		if got := m.toolProgressMode(tc.platform); got != tc.want {
			t.Fatalf("toolProgressMode(%q) = %q, want Hermes platform default %q", tc.platform, got, tc.want)
		}
	}
}

func TestManager_Inbound_AppendsAttachmentsToSubmittedText(t *testing.T) {
	tg := newFakeChannel("dingtalk")
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"dingtalk": "dm-42"},
	}, fk, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "dingtalk",
		ChatID:   "dm-42",
		UserID:   "staff-1",
		MsgID:    "msg-1",
		Kind:     EventSubmit,
		Text:     "please inspect",
		Attachments: []Attachment{
			{
				Kind:      "image",
				URL:       "https://media.dingtalk.example/image.png",
				MediaType: "image",
				SourceID:  "img-code-1",
			},
			{
				Kind:      "file",
				URL:       "file-code-timeout",
				MediaType: "application/octet-stream",
				FileName:  "report.pdf",
				SourceID:  "file-code-timeout",
				Error:     "dingtalk: media download: 429 rate limit",
			},
		},
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})
	got := fk.submitsSnapshot()[0]
	want := "please inspect\n\nAttachments:\n- image: https://media.dingtalk.example/image.png (mediaType=image, sourceId=img-code-1)\n- file report.pdf: file-code-timeout (mediaType=application/octet-stream, sourceId=file-code-timeout, error=dingtalk: media download: 429 rate limit)"
	if got.Text != want {
		t.Fatalf("submitted text = %q, want %q", got.Text, want)
	}
}

func TestManager_Inbound_SubmitInjectsReplyToText(t *testing.T) {
	sl := newFakeChannel("slack")
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"slack": "C123"},
	}, fk, slog.Default())
	if err := m.Register(sl); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	sl.pushInbound(InboundEvent{
		Platform:    "slack",
		ChatID:      "C123",
		UserID:      "U1",
		ThreadID:    "1000.0",
		MsgID:       "1000.5",
		MessageID:   "1000.5",
		Kind:        EventSubmit,
		Text:        "please show details",
		ReplyToText: "cron summary: 3 new emails",
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})
	got := fk.submitsSnapshot()[0]
	want := "[Replying to: \"cron summary: 3 new emails\"]\n\nplease show details"
	if got.Text != want {
		t.Fatalf("submitted text = %q, want %q", got.Text, want)
	}
	for _, wantContext := range []string{
		"**Thread ID:** `1000.0`",
		"**Message ID:** `1000.5`",
	} {
		if !strings.Contains(got.SessionContext, wantContext) {
			t.Fatalf("SessionContext missing %q in:\n%s", wantContext, got.SessionContext)
		}
	}
}

func TestManager_Inbound_BlockedChat_NoSubmit(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "999", Kind: EventSubmit, Text: "hi",
	})

	time.Sleep(50 * time.Millisecond)
	if n := len(fk.submitsSnapshot()); n != 0 {
		t.Errorf("blocked chat should produce 0 submits, got %d", n)
	}
}

func TestManager_Inbound_Cancel(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", Kind: EventCancel,
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		s := fk.submitsSnapshot()
		return len(s) == 1 && s[0].Kind == kernel.PlatformEventCancel
	})
}

func TestManager_Inbound_Reset(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", Kind: EventReset,
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		fk.mu.Lock()
		defer fk.mu.Unlock()
		return fk.resets == 1
	})
}

func TestManager_Inbound_ResetErrorSanitizesOperatorReply(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{resetErr: errors.New("reset failed\n**Injected:** token=plain-secret")}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventReset})

	waitFor(t, 200*time.Millisecond, func() bool { return len(tg.sentSnapshot()) == 1 })
	got := tg.sentSnapshot()[0].Text
	for _, forbidden := range []string{"plain-secret", "**Injected:**", "\n"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("reset error leaked unsafe text %q in %q", forbidden, got)
		}
	}
	if !strings.Contains(got, "Session reset failed: [redacted]") {
		t.Fatalf("reset error missing redaction marker: %q", got)
	}
}

func TestManager_Inbound_Start_RepliesHelp(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", Kind: EventStart,
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		sent := tg.sentSnapshot()
		return len(sent) == 1 &&
			sent[0].ChatID == "42" &&
			strings.Contains(sent[0].Text, "/help") &&
			strings.Contains(sent[0].Text, "/new") &&
			strings.Contains(sent[0].Text, "/stop")
	})
	if n := len(fk.submitsSnapshot()); n != 0 {
		t.Errorf("EventStart should not submit to kernel, got %d", n)
	}
}

func TestManager_Inbound_SubmitCreatesAndRefreshesConversationalSessionMetadata(t *testing.T) {
	ctx := context.Background()
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}
	smap := session.NewMemMap()
	now := time.Date(2026, 4, 30, 1, 0, 0, 0, time.UTC)

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		Now:          func() time.Time { return now },
	}, fk, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := m.handleInbound(ctx, InboundEvent{
		Platform: "telegram", ChatID: "42", UserID: "user-juan", MsgID: "m1",
		Kind: EventSubmit, Text: "hello",
	}); err != nil {
		t.Fatal(err)
	}
	first := fk.submitsSnapshot()[0]
	if first.SessionID == "" || first.SessionID == "telegram:42" || !strings.HasPrefix(first.SessionID, "20260430_010000_") {
		t.Fatalf("first submit SessionID = %q, want generated Hermes-style session id", first.SessionID)
	}
	mapped, err := smap.Get(ctx, "telegram:42")
	if err != nil {
		t.Fatalf("Get session map: %v", err)
	}
	if mapped != first.SessionID {
		t.Fatalf("session map = %q, want first submit session %q", mapped, first.SessionID)
	}
	meta, ok, err := smap.GetMetadata(ctx, first.SessionID)
	if err != nil {
		t.Fatalf("GetMetadata first: %v", err)
	}
	if !ok {
		t.Fatal("normal submit did not create session metadata")
	}
	if meta.Source != "telegram" || meta.ChatID != "42" || meta.UserID != "user-juan" {
		t.Fatalf("metadata source/chat/user = %q/%q/%q, want telegram/42/user-juan", meta.Source, meta.ChatID, meta.UserID)
	}
	if meta.CreatedAt != now.Unix() || meta.UpdatedAt != now.Unix() {
		t.Fatalf("metadata times = created %d updated %d, want %d", meta.CreatedAt, meta.UpdatedAt, now.Unix())
	}

	m.clearTurn()
	now = now.Add(5 * time.Minute)
	if err := m.handleInbound(ctx, InboundEvent{
		Platform: "telegram", ChatID: "42", UserID: "user-juan", MsgID: "m2",
		Kind: EventSubmit, Text: "again",
	}); err != nil {
		t.Fatal(err)
	}
	second := fk.submitsSnapshot()[1]
	if second.SessionID != first.SessionID {
		t.Fatalf("second submit SessionID = %q, want stable %q", second.SessionID, first.SessionID)
	}
	meta, ok, err = smap.GetMetadata(ctx, first.SessionID)
	if err != nil {
		t.Fatalf("GetMetadata second: %v", err)
	}
	if !ok {
		t.Fatal("metadata disappeared after refresh")
	}
	if meta.CreatedAt != time.Date(2026, 4, 30, 1, 0, 0, 0, time.UTC).Unix() {
		t.Fatalf("CreatedAt changed to %d", meta.CreatedAt)
	}
	if meta.UpdatedAt != now.Unix() {
		t.Fatalf("UpdatedAt = %d, want refreshed %d", meta.UpdatedAt, now.Unix())
	}
}

func TestManager_Outbound_StreamsToPinnedChannel(t *testing.T) {
	tg := newTypingActionFakeChannel("telegram")
	frames := make(chan kernel.RenderFrame, 8)
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		CoalesceMs:   10,
	}, fk, slog.Default())
	m.setRenderChan(frames)
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", MsgID: "m1",
		Kind: EventSubmit, Text: "hi",
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})

	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseStreaming, DraftText: "partial",
	}

	waitFor(t, 500*time.Millisecond, func() bool {
		sent := tg.sentSnapshot()
		return len(sent) >= 1 && strings.Contains(sent[0].Text, "partial")
	})
}

func TestManager_Outbound_ToolProgressPersistsAsSeparateMessage(t *testing.T) {
	tg := newTypingActionFakeChannel("telegram")
	frames := make(chan kernel.RenderFrame, 8)
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		CoalesceMs:   10,
	}, fk, slog.Default())
	m.setRenderChan(frames)
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", MsgID: "m1",
		Kind: EventSubmit, Text: "summarize this reddit post",
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})

	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseStreaming,
		SoulEvents: []kernel.SoulEntry{
			{At: time.Now(), Text: "tool: browser_navigate: https://www.reddit.com/r/WebAfterAI/s/example"},
			{At: time.Now(), Text: "tool: terminal: curl -L https://example.test/post.json"},
		},
	}
	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []llm.Message{
			{Role: "user", Content: "summarize this reddit post"},
			{Role: "assistant", Content: "I read the Reddit post via Reddit's embed endpoint."},
		},
	}

	waitFor(t, 500*time.Millisecond, func() bool {
		sent := tg.sentSnapshot()
		return len(sent) >= 2 &&
			strings.Contains(sent[0].Text, "browser") &&
			strings.Contains(sent[1].Text, "I read the Reddit post")
	})
	sent := tg.sentSnapshot()
	if strings.Contains(sent[1].Text, "browser_navigate") || strings.Contains(sent[1].Text, "terminal") {
		t.Fatalf("final answer contains tool progress; sent=%#v", sent)
	}
}

func TestManager_Outbound_ToolProgressOffSuppressesSeparateMessage(t *testing.T) {
	tg := newTypingActionFakeChannel("telegram")
	frames := make(chan kernel.RenderFrame, 8)
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:     map[string]string{"telegram": "42"},
		CoalesceMs:       10,
		ToolProgressMode: "off",
	}, fk, slog.Default())
	m.setRenderChan(frames)
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", MsgID: "m1",
		Kind: EventSubmit, Text: "summarize this reddit post",
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})

	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseStreaming,
		SoulEvents: []kernel.SoulEntry{
			{At: time.Now(), Text: "tool: browser_navigate: https://www.reddit.com/r/WebAfterAI/s/example"},
			{At: time.Now(), Text: "tool: terminal: curl -L https://example.test/post.json"},
		},
	}
	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []llm.Message{
			{Role: "user", Content: "summarize this reddit post"},
			{Role: "assistant", Content: "I read the Reddit post via Reddit's embed endpoint."},
		},
	}

	waitFor(t, 500*time.Millisecond, func() bool {
		sent := tg.sentSnapshot()
		return len(sent) >= 1 && strings.Contains(sent[len(sent)-1].Text, "I read the Reddit post")
	})
	for _, msg := range tg.sentSnapshot() {
		if strings.Contains(msg.Text, "browser_navigate") || strings.Contains(msg.Text, "terminal") {
			t.Fatalf("tool_progress=off sent tool progress message; sent=%#v", tg.sentSnapshot())
		}
	}
}

func TestManager_Outbound_LongFinalAnswerPaginates(t *testing.T) {
	tg := newChannelOnlyFake("telegram")
	frames := make(chan kernel.RenderFrame, 2)
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	m.setRenderChan(frames)
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", MsgID: "m1",
		Kind: EventSubmit, Text: "extract all documentation",
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})

	tail := "TAIL multi agent routing documentation preserved"
	longAnswer := "Extracted documentation:\n" + strings.Repeat("A", maxMessageLen) + "\n" + tail
	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []llm.Message{
			{Role: "user", Content: "extract all documentation"},
			{Role: "assistant", Content: longAnswer},
		},
	}

	waitFor(t, 500*time.Millisecond, func() bool {
		sent := tg.sentSnapshot()
		return len(sent) >= 1
	})
	sent := tg.sentSnapshot()
	if len(sent) != 2 {
		t.Fatalf("sent messages = %d, want 2 pages; sent=%#v", len(sent), sent)
	}
	if !strings.Contains(sent[0].Text, `\(1/2\)`) || !strings.Contains(sent[1].Text, `\(2/2\)`) {
		t.Fatalf("long final answer missing Telegram-safe page markers; sent=%#v", sent)
	}
	if !strings.Contains(sent[1].Text, tail) {
		t.Fatalf("long final answer dropped tail %q; sent=%#v", tail, sent)
	}
	for i, msg := range sent {
		if got := len([]rune(msg.Text)); got > maxMessageLen {
			t.Fatalf("page %d length = %d, want <= %d", i+1, got, maxMessageLen)
		}
	}
	if strings.Contains(strings.Join([]string{sent[0].Text, sent[1].Text}, ""), "…") {
		t.Fatalf("long final answer was ellipsis-truncated instead of paginated; sent=%#v", sent)
	}
}

func TestManager_Outbound_FreshFinalAfterSendsFreshFinal(t *testing.T) {
	tg := &freshFinalFakeChannel{fakeChannel: newFakeChannel("telegram")}
	frames := make(chan kernel.RenderFrame, 8)
	fk := &fakeKernel{}
	var nowMu sync.Mutex
	now := time.Date(2026, 4, 27, 1, 0, 0, 0, time.UTC)
	readNow := func() time.Time {
		nowMu.Lock()
		defer nowMu.Unlock()
		return now
	}
	advanceNow := func(d time.Duration) {
		nowMu.Lock()
		defer nowMu.Unlock()
		now = now.Add(d)
	}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:    map[string]string{"telegram": "42"},
		CoalesceMs:      10,
		FreshFinalAfter: time.Minute,
		Now:             readNow,
	}, fk, slog.Default())
	m.setRenderChan(frames)
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", MsgID: "m1",
		Kind: EventSubmit, Text: "hi",
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})

	frames <- kernel.RenderFrame{Phase: kernel.PhaseStreaming, DraftText: "partial"}
	waitFor(t, 500*time.Millisecond, func() bool {
		sent := tg.sentSnapshot()
		return len(sent) >= 1 && strings.Contains(sent[0].Text, "partial")
	})
	oldID := tg.sentSnapshot()[0].MsgID

	advanceNow(time.Minute)
	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []llm.Message{
			{Role: "user", Content: "hi"},
			{Role: "assistant", Content: "done"},
		},
	}

	waitFor(t, 500*time.Millisecond, func() bool {
		return len(tg.sentSnapshot()) >= 2
	})
	sent := tg.sentSnapshot()
	if sent[1].Text != "done" {
		t.Fatalf("fresh final text = %q, want %q; sent=%#v", sent[1].Text, "done", sent)
	}
	edits := tg.editsSnapshot()
	if len(edits) != 0 {
		t.Fatalf("EditMessageFinal calls = %d, want none before fresh final send; edits=%#v", len(edits), edits)
	}
	if got, want := tg.deletesSnapshot(), []fakeDelete{{ChatID: "42", MsgID: oldID}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("DeleteMessage calls = %#v, want %#v", got, want)
	}
}

type replyPlaceholderFakeChannel struct {
	*fakeChannel

	replyPlaceholders []fakeReplyPlaceholder
	replies           []fakeReplySend
}

type fakeReplyPlaceholder struct{ ChatID, ReplyToMsgID string }
type fakeReplySend struct{ ChatID, ReplyToMsgID, Text string }

func (f *replyPlaceholderFakeChannel) SendReplyPlaceholder(ctx context.Context, chatID, replyToMsgID string) (string, error) {
	f.replyPlaceholders = append(f.replyPlaceholders, fakeReplyPlaceholder{ChatID: chatID, ReplyToMsgID: replyToMsgID})
	return f.fakeChannel.SendPlaceholder(ctx, chatID)
}

func (f *replyPlaceholderFakeChannel) SendReply(ctx context.Context, chatID, replyToMsgID, text string) (string, error) {
	f.replies = append(f.replies, fakeReplySend{ChatID: chatID, ReplyToMsgID: replyToMsgID, Text: text})
	return f.fakeChannel.Send(ctx, chatID, text)
}

func (f *replyPlaceholderFakeChannel) SendChatAction(context.Context, string, string) error {
	return nil
}

func TestManager_Outbound_FirstStreamingContentRepliesToInboundMessage(t *testing.T) {
	tg := &replyPlaceholderFakeChannel{fakeChannel: newFakeChannel("telegram")}
	frames := make(chan kernel.RenderFrame, 2)
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	m.setRenderChan(frames)
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		UserID:   "u",
		MsgID:    "telegram-user-msg-1",
		Kind:     EventSubmit,
		Text:     "hello",
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})

	frames <- kernel.RenderFrame{Phase: kernel.PhaseStreaming, SessionID: "sess-1", DraftText: "partial"}
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(tg.replies) == 1
	})

	if got := tg.replies[0]; got.ChatID != "42" || got.ReplyToMsgID != "telegram-user-msg-1" || !strings.Contains(got.Text, "partial") {
		t.Fatalf("reply send = %+v, want chat 42 replying to inbound message with stream content", got)
	}
	if len(tg.replyPlaceholders) != 0 {
		t.Fatalf("reply placeholders = %#v, want none for Hermes-style first content send", tg.replyPlaceholders)
	}
}

func TestManager_Outbound_ConnectingStartsTypingWithoutHourglassMessage(t *testing.T) {
	tg := newTypingActionFakeChannel("telegram")
	frames := make(chan kernel.RenderFrame, 2)
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		CoalesceMs:   10,
	}, fk, slog.Default())
	m.setRenderChan(frames)
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		UserID:   "u",
		MsgID:    "telegram-user-msg-1",
		Kind:     EventSubmit,
		Text:     "hello",
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})

	frames <- kernel.RenderFrame{Phase: kernel.PhaseConnecting, SessionID: "sess-1", StatusText: "connecting"}
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(tg.actionSnapshot()) == 1
	})

	if sent := tg.sentSnapshot(); len(sent) != 0 {
		t.Fatalf("connecting frame sent visible messages = %#v, want only Telegram typing indicator", sent)
	}
}

func TestManager_Outbound_NonEditableChannelUsesPlainSendForInterimAndFinal(t *testing.T) {
	ch := newChannelOnlyFake("plainchat")
	if _, ok := any(ch).(placeholderEditor); ok {
		t.Fatal("channel-only fixture unexpectedly implements placeholder editing")
	}
	if _, ok := any(ch).(PlaceholderCapable); ok {
		t.Fatal("channel-only fixture unexpectedly implements SendPlaceholder")
	}
	if _, ok := any(ch).(MessageEditor); ok {
		t.Fatal("channel-only fixture unexpectedly implements EditMessage")
	}

	frames := make(chan kernel.RenderFrame, 8)
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"plainchat": "thread-42"},
		CoalesceMs:   10,
	}, fk, slog.Default())
	m.setRenderChan(frames)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	ch.pushInbound(InboundEvent{
		Platform: "plainchat", ChatID: "thread-42", MsgID: "origin-msg",
		Kind: EventSubmit, Text: "hi",
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})

	frames <- kernel.RenderFrame{
		Phase:     kernel.PhaseStreaming,
		DraftText: "I'll inspect the repo first.",
	}
	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []llm.Message{
			{Role: "user", Content: "hi"},
			{Role: "assistant", Content: "done"},
		},
	}

	waitFor(t, 500*time.Millisecond, func() bool {
		return len(ch.sentSnapshot()) == 2
	})

	got := ch.sentSnapshot()
	wantTexts := []string{"I'll inspect the repo first. ▉", "done"}
	for i, want := range wantTexts {
		if got[i].ChatID != "thread-42" {
			t.Fatalf("sent[%d].ChatID = %q, want original chat target %q", i, got[i].ChatID, "thread-42")
		}
		if i == 0 && strings.Contains(got[i].Text, want) {
			continue
		}
		if got[i].Text != want {
			t.Fatalf("sent[%d].Text = %q, want %q; sends=%#v", i, got[i].Text, want, got)
		}
	}
}

func TestManager_Outbound_FinalFrameClearsTurn(t *testing.T) {
	tg := newFakeChannel("telegram")
	frames := make(chan kernel.RenderFrame, 8)
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		CoalesceMs:   10,
	}, fk, slog.Default())
	m.setRenderChan(frames)
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", MsgID: "m1",
		Kind: EventSubmit, Text: "hi",
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})

	frames <- kernel.RenderFrame{Phase: kernel.PhaseStreaming, DraftText: "p1"}
	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []llm.Message{
			{Role: "user", Content: "hi"},
			{Role: "assistant", Content: "hello back"},
		},
	}

	waitFor(t, 500*time.Millisecond, func() bool {
		m.turnMu.Lock()
		defer m.turnMu.Unlock()
		return m.turnPlatform == ""
	})
}

func TestManager_ActiveTurnQueuesFollowUpUntilTerminalFrame(t *testing.T) {
	tg := newFakeChannel("telegram")
	dc := newFakeChannel("discord")
	frames := make(chan kernel.RenderFrame, 8)
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{
			"telegram": "42",
			"discord":  "99",
		},
		CoalesceMs: 10,
	}, fk, slog.Default())
	m.setRenderChan(frames)
	_ = m.Register(tg)
	_ = m.Register(dc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", MsgID: "m1",
		Kind: EventSubmit, Text: "first",
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})

	dc.pushInbound(InboundEvent{
		Platform: "discord", ChatID: "99", MsgID: "m2",
		Kind: EventSubmit, Text: "second",
	})
	time.Sleep(30 * time.Millisecond)
	if got := fk.submitsSnapshot(); len(got) != 1 {
		t.Fatalf("active-turn follow-up submitted immediately: submits=%#v", got)
	}

	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []llm.Message{
			{Role: "user", Content: "first"},
			{Role: "assistant", Content: "first answer"},
		},
	}

	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 2 && len(tg.sentSnapshot()) == 1
	})
	if got := tg.sentSnapshot()[0].Text; got != "first answer" {
		t.Fatalf("first terminal reply routed to telegram = %q, want first answer", got)
	}
	if sent := dc.sentSnapshot(); len(sent) != 0 {
		t.Fatalf("discord received terminal reply for active telegram turn: %#v", sent)
	}
	got := fk.submitsSnapshot()
	if got[1].Text != "second" {
		t.Fatalf("drained follow-up text = %q, want second", got[1].Text)
	}
}

func TestManager_LateArrivalDuringFollowUpDrainQueuesBehindDrainedTurn(t *testing.T) {
	tg := newFakeChannel("telegram")
	frames := make(chan kernel.RenderFrame, 8)
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		CoalesceMs:   10,
	}, fk, slog.Default())
	m.setRenderChan(frames)
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", MsgID: "m1",
		Kind: EventSubmit, Text: "first",
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", MsgID: "m2",
		Kind: EventSubmit, Text: "second",
	})
	time.Sleep(30 * time.Millisecond)
	if got := fk.submitsSnapshot(); len(got) != 1 {
		t.Fatalf("queued follow-up submitted before active turn drained: submits=%#v", got)
	}

	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []llm.Message{
			{Role: "user", Content: "first"},
			{Role: "assistant", Content: "first answer"},
		},
	}
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 2
	})

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", MsgID: "m3",
		Kind: EventSubmit, Text: "third",
	})
	time.Sleep(30 * time.Millisecond)
	if got := fk.submitsSnapshot(); len(got) != 2 {
		t.Fatalf("late arrival submitted during drained turn: submits=%#v", got)
	}

	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []llm.Message{
			{Role: "user", Content: "second"},
			{Role: "assistant", Content: "second answer"},
		},
	}
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 3
	})

	got := fk.submitsSnapshot()
	for i, want := range []string{"first", "second", "third"} {
		if got[i].Text != want {
			t.Fatalf("submit %d text = %q, want %q; submits=%#v", i, got[i].Text, want, got)
		}
	}
}

func TestManager_ShutdownWaitsForActiveTurn(t *testing.T) {
	m := NewManagerWithSubmitter(ManagerConfig{}, &fakeKernel{}, slog.Default())
	m.pinTurn("telegram", "42", "m1")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- m.Shutdown(shutdownCtx)
	}()

	time.Sleep(20 * time.Millisecond)
	select {
	case err := <-done:
		t.Fatalf("Shutdown returned before turn cleared: %v", err)
	default:
	}

	m.clearTurn()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Shutdown: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Shutdown did not return after turn cleared")
	}
}

func TestManager_ShutdownTimesOutWhileTurnActive(t *testing.T) {
	m := NewManagerWithSubmitter(ManagerConfig{}, &fakeKernel{}, slog.Default())
	m.pinTurn("telegram", "42", "m1")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := m.Shutdown(shutdownCtx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown error = %v, want deadline exceeded", err)
	}
}

func TestManager_Inbound_SubmitRejectedDuringShutdown(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second)
	defer shutdownCancel()
	if err := m.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", MsgID: "m1",
		Kind: EventSubmit, Text: "hello",
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		return len(tg.sentSnapshot()) == 1
	})

	if n := len(fk.submitsSnapshot()); n != 0 {
		t.Fatalf("submit count = %d, want 0", n)
	}
	if got := tg.sentSnapshot()[0].Text; !strings.Contains(strings.ToLower(got), "shutting down") {
		t.Fatalf("shutdown reply = %q, want shutdown notice", got)
	}
}

func TestManager_Inbound_CancelAllowedDuringShutdown(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second)
	defer shutdownCancel()
	if err := m.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", Kind: EventCancel,
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		s := fk.submitsSnapshot()
		return len(s) == 1 && s[0].Kind == kernel.PlatformEventCancel
	})

	if n := len(tg.sentSnapshot()); n != 0 {
		t.Fatalf("shutdown cancel should not send a reply, got %d messages", n)
	}
}

func TestManager_RunCleansStartupFailedChannelAndReturnsOriginalError(t *testing.T) {
	startupErr := errors.New("discord: open session: denied")
	ch := &startupFailedChannel{name: "discord", runErr: startupErr}

	m := NewManagerWithSubmitter(ManagerConfig{}, &fakeKernel{}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := m.Run(ctx)
	if !errors.Is(err, startupErr) {
		t.Fatalf("Run error = %v, want startup error %v", err, startupErr)
	}
	if got := ch.disconnectCount(); got != 1 {
		t.Fatalf("disconnect count = %d, want 1", got)
	}
}

func TestManager_RunLogsCleanupErrorWithoutMaskingStartupFailure(t *testing.T) {
	startupErr := errors.New("discord: open session: denied")
	cleanupErr := errors.New("discord: close partial session: denied")
	ch := &startupFailedChannel{name: "discord", runErr: startupErr, disconnectErr: cleanupErr}
	var logs bytes.Buffer

	m := NewManagerWithSubmitter(ManagerConfig{}, &fakeKernel{}, slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := m.Run(ctx)
	if !errors.Is(err, startupErr) {
		t.Fatalf("Run error = %v, want startup error %v", err, startupErr)
	}
	gotLogs := logs.String()
	if !strings.Contains(gotLogs, "defensive channel disconnect after failed startup raised") ||
		!strings.Contains(gotLogs, "discord") ||
		!strings.Contains(gotLogs, cleanupErr.Error()) {
		t.Fatalf("cleanup failure log = %q, want channel name and cleanup error", gotLogs)
	}
}

func TestManager_RunChannelDisconnectTimeoutReturnsStartupFailure(t *testing.T) {
	t.Setenv("HERMES_GATEWAY_ADAPTER_DISCONNECT_TIMEOUT", "0.001")
	startupErr := errors.New("discord: open session: denied")
	releaseDisconnect := make(chan struct{})
	defer close(releaseDisconnect)
	ch := &startupFailedChannel{name: "discord", runErr: startupErr, disconnectWait: releaseDisconnect}
	var logs bytes.Buffer

	m := NewManagerWithSubmitter(ManagerConfig{}, &fakeKernel{}, slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- m.Run(ctx)
	}()

	select {
	case err := <-done:
		if !errors.Is(err, startupErr) {
			t.Fatalf("Run error = %v, want startup error %v", err, startupErr)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Run did not return before channel disconnect timeout")
	}
	if got := ch.disconnectCount(); got != 1 {
		t.Fatalf("disconnect count = %d, want 1", got)
	}
	gotLogs := logs.String()
	if !strings.Contains(gotLogs, "defensive channel disconnect after failed startup timed out") ||
		!strings.Contains(gotLogs, "discord") {
		t.Fatalf("disconnect timeout log = %q, want channel timeout evidence", gotLogs)
	}
}

func TestManager_RunCleansFailedStartupChannelWithoutStoppingHealthyChannels(t *testing.T) {
	startupErr := errors.New("discord: open session: denied")
	failed := &startupFailedChannel{name: "discord", runErr: startupErr}
	healthy := newFakeChannel("telegram")

	m := NewManagerWithSubmitter(ManagerConfig{}, &fakeKernel{}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	if err := m.Register(failed); err != nil {
		t.Fatalf("Register failed channel: %v", err)
	}
	if err := m.Register(healthy); err != nil {
		t.Fatalf("Register healthy channel: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- m.Run(ctx)
	}()

	waitFor(t, 200*time.Millisecond, func() bool {
		return failed.disconnectCount() == 1
	})

	select {
	case err := <-done:
		t.Fatalf("Run returned while a healthy channel was still running: %v", err)
	default:
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run after context cancellation = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}

func TestManager_RunShutdownDisconnectTimeout(t *testing.T) {
	t.Setenv("HERMES_GATEWAY_ADAPTER_DISCONNECT_TIMEOUT", "0.001")
	releaseRun := make(chan struct{})
	releaseDisconnect := make(chan struct{})
	defer close(releaseRun)
	defer close(releaseDisconnect)

	ch := &shutdownBlockedChannel{
		name:              "discord",
		releaseRun:        releaseRun,
		releaseDisconnect: releaseDisconnect,
		started:           make(chan struct{}),
	}
	var logs bytes.Buffer
	m := NewManagerWithSubmitter(ManagerConfig{}, &fakeKernel{}, slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- m.Run(ctx)
	}()
	<-ch.started

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run after planned shutdown = %v, want nil", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Run did not return before shutdown disconnect timeout")
	}
	if got := ch.disconnectCount(); got != 1 {
		t.Fatalf("disconnect count = %d, want 1", got)
	}
	gotLogs := logs.String()
	if !strings.Contains(gotLogs, "defensive channel disconnect during shutdown timed out") ||
		!strings.Contains(gotLogs, "discord") {
		t.Fatalf("shutdown disconnect timeout log = %q, want shutdown channel timeout evidence", gotLogs)
	}
}

type startupFailedChannel struct {
	name           string
	runErr         error
	disconnectErr  error
	disconnectWait <-chan struct{}

	mu          sync.Mutex
	disconnects int
}

func (c *startupFailedChannel) Name() string { return c.name }

func (c *startupFailedChannel) Run(context.Context, chan<- InboundEvent) error {
	return c.runErr
}

func (c *startupFailedChannel) Send(context.Context, string, string) (string, error) {
	return "", errors.New("startup failed channel cannot send")
}

func (c *startupFailedChannel) Disconnect(context.Context) error {
	c.mu.Lock()
	c.disconnects++
	c.mu.Unlock()
	if c.disconnectWait != nil {
		<-c.disconnectWait
	}
	return c.disconnectErr
}

func (c *startupFailedChannel) disconnectCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.disconnects
}

type shutdownBlockedChannel struct {
	name              string
	releaseRun        <-chan struct{}
	releaseDisconnect <-chan struct{}
	started           chan struct{}

	mu          sync.Mutex
	disconnects int
}

func (c *shutdownBlockedChannel) Name() string { return c.name }

func (c *shutdownBlockedChannel) Run(context.Context, chan<- InboundEvent) error {
	close(c.started)
	<-c.releaseRun
	return nil
}

func (c *shutdownBlockedChannel) Send(context.Context, string, string) (string, error) {
	return "", errors.New("shutdown blocked channel cannot send")
}

func (c *shutdownBlockedChannel) Disconnect(context.Context) error {
	c.mu.Lock()
	c.disconnects++
	c.mu.Unlock()
	<-c.releaseDisconnect
	return nil
}

func (c *shutdownBlockedChannel) disconnectCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.disconnects
}

func TestManager_FailedProviderFrameSanitizesAndClearsActiveTurn(t *testing.T) {
	ch := newChannelOnlyFake("telegram")
	m := NewManager(ManagerConfig{}, nil, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	m.pinTurn("telegram", "42", "msg-1")
	var co *coalescer
	var coCancel context.CancelFunc
	m.dispatchFrame(context.Background(), kernel.RenderFrame{
		Phase:     kernel.PhaseFailed,
		LastError: "Forbidden: <html><body><svg>bad</svg> secret sk-test-123</body></html>",
	}, &co, &coCancel)

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("failed provider frame should send exactly one error reply, got %d", len(sent))
	}
	got := sent[0].Text
	for _, forbidden := range []string{"<html", "<svg", "sk-test-123", "secret"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("gateway leaked provider HTML/secret marker %q in %q", forbidden, got)
		}
	}
	if !strings.Contains(got, "provider returned HTML error body") {
		t.Fatalf("gateway error reply = %q, want sanitized provider HTML evidence", got)
	}
	if m.hasActiveTurn() {
		t.Fatalf("failed provider frame must clear active turn so admission does not wedge")
	}
}

func TestManager_StartupIdleFrameDoesNotClearPinnedTurn(t *testing.T) {
	ch := newChannelOnlyFake("telegram")
	m := NewManager(ManagerConfig{}, nil, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	m.pinTurn("telegram", "42", "msg-1")
	var co *coalescer
	var coCancel context.CancelFunc
	m.dispatchFrame(context.Background(), kernel.RenderFrame{
		Phase:      kernel.PhaseIdle,
		StatusText: "idle",
	}, &co, &coCancel)

	if sent := ch.sentSnapshot(); len(sent) != 0 {
		t.Fatalf("startup idle frame should not send or finalize active turn: %#v", sent)
	}
	if !m.hasActiveTurn() {
		t.Fatalf("startup idle frame cleared active turn before provider result arrived")
	}

	m.dispatchFrame(context.Background(), kernel.RenderFrame{
		Phase:     kernel.PhaseFailed,
		LastError: "Forbidden: <html><body>bad</body></html>",
	}, &co, &coCancel)

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("failed provider frame should send exactly one error reply, got %d", len(sent))
	}
	if !strings.Contains(sent[0].Text, "provider returned HTML error body") {
		t.Fatalf("gateway error reply = %q, want sanitized provider HTML evidence", sent[0].Text)
	}
	if m.hasActiveTurn() {
		t.Fatalf("failed provider frame must still clear active turn after stale startup idle")
	}
}

func (f *fakeKernel) resetsSnapshot() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.resets
}

func TestCheckAutoReset_NonePolicyDoesNotReset(t *testing.T) {
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	smap := session.NewMemMap()
	fk := &fakeKernel{}

	if err := smap.PutMetadata(context.Background(), session.Metadata{
		SessionID: "test-session",
		UpdatedAt: now.Add(-999 * time.Hour).Unix(),
	}); err != nil {
		t.Fatal(err)
	}

	m := NewManagerWithSubmitter(ManagerConfig{
		SessionMap:         smap,
		Now:                func() time.Time { return now },
		SessionResetPolicy: "none",
	}, fk, slog.Default())

	m.checkAutoReset(context.Background(), "test-session")
	if n := fk.resetsSnapshot(); n != 0 {
		t.Errorf("none policy: resets = %d, want 0", n)
	}
}

func TestCheckAutoReset_EmptyPolicyDoesNotReset(t *testing.T) {
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	smap := session.NewMemMap()
	fk := &fakeKernel{}

	if err := smap.PutMetadata(context.Background(), session.Metadata{
		SessionID: "test-session",
		UpdatedAt: now.Add(-999 * time.Hour).Unix(),
	}); err != nil {
		t.Fatal(err)
	}

	m := NewManagerWithSubmitter(ManagerConfig{SessionMap: smap, Now: func() time.Time { return now }}, fk, slog.Default())

	m.checkAutoReset(context.Background(), "test-session")
	if n := fk.resetsSnapshot(); n != 0 {
		t.Errorf("empty policy: resets = %d, want 0", n)
	}
}

func TestCheckAutoReset_InactivityPastThresholdResets(t *testing.T) {
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	smap := session.NewMemMap()
	fk := &fakeKernel{}

	if err := smap.PutMetadata(context.Background(), session.Metadata{
		SessionID: "test-session",
		UpdatedAt: now.Add(-36 * time.Hour).Unix(),
	}); err != nil {
		t.Fatal(err)
	}

	m := NewManagerWithSubmitter(ManagerConfig{
		SessionMap:              smap,
		Now:                     func() time.Time { return now },
		SessionResetPolicy:      "inactivity",
		SessionResetIdleMinutes: 1440,
	}, fk, slog.Default())

	m.checkAutoReset(context.Background(), "test-session")
	if n := fk.resetsSnapshot(); n != 1 {
		t.Errorf("inactivity past threshold: resets = %d, want 1", n)
	}
}

func TestCheckAutoReset_InactivityBelowThresholdDoesNotReset(t *testing.T) {
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	smap := session.NewMemMap()
	fk := &fakeKernel{}

	if err := smap.PutMetadata(context.Background(), session.Metadata{
		SessionID: "test-session",
		UpdatedAt: now.Add(-1 * time.Hour).Unix(),
	}); err != nil {
		t.Fatal(err)
	}

	m := NewManagerWithSubmitter(ManagerConfig{
		SessionMap:              smap,
		Now:                     func() time.Time { return now },
		SessionResetPolicy:      "inactivity",
		SessionResetIdleMinutes: 1440,
	}, fk, slog.Default())

	m.checkAutoReset(context.Background(), "test-session")
	if n := fk.resetsSnapshot(); n != 0 {
		t.Errorf("inactivity below threshold: resets = %d, want 0", n)
	}
}

func TestCheckAutoReset_DailyPastBoundaryResets(t *testing.T) {
	now := time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC)
	smap := session.NewMemMap()
	fk := &fakeKernel{}

	if err := smap.PutMetadata(context.Background(), session.Metadata{
		SessionID: "test-session",
		UpdatedAt: time.Date(2026, 5, 9, 2, 0, 0, 0, time.UTC).Unix(),
	}); err != nil {
		t.Fatal(err)
	}

	m := NewManagerWithSubmitter(ManagerConfig{
		SessionMap:            smap,
		Now:                   func() time.Time { return now },
		SessionResetPolicy:    "daily",
		SessionResetDailyHour: 4,
	}, fk, slog.Default())

	m.checkAutoReset(context.Background(), "test-session")
	if n := fk.resetsSnapshot(); n != 1 {
		t.Errorf("daily past boundary: resets = %d, want 1", n)
	}
}

func TestCheckAutoReset_DailyBeforeBoundaryDoesNotReset(t *testing.T) {
	now := time.Date(2026, 5, 10, 3, 0, 0, 0, time.UTC)
	smap := session.NewMemMap()
	fk := &fakeKernel{}

	if err := smap.PutMetadata(context.Background(), session.Metadata{
		SessionID: "test-session",
		UpdatedAt: time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC).Unix(),
	}); err != nil {
		t.Fatal(err)
	}

	m := NewManagerWithSubmitter(ManagerConfig{
		SessionMap:            smap,
		Now:                   func() time.Time { return now },
		SessionResetPolicy:    "daily",
		SessionResetDailyHour: 4,
	}, fk, slog.Default())

	m.checkAutoReset(context.Background(), "test-session")
	if n := fk.resetsSnapshot(); n != 0 {
		t.Errorf("daily before boundary: resets = %d, want 0", n)
	}
}

func TestCheckAutoReset_BothInactivityResets(t *testing.T) {
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	smap := session.NewMemMap()
	fk := &fakeKernel{}

	if err := smap.PutMetadata(context.Background(), session.Metadata{
		SessionID: "test-session",
		UpdatedAt: now.Add(-36 * time.Hour).Unix(),
	}); err != nil {
		t.Fatal(err)
	}

	m := NewManagerWithSubmitter(ManagerConfig{
		SessionMap:              smap,
		Now:                     func() time.Time { return now },
		SessionResetPolicy:      "both",
		SessionResetIdleMinutes: 1440,
		SessionResetDailyHour:   4,
	}, fk, slog.Default())

	m.checkAutoReset(context.Background(), "test-session")
	if n := fk.resetsSnapshot(); n != 1 {
		t.Errorf("both/inactivity: resets = %d, want 1", n)
	}
}

func TestCheckAutoReset_MissingMetadataDoesNotReset(t *testing.T) {
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	smap := session.NewMemMap()
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		SessionMap:              smap,
		Now:                     func() time.Time { return now },
		SessionResetPolicy:      "inactivity",
		SessionResetIdleMinutes: 1440,
	}, fk, slog.Default())

	m.checkAutoReset(context.Background(), "unknown-session")
	if n := fk.resetsSnapshot(); n != 0 {
		t.Errorf("missing metadata: resets = %d, want 0", n)
	}
}
