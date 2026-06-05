package slack

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/internal/adaptertest"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

type approvalBlockCall struct {
	channelID string
	threadTS  string
	ts        string
	text      string
	blocks    []SlackBlock
	updated   bool
}

type approvalBlockClient struct {
	*mockClient

	blockMu  sync.Mutex
	blockLog []approvalBlockCall
}

func newApprovalBlockClient() *approvalBlockClient {
	return &approvalBlockClient{mockClient: newMockClient()}
}

func (m *approvalBlockClient) PostBlockMessage(ctx context.Context, channelID, threadTS, text string, blocks []SlackBlock) (string, error) {
	ts, err := m.PostMessage(ctx, channelID, threadTS, text)
	if err != nil {
		return "", err
	}
	m.blockMu.Lock()
	defer m.blockMu.Unlock()
	m.blockLog = append(m.blockLog, approvalBlockCall{
		channelID: channelID,
		threadTS:  threadTS,
		ts:        ts,
		text:      text,
		blocks:    cloneSlackBlocks(blocks),
	})
	return ts, nil
}

func (m *approvalBlockClient) UpdateBlockMessage(ctx context.Context, channelID, ts, text string, blocks []SlackBlock) error {
	if err := m.UpdateMessage(ctx, channelID, ts, text); err != nil {
		return err
	}
	m.blockMu.Lock()
	defer m.blockMu.Unlock()
	m.blockLog = append(m.blockLog, approvalBlockCall{
		channelID: channelID,
		threadTS:  m.threadByTS[ts],
		ts:        ts,
		text:      text,
		blocks:    cloneSlackBlocks(blocks),
		updated:   true,
	})
	return nil
}

func (m *approvalBlockClient) blockOutputs() []approvalBlockCall {
	m.blockMu.Lock()
	defer m.blockMu.Unlock()
	out := make([]approvalBlockCall, len(m.blockLog))
	copy(out, m.blockLog)
	return out
}

func TestSlackApprovalButtons_PostsBlockKitPrompt(t *testing.T) {
	client := newApprovalBlockClient()
	resolver := &adaptertest.ApprovalRecorder{}
	b := New(Config{
		AllowedChannelID: "C123",
		ApprovalResolver: resolver,
	}, client, newIdleSlackKernel(), nil)

	msgTS, err := b.SendExecApproval(context.Background(), ApprovalPrompt{
		ChannelID:   "C123",
		ThreadTS:    "1711111111.000100",
		Command:     "rm -rf /tmp/project",
		SessionKey:  "slack:C123:sess-1",
		Description: "dangerous command",
	})
	if err != nil {
		t.Fatalf("SendExecApproval: %v", err)
	}
	if msgTS == "" {
		t.Fatal("message timestamp empty")
	}

	outputs := client.blockOutputs()
	if len(outputs) != 1 {
		t.Fatalf("block outputs = %d, want 1", len(outputs))
	}
	got := outputs[0]
	if got.channelID != "C123" || got.threadTS != "1711111111.000100" {
		t.Fatalf("prompt route = %+v, want channel C123 thread 1711111111.000100", got)
	}
	if !strings.Contains(got.text, "Command approval required") || !strings.Contains(got.text, "rm -rf") {
		t.Fatalf("fallback text = %q, want command approval preview", got.text)
	}

	section := approvalSectionText(t, got.blocks)
	if !strings.Contains(section, "*Command Approval Required*") {
		t.Fatalf("section text = %q, want approval heading", section)
	}
	if !strings.Contains(section, "```rm -rf /tmp/project```") {
		t.Fatalf("section text = %q, want command code block", section)
	}
	if !strings.Contains(section, "Reason: dangerous command") {
		t.Fatalf("section text = %q, want description", section)
	}

	buttons := approvalButtons(t, got.blocks)
	want := []struct {
		label    string
		actionID string
		style    string
	}{
		{label: "Allow Once", actionID: "hermes_approve_once", style: "primary"},
		{label: "Allow Session", actionID: "hermes_approve_session"},
		{label: "Always Allow", actionID: "hermes_approve_always"},
		{label: "Deny", actionID: "hermes_deny", style: "danger"},
	}
	if len(buttons) != len(want) {
		t.Fatalf("buttons = %d, want %d: %#v", len(buttons), len(want), buttons)
	}
	for i, w := range want {
		if label := slackTextObjectText(buttons[i]["text"]); label != w.label {
			t.Fatalf("button %d label = %q, want %q", i, label, w.label)
		}
		if buttons[i]["action_id"] != w.actionID {
			t.Fatalf("button %d action_id = %q, want %q", i, buttons[i]["action_id"], w.actionID)
		}
		if buttons[i]["value"] != "slack:C123:sess-1" {
			t.Fatalf("button %d value = %q, want session key", i, buttons[i]["value"])
		}
		style, _ := buttons[i]["style"].(string)
		if style != w.style {
			t.Fatalf("button %d style = %q, want %q", i, style, w.style)
		}
	}
}

func TestSlackApprovalButtons_TruncatesLongCommandWithinSlackLimits(t *testing.T) {
	client := newApprovalBlockClient()
	b := New(Config{
		AllowedChannelID: "C123",
		ApprovalResolver: &adaptertest.ApprovalRecorder{},
	}, client, newIdleSlackKernel(), nil)

	_, err := b.SendExecApproval(context.Background(), ApprovalPrompt{
		ChannelID:   "C123",
		ThreadTS:    "1711111111.000200",
		Command:     strings.Repeat("x", 5000),
		SessionKey:  "slack:C123:sess-2",
		Description: strings.Repeat("needs review ", 300),
	})
	if err != nil {
		t.Fatalf("SendExecApproval: %v", err)
	}

	outputs := client.blockOutputs()
	section := approvalSectionText(t, outputs[0].blocks)
	if len([]rune(section)) > slackApprovalSectionLimit {
		t.Fatalf("section length = %d, want <= %d", len([]rune(section)), slackApprovalSectionLimit)
	}
	if !strings.Contains(section, "...") {
		t.Fatalf("section text = %q, want truncation marker", section)
	}
	if buttons := approvalButtons(t, outputs[0].blocks); len(buttons) != 4 {
		t.Fatalf("buttons = %d, want 4 after truncation", len(buttons))
	}
}

func TestSlackApprovalButtons_UnavailableWhenSlackOrApprovalStorageMissing(t *testing.T) {
	resolver := &adaptertest.ApprovalRecorder{}
	b := New(Config{AllowedChannelID: "C123", ApprovalResolver: resolver}, nil, newIdleSlackKernel(), nil)
	_, err := b.SendExecApproval(context.Background(), ApprovalPrompt{
		ChannelID:  "C123",
		Command:    "rm -rf /tmp/project",
		SessionKey: "slack:C123:sess-unavailable",
	})
	if !errors.Is(err, ErrSlackApprovalUnavailable) {
		t.Fatalf("nil client err = %v, want ErrSlackApprovalUnavailable", err)
	}

	client := newApprovalBlockClient()
	b = New(Config{AllowedChannelID: "C123"}, client, newIdleSlackKernel(), nil)
	_, err = b.SendExecApproval(context.Background(), ApprovalPrompt{
		ChannelID:  "C123",
		Command:    "rm -rf /tmp/project",
		SessionKey: "slack:C123:sess-no-store",
	})
	if !errors.Is(err, ErrSlackApprovalUnavailable) {
		t.Fatalf("missing resolver err = %v, want ErrSlackApprovalUnavailable", err)
	}

	b = New(Config{AllowedChannelID: "C123", ApprovalResolver: resolver}, client, newIdleSlackKernel(), nil)
	_, err = b.SendExecApproval(context.Background(), ApprovalPrompt{
		ChannelID:  "C999",
		Command:    "rm -rf /tmp/project",
		SessionKey: "slack:C999:sess-blocked",
	})
	if !errors.Is(err, ErrSlackApprovalUnavailable) {
		t.Fatalf("unregistered channel err = %v, want ErrSlackApprovalUnavailable", err)
	}
	if outputs := client.blockOutputs(); len(outputs) != 0 {
		t.Fatalf("block outputs = %+v, want no approval prompt when unavailable", outputs)
	}
	if calls := resolver.Snapshot(); len(calls) != 0 {
		t.Fatalf("resolver calls = %+v, want no command approval resolution on unavailable prompt", calls)
	}
}

func TestSlackApprovalCallback_ResolvesOnceAndUpdatesMessage(t *testing.T) {
	client := newApprovalBlockClient()
	resolver := &adaptertest.ApprovalRecorder{}
	b := New(Config{
		AllowedChannelID: "C123",
		ApprovalResolver: resolver,
	}, client, newIdleSlackKernel(), nil)
	msgTS := sendSlackApprovalPrompt(t, b, "slack:C123:sess-once")

	b.handleEvent(context.Background(), Event{
		RequestID: "callback-once-1",
		ChannelID: "C123",
		UserID:    "U42",
		ApprovalAction: &ApprovalAction{
			ActionID:   "hermes_approve_once",
			SessionKey: "slack:C123:sess-once",
			MessageTS:  msgTS,
			ChannelID:  "C123",
			UserID:     "U42",
			UserName:   "ada",
		},
	})

	if !client.wasAcked("callback-once-1") {
		t.Fatal("callback was not acked")
	}
	calls := client.calls()
	ackIndex := indexOfCall(calls, "ack:callback-once-1")
	updateIndex := indexOfCall(calls, "update:"+msgTS)
	if ackIndex == -1 || updateIndex == -1 || ackIndex > updateIndex {
		t.Fatalf("calls = %v, want ack before update", calls)
	}
	resolved := resolver.Snapshot()
	if len(resolved) != 1 {
		t.Fatalf("resolver calls = %+v, want one", resolved)
	}
	if resolved[0].SessionKey != "slack:C123:sess-once" || resolved[0].Choice != gateway.ApprovalChoiceOnce {
		t.Fatalf("resolution = %+v, want once for session key", resolved[0])
	}
	if resolved[0].Platform != "slack" || resolved[0].ChatID != "C123" || resolved[0].MessageID != msgTS || resolved[0].ActorID != "U42" {
		t.Fatalf("resolution metadata = %+v, want redacted Slack channel evidence", resolved[0])
	}

	update := lastUpdatedBlockCall(t, client)
	if !strings.Contains(update.text, "Approved once by ada") {
		t.Fatalf("update text = %q, want actor decision", update.text)
	}
	if len(update.blocks) != 2 {
		t.Fatalf("updated blocks = %#v, want original section plus context", update.blocks)
	}
	if _, ok := findApprovalBlock(update.blocks, "actions"); ok {
		t.Fatalf("updated blocks kept decision buttons: %#v", update.blocks)
	}
}

func TestSlackApprovalCallback_DoubleClickAckedWithoutSecondResolution(t *testing.T) {
	client := newApprovalBlockClient()
	resolver := &adaptertest.ApprovalRecorder{}
	b := New(Config{
		AllowedChannelID: "C123",
		ApprovalResolver: resolver,
	}, client, newIdleSlackKernel(), nil)
	msgTS := sendSlackApprovalPrompt(t, b, "slack:C123:sess-double")

	click := func(requestID string) {
		b.handleEvent(context.Background(), Event{
			RequestID: requestID,
			ChannelID: "C123",
			UserID:    "U42",
			ApprovalAction: &ApprovalAction{
				ActionID:   "hermes_approve_session",
				SessionKey: "slack:C123:sess-double",
				MessageTS:  msgTS,
				ChannelID:  "C123",
				UserID:     "U42",
				UserName:   "ada",
			},
		})
	}
	click("callback-double-1")
	click("callback-double-2")

	if !client.wasAcked("callback-double-2") {
		t.Fatal("second callback was not acked")
	}
	if got := len(resolver.Snapshot()); got != 1 {
		t.Fatalf("resolver calls = %d, want 1", got)
	}
	updates := 0
	for _, call := range client.blockOutputs() {
		if call.updated {
			updates++
		}
	}
	if updates != 1 {
		t.Fatalf("updates = %d, want one resolved-message update", updates)
	}
}

func TestSlackApprovalCallback_DenyMapsToGatewayChoice(t *testing.T) {
	client := newApprovalBlockClient()
	resolver := &adaptertest.ApprovalRecorder{}
	b := New(Config{
		AllowedChannelID: "C123",
		ApprovalResolver: resolver,
	}, client, newIdleSlackKernel(), nil)
	msgTS := sendSlackApprovalPrompt(t, b, "slack:C123:sess-deny")

	b.handleEvent(context.Background(), Event{
		RequestID: "callback-deny-1",
		ChannelID: "C123",
		UserID:    "U42",
		ApprovalAction: &ApprovalAction{
			ActionID:   "hermes_deny",
			SessionKey: "slack:C123:sess-deny",
			MessageTS:  msgTS,
			ChannelID:  "C123",
			UserID:     "U42",
			UserName:   "ada",
		},
	})

	resolved := resolver.Snapshot()
	if len(resolved) != 1 {
		t.Fatalf("resolver calls = %+v, want one deny", resolved)
	}
	if resolved[0].Choice != gateway.ApprovalChoiceDeny {
		t.Fatalf("choice = %q, want deny", resolved[0].Choice)
	}
	update := lastUpdatedBlockCall(t, client)
	if !strings.Contains(update.text, "Denied by ada") {
		t.Fatalf("update text = %q, want denied actor message", update.text)
	}
}

func sendSlackApprovalPrompt(t *testing.T, b *Bot, sessionKey string) string {
	t.Helper()
	msgTS, err := b.SendExecApproval(context.Background(), ApprovalPrompt{
		ChannelID:   "C123",
		ThreadTS:    "1711111111.000300",
		Command:     "rm -rf /tmp/project",
		SessionKey:  sessionKey,
		Description: "dangerous command",
	})
	if err != nil {
		t.Fatalf("SendExecApproval: %v", err)
	}
	return msgTS
}

func lastUpdatedBlockCall(t *testing.T, client *approvalBlockClient) approvalBlockCall {
	t.Helper()
	var got approvalBlockCall
	found := false
	for _, call := range client.blockOutputs() {
		if call.updated {
			got = call
			found = true
		}
	}
	if !found {
		t.Fatal("no updated block call found")
	}
	return got
}

func approvalSectionText(t *testing.T, blocks []SlackBlock) string {
	t.Helper()
	block, ok := findApprovalBlock(blocks, "section")
	if !ok {
		t.Fatalf("blocks = %#v, want section block", blocks)
	}
	return slackTextObjectText(block["text"])
}

func approvalButtons(t *testing.T, blocks []SlackBlock) []SlackBlock {
	t.Helper()
	block, ok := findApprovalBlock(blocks, "actions")
	if !ok {
		t.Fatalf("blocks = %#v, want actions block", blocks)
	}
	elements, ok := block["elements"].([]SlackBlock)
	if !ok {
		t.Fatalf("actions elements = %#v, want []SlackBlock", block["elements"])
	}
	return elements
}

func findApprovalBlock(blocks []SlackBlock, blockType string) (SlackBlock, bool) {
	for _, block := range blocks {
		if block["type"] == blockType {
			return block, true
		}
	}
	return nil, false
}

func slackTextObjectText(value any) string {
	switch typed := value.(type) {
	case SlackBlock:
		text, _ := typed["text"].(string)
		return text
	case map[string]any:
		text, _ := typed["text"].(string)
		return text
	default:
		return ""
	}
}

func indexOfCall(calls []string, want string) int {
	for i, call := range calls {
		if call == want {
			return i
		}
	}
	return -1
}
