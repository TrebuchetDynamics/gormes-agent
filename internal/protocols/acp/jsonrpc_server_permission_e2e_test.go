package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"
)

type acpE2EClient struct {
	t        *testing.T
	enc      *json.Encoder
	dec      *bufio.Scanner
	nextID   int
	updates  []map[string]any
	requests []map[string]any
	answer   string
	noAnswer bool
}

func (c *acpE2EClient) sendResponse(id any, result map[string]any) {
	c.t.Helper()
	if err := c.enc.Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": result}); err != nil {
		c.t.Fatalf("write client response: %v", err)
	}
}

func (c *acpE2EClient) request(method string, params map[string]any) map[string]any {
	c.t.Helper()
	c.nextID++
	wantID := strconv.Itoa(c.nextID)
	if err := c.enc.Encode(map[string]any{"jsonrpc": "2.0", "id": c.nextID, "method": method, "params": params}); err != nil {
		c.t.Fatalf("write %s: %v", method, err)
	}
	for c.dec.Scan() {
		line := strings.TrimSpace(c.dec.Text())
		if line == "" {
			continue
		}
		var frame map[string]any
		if err := json.Unmarshal([]byte(line), &frame); err != nil {
			c.t.Fatalf("decode frame %q: %v", line, err)
		}
		switch frame["method"] {
		case "session/update":
			params, _ := frame["params"].(map[string]any)
			update, _ := params["update"].(map[string]any)
			c.updates = append(c.updates, update)
			continue
		case "session/request_permission":
			c.requests = append(c.requests, frame)
			if c.noAnswer {
				continue
			}
			c.sendResponse(frame["id"], map[string]any{
				"outcome": map[string]any{"outcome": "selected", "optionId": c.answer},
			})
			continue
		}
		if jsonRPCIDKey(frame["id"]) == wantID {
			if rpcErr, ok := frame["error"]; ok {
				c.t.Fatalf("%s error: %#v", method, rpcErr)
			}
			result, _ := frame["result"].(map[string]any)
			return result
		}
	}
	c.t.Fatalf("%s: server closed before response", method)
	return nil
}

func boolPtr(value bool) *bool { return &value }

func TestACPEndToEndPermissionRoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		answer     string
		wantResult string
		wantStatus string
	}{
		{name: "allow once", answer: "allow_once", wantResult: "once", wantStatus: "completed"},
		{name: "deny", answer: "deny", wantResult: "deny", wantStatus: "failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runACPEndToEnd(t, tt.answer, false, tt.wantResult, tt.wantStatus)
		})
	}
}

func TestACPEndToEndPermissionTimeoutFailsClosed(t *testing.T) {
	runACPEndToEnd(t, "", true, "deny", "failed")
}

func runACPEndToEnd(t *testing.T, answer string, noAnswer bool, wantResult, wantStatus string) {
	t.Helper()
	decisionCh := make(chan PermissionDecision, 1)
	var runtime *SessionRuntime
	runner := PromptRunnerFunc(func(ctx context.Context, req RuntimePromptRequest, emit func(PromptEvent)) (PromptResult, error) {
		emit(PromptEvent{Kind: PromptEventAgentThoughtChunk, Text: "thinking"})
		emit(PromptEvent{Kind: PromptEventAgentMessageChunk, Text: "hello"})
		emit(PromptEvent{Kind: PromptEventToolCall, ToolCall: &ToolCallEvent{
			ToolCallID: "tc-1",
			Title:      "Run echo",
			Kind:       ToolKindExecute,
			Status:     "pending",
			Content:    []ToolCallContent{TextToolCallContent("$ echo hi")},
			RawInput:   map[string]any{"command": "echo hi"},
		}})
		toolCall := NewPermissionToolCall("echo hi", "run command")
		decision, err := runtime.PermissionBroker().Request(ctx, req.SessionID, PermissionRequest{
			Key:         "terminal:echo hi",
			Command:     "echo hi",
			Description: "run command",
			SessionID:   req.SessionID,
			ToolCall:    &toolCall,
			Options:     BuildPermissionOptions(true, true, false),
		})
		if err != nil {
			return PromptResult{}, err
		}
		decisionCh <- decision
		emit(PromptEvent{Kind: PromptEventToolCallUpdate, ToolCall: &ToolCallEvent{
			ToolCallID: "tc-1",
			Status:     wantStatus,
			Content:    []ToolCallContent{TextToolCallContent(decision.Result)},
		}})
		return PromptResult{Final: "done", StopReason: "end_turn"}, nil
	})
	runtime = NewSessionRuntime(SessionRuntimeConfig{
		Provider:            "openrouter",
		Model:               "gpt-test",
		Runner:              runner,
		CredentialAvailable: boolPtr(true),
		IDGenerator:         func() string { return "e2e-session" },
	})

	serverIn, clientOut := io.Pipe()
	clientIn, serverOut := io.Pipe()
	server := NewJSONRPCServer(runtime)
	server.PermissionTimeout = 200 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- server.Handle(ctx, serverIn, serverOut)
		_ = serverOut.Close()
	}()

	client := &acpE2EClient{
		t:        t,
		enc:      json.NewEncoder(clientOut),
		dec:      bufio.NewScanner(clientIn),
		answer:   answer,
		noAnswer: noAnswer,
	}
	client.dec.Buffer(make([]byte, 64*1024), 1024*1024)
	defer func() {
		_ = clientOut.Close()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("ACP server did not stop after client EOF")
		}
		_ = clientIn.Close()
	}()

	init := client.request("initialize", map[string]any{"protocolVersion": 1, "clientInfo": map[string]any{"name": "e2e"}})
	authMethods, _ := init["authMethods"].([]any)
	if len(authMethods) != 2 {
		t.Fatalf("authMethods = %#v, want provider plus terminal", authMethods)
	}
	auth := client.request("authenticate", map[string]any{"methodId": "openrouter"})
	if auth["authenticated"] != true {
		t.Fatalf("authenticate = %#v", auth)
	}
	session := client.request("session/new", map[string]any{"cwd": "/repo", "mcpServers": []any{}})
	if session["sessionId"] != "e2e-session" {
		t.Fatalf("session = %#v", session)
	}
	result := client.request("session/prompt", map[string]any{
		"sessionId": "e2e-session",
		"prompt":    []any{map[string]any{"type": "text", "text": "run echo"}},
	})
	if result["stopReason"] != "end_turn" {
		t.Fatalf("prompt result = %#v", result)
	}

	if len(client.requests) != 1 {
		t.Fatalf("permission requests = %#v, want one", client.requests)
	}
	params, _ := client.requests[0]["params"].(map[string]any)
	if params["sessionId"] != "e2e-session" {
		t.Fatalf("permission session = %#v", params["sessionId"])
	}
	toolCall, _ := params["toolCall"].(map[string]any)
	if toolCall["status"] != "pending" || toolCall["kind"] != "execute" {
		t.Fatalf("permission toolCall = %#v", toolCall)
	}
	content, _ := toolCall["content"].([]any)
	first := content[0].(map[string]any)
	inner, _ := first["content"].(map[string]any)
	if len(content) != 1 || first["type"] != "content" || !strings.Contains(inner["text"].(string), "$ echo hi") {
		t.Fatalf("permission content = %#v", content)
	}
	options, _ := params["options"].([]any)
	var optionIDs []string
	for _, option := range options {
		optionIDs = append(optionIDs, option.(map[string]any)["optionId"].(string))
	}
	if strings.Join(optionIDs, ",") != "allow_once,allow_session,allow_always,deny,deny_always" {
		t.Fatalf("options = %v", optionIDs)
	}

	select {
	case decision := <-decisionCh:
		if decision.Result != wantResult {
			t.Fatalf("decision = %+v, want result %q", decision, wantResult)
		}
		if noAnswer && decision.Reason != "permission_timeout" {
			t.Fatalf("timeout decision reason = %q", decision.Reason)
		}
	case <-time.After(time.Second):
		t.Fatal("runner did not record permission decision")
	}

	updates := map[string]map[string]any{}
	for _, update := range client.updates {
		kind, _ := update["sessionUpdate"].(string)
		updates[kind] = update
	}
	thought := updates["agent_thought_chunk"]
	if thought["content"].(map[string]any)["text"] != "thinking" {
		t.Fatalf("thought update = %#v", thought)
	}
	message := updates["agent_message_chunk"]
	if message["content"].(map[string]any)["text"] != "hello" {
		t.Fatalf("message update = %#v", message)
	}
	start := updates["tool_call"]
	if start["toolCallId"] != "tc-1" || start["title"] != "Run echo" || start["kind"] != "execute" {
		t.Fatalf("tool_call update = %#v", start)
	}
	complete := updates["tool_call_update"]
	if complete["toolCallId"] != "tc-1" || complete["status"] != wantStatus {
		t.Fatalf("tool_call_update = %#v", complete)
	}
}
