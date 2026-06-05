package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

const mcpserverProtocolVersion = "2024-11-05"

type MCPServer struct {
	mu           sync.Mutex
	stdin        *os.File
	stdout       *os.File
	tools        map[string]ToolDef
	sessionStore SessionLister
	channelDir   ChannelDirectoryProvider
	db           interface {
		QueryContext(ctx context.Context, query string, args ...any) (interface {
			Close()
			Next() bool
		}, error)
	}
	toolsMeta map[string]ToolMeta
}

type ToolDef struct {
	Description string
	InputSchema map[string]interface{}
	Handler     func(ctx context.Context, args map[string]interface{}) (interface{}, error)
}

type ToolMeta struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
}

type SessionLister interface {
	ListAll(ctx context.Context) ([]SessionEntry, error)
}

type SessionEntry struct {
	SessionKey string
	Source     string
	ChatID     string
	Title      string
	UserID     string
	CreatedAt  int64
	UpdatedAt  int64
}

type mcpJSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

type mcpJSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *mcpErr     `json:"error,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

type mcpErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *MCPServer) RegisterTool(name, description string, inputSchema map[string]interface{}, handler func(ctx context.Context, args map[string]interface{}) (interface{}, error)) {
	if s.tools == nil {
		s.tools = make(map[string]ToolDef)
	}
	if s.toolsMeta == nil {
		s.toolsMeta = make(map[string]ToolMeta)
	}
	s.tools[name] = ToolDef{
		Description: description,
		InputSchema: inputSchema,
		Handler:     handler,
	}
	s.toolsMeta[name] = ToolMeta{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
	}
}

func (s *MCPServer) ServeSTDIO(ctx context.Context) error {
	s.mu.Lock()
	s.stdin = os.Stdin
	s.stdout = os.Stdout
	s.mu.Unlock()

	return s.serve(ctx, os.Stdin, os.Stdout)
}

func (s *MCPServer) serve(ctx context.Context, in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err.Error() == "EOF" {
				return nil
			}
			continue
		}
		// Remove trailing newline
		if len(line) > 0 && line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}

		var req mcpJSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}

		s.handleRequest(ctx, req, out)
	}
}

func (s *MCPServer) handleRequest(ctx context.Context, req mcpJSONRPCRequest, out io.Writer) {
	resp := mcpJSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	result, err := s.dispatch(ctx, req.Method, req.Params)
	if err != nil {
		resp.Error = &mcpErr{Code: -32603, Message: err.Error()}
	} else {
		resp.Result = result
	}

	data, _ := json.Marshal(resp)
	out.Write(data)
	out.Write([]byte{'\n'})
}

func (s *MCPServer) dispatch(ctx context.Context, method string, params json.RawMessage) (interface{}, error) {
	switch method {
	case "initialize":
		return s.handleInitialize(params)
	case "tools/list":
		return s.handleToolsList(params)
	case "tools/call":
		return s.handleToolsCall(ctx, params)
	case "notifications/initialized":
		return nil, nil
	default:
		return nil, fmt.Errorf("method not found: %s", method)
	}
}

func (s *MCPServer) handleInitialize(params json.RawMessage) (interface{}, error) {
	var p struct {
		ProtocolVersion string `json:"protocolVersion"`
		ClientInfo      struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"clientInfo"`
		Capabilities struct {
			Roots struct {
				ListChanged bool `json:"listChanged"`
			} `json:"roots"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	capabilities := map[string]interface{}{
		"tools": map[string]interface{}{},
	}
	result := map[string]interface{}{
		"protocolVersion": mcpserverProtocolVersion,
		"capabilities":    capabilities,
		"serverInfo": map[string]interface{}{
			"name":    "gormes-mcp-server",
			"version": "1.0.0",
		},
	}
	return result, nil
}

func (s *MCPServer) handleToolsList(params json.RawMessage) (interface{}, error) {
	tools := make([]ToolMeta, 0, len(s.toolsMeta))
	for _, t := range s.toolsMeta {
		tools = append(tools, t)
	}
	return map[string]interface{}{
		"tools": tools,
	}, nil
}

func (s *MCPServer) handleToolsCall(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments,omitempty"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	tool, ok := s.tools[p.Name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", p.Name)
	}
	result, err := tool.Handler(ctx, p.Arguments)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": formatResult(result),
			},
		},
	}, nil
}

func formatResult(r interface{}) string {
	b, _ := json.Marshal(r)
	return string(b)
}

type MCPToolResult struct {
	Content []MCPToolContent
	IsError bool
}

type MCPToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func toolDescription(name string) string {
	descriptions := map[string]string{
		"conversations_list": "List all conversations/sessions",
		"messages_list":      "List messages in a session",
		"messages_get":       "Get a specific message",
		"tools_list":         "List all available tools",
		"sessions_list":      "List all sessions",
	}
	if d, ok := descriptions[name]; ok {
		return d
	}
	return ""
}

func toolInputSchema(name string) map[string]interface{} {
	switch name {
	case "conversations_list":
		return map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"limit": map[string]interface{}{
					"type":    "number",
					"default": 50,
				},
			},
		}
	case "messages_list":
		return map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"session_key": map[string]interface{}{
					"type": "string",
				},
				"limit": map[string]interface{}{
					"type":    "number",
					"default": 50,
				},
			},
			"required": []string{"session_key"},
		}
	case "messages_get":
		return map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"session_key": map[string]interface{}{
					"type": "string",
				},
				"message_id": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []string{"session_key", "message_id"},
		}
	default:
		return map[string]interface{}{"type": "object"}
	}
}

const defaultMCPListLimit = 50

func normalizeMCPListLimit(args map[string]interface{}) int {
	limit := defaultMCPListLimit
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	if limit <= 0 {
		return defaultMCPListLimit
	}
	return limit
}

func (s *MCPServer) conversationsListHandler(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	limit := normalizeMCPListLimit(args)
	var sessions []SessionEntry
	var err error
	if s.sessionStore != nil {
		sessions, err = s.sessionStore.ListAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("list sessions: %w", err)
		}
	} else {
		sessions = []SessionEntry{}
	}
	if limit < len(sessions) {
		sessions = sessions[:limit]
	}
	type conversation struct {
		SessionKey string `json:"session_key"`
		Source     string `json:"source"`
		ChatID     string `json:"chat_id"`
		Title      string `json:"title"`
		UserID     string `json:"user_id"`
		UpdatedAt  int64  `json:"updated_at"`
	}
	convs := make([]conversation, 0, len(sessions))
	for _, ses := range sessions {
		convs = append(convs, conversation{
			SessionKey: ses.SessionKey,
			Source:     ses.Source,
			ChatID:     ses.ChatID,
			Title:      ses.Title,
			UserID:     ses.UserID,
			UpdatedAt:  ses.UpdatedAt,
		})
	}
	return map[string]interface{}{
		"count":         len(convs),
		"conversations": convs,
	}, nil
}

func (s *MCPServer) messagesListHandler(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	sessionKey, ok := args["session_key"].(string)
	if !ok || sessionKey == "" {
		return nil, fmt.Errorf("session_key is required")
	}
	limit := normalizeMCPListLimit(args)
	var messages []MessageEntry
	var err error
	if s.db != nil {
		messages, err = listMessages(ctx, s.db, sessionKey, limit)
		if err != nil {
			return nil, fmt.Errorf("list messages: %w", err)
		}
	} else {
		messages = []MessageEntry{}
	}
	type msg struct {
		ID        string `json:"id"`
		Role      string `json:"role"`
		Content   string `json:"content"`
		Timestamp int64  `json:"timestamp"`
	}
	msgs := make([]msg, 0, len(messages))
	for _, m := range messages {
		msgs = append(msgs, msg{
			ID:        m.ID,
			Role:      m.Role,
			Content:   m.Content,
			Timestamp: m.Timestamp,
		})
	}
	return map[string]interface{}{
		"session_key": sessionKey,
		"count":       len(msgs),
		"messages":    msgs,
	}, nil
}

func (s *MCPServer) messagesGetHandler(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	sessionKey, ok := args["session_key"].(string)
	if !ok || sessionKey == "" {
		return nil, fmt.Errorf("session_key is required")
	}
	messageID, ok := args["message_id"].(string)
	if !ok || messageID == "" {
		return nil, fmt.Errorf("message_id is required")
	}
	var m MessageEntry
	var err error
	if s.db != nil {
		m, err = getMessage(ctx, s.db, sessionKey, messageID)
		if err != nil {
			return nil, fmt.Errorf("get message: %w", err)
		}
		if m.ID == "" {
			return nil, fmt.Errorf("message not found: %s", messageID)
		}
	}
	return map[string]interface{}{
		"session_key":     sessionKey,
		"id":              m.ID,
		"role":            m.Role,
		"message_content": m.Content,
		"timestamp":       m.Timestamp,
	}, nil
}

func (s *MCPServer) toolsListHandler(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	tools := make([]map[string]interface{}, 0, len(s.tools))
	for name := range s.tools {
		meta := s.toolsMeta[name]
		tools = append(tools, map[string]interface{}{
			"name":        meta.Name,
			"description": meta.Description,
			"inputSchema": meta.InputSchema,
		})
	}
	return map[string]interface{}{
		"count": len(tools),
		"tools": tools,
	}, nil
}

func (s *MCPServer) sessionsListHandler(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	limit := normalizeMCPListLimit(args)
	var sessions []SessionEntry
	var err error
	if s.sessionStore != nil {
		sessions, err = s.sessionStore.ListAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("list sessions: %w", err)
		}
	} else {
		sessions = []SessionEntry{}
	}
	if limit < len(sessions) {
		sessions = sessions[:limit]
	}
	type sess struct {
		SessionKey string `json:"_session_key"`
		Source     string `json:"source"`
		ChatID     string `json:"chat_id"`
		Title      string `json:"title"`
		UserID     string `json:"user_id"`
		CreatedAt  int64  `json:"created_at"`
		UpdatedAt  int64  `json:"updated_at"`
	}
	out := make([]sess, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, sess{
			SessionKey: s.SessionKey,
			Source:     s.Source,
			ChatID:     s.ChatID,
			Title:      s.Title,
			UserID:     s.UserID,
			CreatedAt:  s.CreatedAt,
			UpdatedAt:  s.UpdatedAt,
		})
	}
	return map[string]interface{}{
		"count":    len(out),
		"sessions": out,
	}, nil
}

type MessageEntry struct {
	ID        string
	Role      string
	Content   string
	Timestamp int64
}

func listMessages(ctx context.Context, db interface {
	QueryContext(ctx context.Context, query string, args ...any) (interface {
		Close()
		Next() bool
	}, error)
}, sessionKey string, limit int) ([]MessageEntry, error) {
	return nil, nil
}

func getMessage(ctx context.Context, db interface {
	QueryContext(ctx context.Context, query string, args ...any) (interface {
		Close()
		Next() bool
	}, error)
}, sessionKey, messageID string) (MessageEntry, error) {
	return MessageEntry{}, nil
}

func (s *MCPServer) RegisterDefaultTools() {
	s.RegisterTool("conversations_list", "List all conversations/sessions", toolInputSchema("conversations_list"), s.conversationsListHandler)
	s.RegisterTool("messages_list", "List messages in a session", toolInputSchema("messages_list"), s.messagesListHandler)
	s.RegisterTool("messages_get", "Get a specific message", toolInputSchema("messages_get"), s.messagesGetHandler)
	s.RegisterTool("tools_list", "List all available tools", toolInputSchema("tools_list"), s.toolsListHandler)
	s.RegisterTool("sessions_list", "List all sessions", toolInputSchema("sessions_list"), s.sessionsListHandler)
	s.RegisterTool("channels_list", "List all channels across messaging platforms", toolInputSchema("channels_list"), s.channelsListHandler)
}
