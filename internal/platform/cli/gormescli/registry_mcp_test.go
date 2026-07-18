package gormescli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestBuildDefaultRegistryRegistersConfiguredHTTPMCPTool(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "fixture", Version: "1"}, nil)
	var called atomic.Int64
	server.AddTool(&mcpsdk.Tool{
		Name:        "echo",
		Description: "Echo through MCP",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`),
	}, func(_ context.Context, request *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		var input struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(request.Params.Arguments, &input); err != nil {
			return nil, err
		}
		called.Add(1)
		return &mcpsdk.CallToolResult{Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: "echo:" + input.Text}}}, nil
	})
	handler := mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server { return server }, &mcpsdk.StreamableHTTPOptions{JSONResponse: true})
	var deletes atomic.Int64
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer private-config-secret" {
			t.Errorf("missing configured Authorization header")
		}
		if request.Method == http.MethodDelete {
			deletes.Add(1)
		}
		handler.ServeHTTP(w, request)
	})
	httpServer := httptest.NewServer(wrapped)
	defer httpServer.Close()

	reg := BuildDefaultRegistry(context.Background(), config.Config{MCPServers: map[string]any{
		"fixture": map[string]any{
			"url":     httpServer.URL,
			"headers": map[string]any{"Authorization": "Bearer private-config-secret"},
			"enabled": true,
		},
	}}, nil, "")
	tool, ok := reg.Get("mcp__fixture__echo")
	if !ok {
		t.Fatal("configured MCP tool not registered in default registry")
	}
	var schema map[string]any
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("descriptor schema invalid: %v", err)
	}
	properties, _ := schema["properties"].(map[string]any)
	if tool.Description() != "Echo through MCP" || schema["type"] != "object" || properties["text"] == nil {
		t.Fatalf("descriptor description=%q schema=%s", tool.Description(), tool.Schema())
	}

	executor := tools.NewInProcessToolExecutor(reg)
	events, err := executor.Execute(context.Background(), tools.ToolRequest{
		ToolName: "mcp__fixture__echo",
		Input:    json.RawMessage(`{"text":"hello"}`),
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	var output json.RawMessage
	for event := range events {
		if event.Err != nil {
			t.Fatalf("tool event failed: %v", event.Err)
		}
		if event.Type == "output" {
			output = event.Output
		}
	}
	var rendered string
	if err := json.Unmarshal(output, &rendered); err != nil || !strings.Contains(rendered, "[UNTRUSTED_CONTENT source=mcp_output") || !strings.Contains(rendered, "echo:hello") {
		t.Fatalf("output=%s rendered=%q err=%v", output, rendered, err)
	}
	if called.Load() != 1 {
		t.Fatalf("MCP call count=%d, want 1", called.Load())
	}
	if deletes.Load() != 2 {
		t.Fatalf("session DELETE count=%d, want discovery and invocation cleanup", deletes.Load())
	}
}
