package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestRPCModePromptEmitsHeaderAcceptedResponseAndLifecycleEvents(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	fake := &rpcModeFakeRuntime{}
	cmd := newRootCommandWithRuntime(rootRuntime{
		runResolvedTUI: func(*cobra.Command, tuiInvocation) error {
			t.Fatal("runResolvedTUI was called for --mode rpc")
			return nil
		},
		runRPC: func(cmd *cobra.Command, _ rpcInvocation) error {
			return gateway.RunRPCMode(context.Background(), gateway.RPCModeOptions{
				In:      cmd.InOrStdin(),
				Out:     cmd.OutOrStdout(),
				Runtime: fake,
			})
		},
	})
	const prompt = "hello\u2028world"
	cmd.SetIn(strings.NewReader(`{"id":"p1","type":"prompt","message":"` + prompt + `"}` + "\n"))

	stdout, stderr, err := executeRootCommandForTest(cmd, "--mode", "rpc", "--no-session")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want no startup chatter", stderr)
	}
	if len(fake.prompts) != 1 || fake.prompts[0] != "hello\u2028world" {
		t.Fatalf("prompt seen by runtime = %#v, want single LF-framed prompt with U+2028 preserved", fake.prompts)
	}

	lines := decodeRPCJSONLines(t, stdout)
	wantTypes := []string{"session", "response", "agent_start", "message_start", "tool_execution_start", "tool_execution_end", "message_update", "message_end", "agent_end"}
	if len(lines) != len(wantTypes) {
		t.Fatalf("line count = %d, want %d\nstdout=%s", len(lines), len(wantTypes), stdout)
	}
	for i, want := range wantTypes {
		if got := stringField(lines[i], "type"); got != want {
			t.Fatalf("line %d type = %q, want %q\nstdout=%s", i+1, got, want, stdout)
		}
	}
	if lines[0]["version"] != float64(1) || stringField(lines[0], "session_id") != "rpc-test-session" {
		t.Fatalf("session header = %#v", lines[0])
	}
	if stringField(lines[1], "id") != "p1" || stringField(lines[1], "command") != "prompt" || lines[1]["success"] != true {
		t.Fatalf("prompt response = %#v, want correlated success", lines[1])
	}
	if stringField(lines[4], "toolName") != "read_file" {
		t.Fatalf("tool start event = %#v", lines[4])
	}
	update := lines[6]["assistantMessageEvent"].(map[string]any)
	if update["type"] != "text_delta" || update["delta"] != "fixture answer" {
		t.Fatalf("message update = %#v", lines[6])
	}
}

func TestRPCModeStructuredStateQueueAbortAndErrors(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	fake := &rpcModeFakeRuntime{}
	cmd := newRootCommandWithRuntime(rootRuntime{
		runResolvedTUI: func(*cobra.Command, tuiInvocation) error {
			t.Fatal("runResolvedTUI was called for --mode rpc")
			return nil
		},
		runRPC: func(cmd *cobra.Command, _ rpcInvocation) error {
			return gateway.RunRPCMode(context.Background(), gateway.RPCModeOptions{
				In:      cmd.InOrStdin(),
				Out:     cmd.OutOrStdout(),
				Runtime: fake,
			})
		},
	})
	cmd.SetIn(strings.NewReader(strings.Join([]string{
		`{"id":"state-1","type":"get_state"}`,
		`{"id":"m1","type":"set_model","provider":"demo","modelId":"demo-model"}`,
		`{"id":"s1","type":"steer","message":"pivot"}`,
		`{"id":"f1","type":"follow_up","message":"after"}`,
		`{"id":"a1","type":"abort"}`,
		`{"id":"x1","type":"wat"}`,
		`{not-json}`,
	}, "\n") + "\n"))

	stdout, stderr, err := executeRootCommandForTest(cmd, "--mode", "rpc", "--no-session")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want structured stdout errors only", stderr)
	}
	lines := decodeRPCJSONLines(t, stdout)
	byID := map[string]map[string]any{}
	var queueEvents int
	for _, line := range lines {
		if stringField(line, "type") == "queue_update" {
			queueEvents++
		}
		if id := stringField(line, "id"); id != "" {
			byID[id] = line
		}
	}
	if got := byID["state-1"]; got == nil || got["success"] != true || stringField(got, "command") != "get_state" {
		t.Fatalf("get_state response = %#v", got)
	}
	if got := byID["m1"]; got == nil || got["success"] != false || !strings.Contains(stringField(got, "error"), "unsupported") {
		t.Fatalf("set_model response = %#v, want structured unsupported error", got)
	}
	for _, id := range []string{"s1", "f1", "a1"} {
		if got := byID[id]; got == nil || got["success"] != true {
			t.Fatalf("%s response = %#v, want success", id, got)
		}
	}
	if queueEvents < 2 {
		t.Fatalf("queue_update events = %d, want steer and follow_up updates\nstdout=%s", queueEvents, stdout)
	}
	if got := byID["x1"]; got == nil || got["success"] != false || !strings.Contains(stringField(got, "error"), "Unknown command") {
		t.Fatalf("unknown response = %#v", got)
	}
	var parseErr map[string]any
	for _, line := range lines {
		if stringField(line, "command") == "parse" {
			parseErr = line
			break
		}
	}
	if parseErr == nil || parseErr["success"] != false || !strings.Contains(stringField(parseErr, "error"), "Failed to parse command") {
		t.Fatalf("parse error response = %#v", parseErr)
	}
}

type rpcModeFakeRuntime struct {
	prompts []string
	steers  []string
	follows []string
	aborted bool
}

func (f *rpcModeFakeRuntime) Header(context.Context) gateway.RPCRecord {
	return gateway.RPCRecord{"type": "session", "version": 1, "session_id": "rpc-test-session", "cwd": "/tmp/gormes-rpc"}
}

func (f *rpcModeFakeRuntime) State(context.Context) (gateway.RPCRecord, error) {
	return gateway.RPCRecord{"sessionId": "rpc-test-session", "isStreaming": false, "messageCount": 0, "pendingMessageCount": len(f.steers) + len(f.follows)}, nil
}

func (f *rpcModeFakeRuntime) Messages(context.Context) ([]gateway.RPCRecord, error) { return nil, nil }

func (f *rpcModeFakeRuntime) Prompt(_ context.Context, req gateway.RPCPromptRequest) (<-chan gateway.RPCRecord, error) {
	f.prompts = append(f.prompts, req.Message)
	events := []gateway.RPCRecord{
		{"type": "agent_start"},
		{"type": "message_start", "message": gateway.RPCRecord{"role": "assistant", "content": ""}},
		{"type": "tool_execution_start", "toolCallId": "call-read", "toolName": "read_file", "args": gateway.RPCRecord{"path": "README.md"}},
		{"type": "tool_execution_end", "toolCallId": "call-read", "toolName": "read_file", "result": gateway.RPCRecord{"content": []gateway.RPCRecord{{"type": "text", "text": "fixture"}}}, "isError": false},
		{"type": "message_update", "message": gateway.RPCRecord{"role": "assistant"}, "assistantMessageEvent": gateway.RPCRecord{"type": "text_delta", "delta": "fixture answer"}},
		{"type": "message_end", "message": gateway.RPCRecord{"role": "assistant", "content": "fixture answer"}},
		{"type": "agent_end", "messages": []gateway.RPCRecord{{"role": "assistant", "content": "fixture answer"}}},
	}
	ch := make(chan gateway.RPCRecord, len(events))
	for _, ev := range events {
		ch <- ev
	}
	close(ch)
	return ch, nil
}

func (f *rpcModeFakeRuntime) Steer(_ context.Context, message string) (gateway.RPCQueueState, error) {
	f.steers = append(f.steers, message)
	return gateway.RPCQueueState{Steering: append([]string(nil), f.steers...), FollowUp: append([]string(nil), f.follows...)}, nil
}

func (f *rpcModeFakeRuntime) FollowUp(_ context.Context, message string) (gateway.RPCQueueState, error) {
	f.follows = append(f.follows, message)
	return gateway.RPCQueueState{Steering: append([]string(nil), f.steers...), FollowUp: append([]string(nil), f.follows...)}, nil
}

func (f *rpcModeFakeRuntime) Abort(context.Context) error {
	f.aborted = true
	return nil
}

func decodeRPCJSONLines(t *testing.T, text string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for i, line := range strings.Split(strings.TrimSuffix(text, "\n"), "\n") {
		if line == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(strings.TrimSuffix(line, "\r")), &rec); err != nil {
			t.Fatalf("line %d is not JSON: %q: %v\nall stdout=%s", i+1, line, err, text)
		}
		out = append(out, rec)
	}
	return out
}

func stringField(rec map[string]any, key string) string {
	if rec == nil {
		return ""
	}
	v, _ := rec[key].(string)
	return v
}
