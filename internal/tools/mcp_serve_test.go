package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestMCPServer_Initialize(t *testing.T) {
	s := &MCPServer{}
	s.RegisterTool("test_tool", "A test tool", map[string]interface{}{}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{"ok": true}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create two pipes for bidirectional communication
	serverR, clientW := net.Pipe()
	clientR, serverW := net.Pipe()
	defer serverR.Close()
	defer serverW.Close()
	defer clientR.Close()
	defer clientW.Close()

	// Start server
	go func() {
		s.serve(ctx, serverR, serverW)
	}()

	// Client sends initialize request
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
			"capabilities": map[string]interface{}{},
		},
	}
	b, _ := json.Marshal(initReq)
	req := append(b, '\n')
	clientW.Write(req)

	// Client reads response
	bufR := bufio.NewReader(clientR)
	line, err := bufR.ReadBytes('\n')
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var resp mcpJSONRPCResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("initialize error: %s", resp.Error.Message)
	}
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map, got %T", resp.Result)
	}
	if pv, ok := result["protocolVersion"].(string); !ok || pv != "2024-11-05" {
		t.Errorf("expected protocol version 2024-11-05, got %v", result["protocolVersion"])
	}
}

func TestMCPServer_ToolsList(t *testing.T) {
	s := &MCPServer{}
	s.RegisterTool("my_tool", "A test tool", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
		},
	}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{"ok": true}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverR, clientW := net.Pipe()
	clientR, serverW := net.Pipe()
	defer serverR.Close()
	defer serverW.Close()
	defer clientR.Close()
	defer clientW.Close()

	go func() {
		s.serve(ctx, serverR, serverW)
	}()

	toolsReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}
	b, _ := json.Marshal(toolsReq)
	clientW.Write(append(b, '\n'))

	bufR := bufio.NewReader(clientR)
	line, err := bufR.ReadBytes('\n')
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var resp mcpJSONRPCResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("tools/list error: code=%d msg=%s", resp.Error.Code, resp.Error.Message)
	}
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map, got %T", resp.Result)
	}
	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatalf("expected tools array, got %T", result["tools"])
	}
	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}
}

func TestMCPServer_ToolsCall(t *testing.T) {
	s := &MCPServer{}
	s.RegisterTool("echo", "Echo back the input", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"value": map[string]interface{}{"type": "string"},
		},
	}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{"echoed": args["value"]}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverR, clientW := net.Pipe()
	clientR, serverW := net.Pipe()
	defer serverR.Close()
	defer serverW.Close()
	defer clientR.Close()
	defer clientW.Close()

	go func() {
		s.serve(ctx, serverR, serverW)
	}()

	callReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "echo",
			"arguments": map[string]interface{}{
				"value": "hello world",
			},
		},
	}
	b, _ := json.Marshal(callReq)
	clientW.Write(append(b, '\n'))

	bufR := bufio.NewReader(clientR)
	line, err := bufR.ReadBytes('\n')
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var resp mcpJSONRPCResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("tools/call error: code=%d msg=%s", resp.Error.Code, resp.Error.Message)
	}
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map, got %T", resp.Result)
	}
	content, ok := result["content"].([]interface{})
	if !ok {
		t.Fatalf("expected content array, got %T", result["content"])
	}
	if len(content) == 0 {
		t.Fatal("expected at least one content item")
	}
}

func TestMCPServer_UnknownMethod(t *testing.T) {
	s := &MCPServer{}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverR, clientW := net.Pipe()
	clientR, serverW := net.Pipe()
	defer serverR.Close()
	defer serverW.Close()
	defer clientR.Close()
	defer clientW.Close()

	go func() {
		s.serve(ctx, serverR, serverW)
	}()

	unknownReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "unknown/method",
		"params":  map[string]interface{}{},
	}
	b, _ := json.Marshal(unknownReq)
	clientW.Write(append(b, '\n'))

	bufR := bufio.NewReader(clientR)
	line, err := bufR.ReadBytes('\n')
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var resp mcpJSONRPCResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != -32603 {
		t.Errorf("expected error code -32603, got %d", resp.Error.Code)
	}
}

func TestMCPServer_RegisterDefaultTools(t *testing.T) {
	s := &MCPServer{}
	s.RegisterDefaultTools()

	if len(s.tools) != 6 {
		t.Errorf("expected 6 default tools, got %d", len(s.tools))
	}

	expectedTools := []string{"conversations_list", "messages_list", "messages_get", "tools_list", "sessions_list", "channels_list"}
	for _, name := range expectedTools {
		if _, ok := s.tools[name]; !ok {
			t.Errorf("expected tool %q to be registered", name)
		}
	}
}

func TestFormatResult(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"map", map[string]interface{}{"key": "value"}, `{"key":"value"}`},
		{"slice", []int{1, 2, 3}, `[1,2,3]`},
		{"string", "hello", `"hello"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatResult(tt.input)
			if got != tt.expected {
				t.Errorf("formatResult() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestToolDescription(t *testing.T) {
	if d := toolDescription("conversations_list"); d != "List all conversations/sessions" {
		t.Errorf("expected 'List all conversations/sessions', got %q", d)
	}
	if d := toolDescription("unknown_tool"); d != "" {
		t.Errorf("expected empty string for unknown tool, got %q", d)
	}
}

func TestToolInputSchema(t *testing.T) {
	schema := toolInputSchema("messages_list")
	if schema["type"] != "object" {
		t.Errorf("expected type 'object', got %v", schema["type"])
	}
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties to be map")
	}
	if _, ok := props["session_key"]; !ok {
		t.Error("expected session_key property")
	}
}

func TestMCPServer_HandleToolsCall_ToolNotFound(t *testing.T) {
	s := &MCPServer{}

	params := map[string]interface{}{
		"name":      "nonexistent",
		"arguments": map[string]interface{}{},
	}
	b, _ := json.Marshal(params)

	result, err := s.handleToolsCall(context.Background(), b)
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}
	if result != nil {
		t.Error("expected nil result for nonexistent tool")
	}
}
