package acp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

func TestACPJSONRPCInitializeAuthenticateAndImageCapability(t *testing.T) {
	runtime := NewSessionRuntime(SessionRuntimeConfig{
		Provider: "openrouter",
		Model:    "gpt-5.4",
	})
	var out bytes.Buffer
	err := NewJSONRPCServer(runtime).Handle(context.Background(), strings.NewReader(strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1,"clientInfo":{"name":"zed"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"authenticate","params":{"methodId":"OpenRouter"}}`,
		`{"jsonrpc":"2.0","id":3,"method":"authenticate","params":{"methodId":"totally-invalid-method"}}`,
	}, "\n")+"\n"), &out)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	frames := decodeJSONRPCFrames(t, out.String())
	if len(frames) != 3 {
		t.Fatalf("frames = %d, want 3\n%s", len(frames), out.String())
	}
	initResult := resultMap(t, frames[0])
	if got := nestedBool(initResult, "agentCapabilities", "promptCapabilities", "image"); !got {
		t.Fatalf("initialize prompt image capability = false, result=%v", initResult)
	}
	if got := nestedBool(initResult, "agentCapabilities", "loadSession"); !got {
		t.Fatalf("initialize loadSession = false, result=%v", initResult)
	}
	authMethods := initResult["authMethods"].([]any)
	if len(authMethods) != 1 || authMethods[0].(map[string]any)["id"] != "openrouter" {
		t.Fatalf("authMethods = %#v, want openrouter", authMethods)
	}
	if resultMap(t, frames[1])["authenticated"] != true {
		t.Fatalf("matching authenticate result = %#v", frames[1]["result"])
	}
	if _, ok := frames[2]["result"]; !ok || frames[2]["result"] != nil {
		t.Fatalf("mismatched authenticate result = %#v, want explicit null", frames[2])
	}
}

func TestACPJSONRPCSessionLifecycle(t *testing.T) {
	smap := session.NewMemMap()
	runtime := NewSessionRuntime(SessionRuntimeConfig{
		Provider:   "openrouter",
		Model:      "gpt-5.4",
		SessionMap: smap,
		IDGenerator: func() string {
			return "session-one"
		},
	})
	var out bytes.Buffer
	err := NewJSONRPCServer(runtime).Handle(context.Background(), strings.NewReader(strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"E:\\Projects\\Paperclip"}}`,
		`{"jsonrpc":"2.0","id":2,"method":"session/load","params":{"sessionId":"session-one","cwd":"/tmp/load"}}`,
		`{"jsonrpc":"2.0","id":3,"method":"session/resume","params":{"sessionId":"restored","cwd":"/repo"}}`,
		`{"jsonrpc":"2.0","id":4,"method":"session/cancel","params":{"sessionId":"session-one"}}`,
	}, "\n")+"\n"), &out)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	frames := decodeJSONRPCFrames(t, out.String())
	newResult := resultMap(t, frames[0])
	if newResult["sessionId"] != "session-one" {
		t.Fatalf("new session id = %#v, want session-one", newResult["sessionId"])
	}
	if newResult["cwd"] != "/mnt/e/Projects/Paperclip" {
		t.Fatalf("translated cwd = %#v", newResult["cwd"])
	}
	if got, err := smap.Get(context.Background(), "acp:session-one"); err != nil || got != "session-one" {
		t.Fatalf("persisted ACP session = %q, %v; want session-one", got, err)
	}
	loadResult := resultMap(t, frames[1])
	if loadResult["sessionId"] != "session-one" || loadResult["cwd"] != "/tmp/load" {
		t.Fatalf("load result = %#v, want session-one /tmp/load", loadResult)
	}
	if frames[2]["result"] != nil {
		t.Fatalf("resume non-ACP session result = %#v, want null", frames[2]["result"])
	}
	cancelResult := resultMap(t, frames[3])
	if cancelResult["cancelled"] != true {
		t.Fatalf("cancel result = %#v, want cancelled", cancelResult)
	}
	assertOnlyJSONRPCFrames(t, out.String())
}

func TestACPJSONRPCPromptRunCloseout(t *testing.T) {
	runtime := NewSessionRuntime(SessionRuntimeConfig{
		Provider: "openrouter",
		Model:    "gpt-5.4",
		IDGenerator: func() string {
			return "prompt-session"
		},
		Runner: PromptRunnerFunc(func(ctx context.Context, req RuntimePromptRequest, emit func(PromptEvent)) (PromptResult, error) {
			if req.Text != "say hi" {
				t.Fatalf("runner text = %q, want say hi", req.Text)
			}
			emit(PromptEvent{Kind: PromptEventAgentMessageChunk, Text: "hello"})
			return PromptResult{
				Final:      "hello",
				StopReason: "end_turn",
				Title:      "Greeting",
				Usage:      &ACPUsage{InputTokens: 3, OutputTokens: 2, TotalTokens: 5},
			}, nil
		}),
	})
	sess, err := runtime.NewSession(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	var out bytes.Buffer
	err = NewJSONRPCServer(runtime).Handle(context.Background(), strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"session/prompt","params":{"sessionId":"`+sess.ID+`","prompt":[{"type":"text","text":"say hi"}]}}`+"\n",
	), &out)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	frames := decodeJSONRPCFrames(t, out.String())
	if got := countSessionUpdates(frames, "agent_message_chunk", "hello"); got != 1 {
		t.Fatalf("agent_message_chunk hello count = %d, want 1\n%s", got, out.String())
	}
	if got := countSessionUpdates(frames, "session_title_update", "Greeting"); got != 1 {
		t.Fatalf("title update count = %d, want 1\n%s", got, out.String())
	}
	if got := countSessionUpdates(frames, "usage_update", ""); got != 1 {
		t.Fatalf("usage update count = %d, want 1\n%s", got, out.String())
	}
	result := resultMap(t, frames[len(frames)-1])
	if result["stopReason"] != "end_turn" {
		t.Fatalf("prompt stopReason = %#v, want end_turn", result["stopReason"])
	}
}

func decodeJSONRPCFrames(t *testing.T, raw string) []map[string]any {
	t.Helper()
	scanner := bufio.NewScanner(strings.NewReader(raw))
	var frames []map[string]any
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var frame map[string]any
		if err := json.Unmarshal([]byte(line), &frame); err != nil {
			t.Fatalf("non-JSON-RPC line %q: %v", line, err)
		}
		frames = append(frames, frame)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan frames: %v", err)
	}
	return frames
}

func assertOnlyJSONRPCFrames(t *testing.T, raw string) {
	t.Helper()
	for _, frame := range decodeJSONRPCFrames(t, raw) {
		if frame["jsonrpc"] != "2.0" {
			t.Fatalf("frame missing jsonrpc marker: %#v", frame)
		}
	}
}

func resultMap(t *testing.T, frame map[string]any) map[string]any {
	t.Helper()
	result, ok := frame["result"].(map[string]any)
	if !ok {
		t.Fatalf("frame result = %#v, want object; frame=%#v", frame["result"], frame)
	}
	return result
}

func nestedBool(root map[string]any, path ...string) bool {
	var cur any = root
	for _, elem := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return false
		}
		cur = m[elem]
	}
	got, _ := cur.(bool)
	return got
}

func countSessionUpdates(frames []map[string]any, kind, text string) int {
	count := 0
	for _, frame := range frames {
		if frame["method"] != "session/update" {
			continue
		}
		params, _ := frame["params"].(map[string]any)
		update, _ := params["update"].(map[string]any)
		if update["sessionUpdate"] != kind {
			continue
		}
		if text != "" && update["text"] != text && update["title"] != text {
			continue
		}
		count++
	}
	return count
}
