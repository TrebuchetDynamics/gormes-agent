package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const geminiCloudCodeAPIMode = "gemini_cloudcode"

type geminiCloudCodeTransport struct{}

func (geminiCloudCodeTransport) APIMode() string { return geminiCloudCodeAPIMode }

func (geminiCloudCodeTransport) BuildRequest(req ChatRequest) (ProviderRequest, error) {
	descriptors := SanitizeToolDescriptors(req.Tools)
	body, err := buildGeminiCloudCodeRequestBody(req, descriptors)
	if err != nil {
		return ProviderRequest{}, err
	}
	return ProviderRequest{
		APIMode:         geminiCloudCodeAPIMode,
		Path:            "/v1/projects/-/locations/-/publishers/google/models/" + req.Model + ":generateContent",
		Body:            body,
		ToolDescriptors: descriptors,
	}, nil
}

func (geminiCloudCodeTransport) OpenFixtureStream(body io.ReadCloser, req ProviderRequest) (Stream, error) {
	defer body.Close()
	raw, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	return newGeminiCloudCodeStream(raw, req.ToolDescriptors), nil
}

func buildGeminiCloudCodeRequestBody(req ChatRequest, descriptors []ToolDescriptor) ([]byte, error) {
	var systemInstruction string
	var contents []geminiContent
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemInstruction = msg.Content
			continue
		}
		content := geminiContent{Role: geminiRole(msg.Role)}
		content.Parts = append(content.Parts, geminiPart{Text: msg.Content})
		for _, call := range msg.ToolCalls {
			content.Parts = append(content.Parts, geminiPart{
				FunctionCall: &geminiFunctionCall{
					Name: call.Name,
					Args: call.Arguments,
				},
			})
		}
		if msg.ToolCallID != "" {
			content.Parts = append(content.Parts, geminiPart{
				FunctionResponse: &geminiFunctionResponse{
					Name:     msg.Name,
					Response: json.RawMessage(msg.Content),
				},
			})
		}
		contents = append(contents, content)
	}

	payload := geminiRequest{
		Model:    req.Model,
		Contents: contents,
	}
	if systemInstruction != "" {
		payload.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: systemInstruction}},
		}
	}
	if req.Temperature != nil {
		payload.GenerationConfig.Temperature = *req.Temperature
	}
	// Always send maxOutputTokens. Without it some Gemini model variants (and
	// Vertex-hosted Gemini) default to a very low limit that truncates tool
	// calls mid-stream. Use a generous floor when the caller has not requested
	// a specific cap. Mirrors Hermes fix(gemini): default native maxOutputTokens
	// (ec46f5912).
	const geminiDefaultMaxOutputTokens = 65535
	if req.MaxTokens > 0 {
		payload.GenerationConfig.MaxOutputTokens = req.MaxTokens
	} else {
		payload.GenerationConfig.MaxOutputTokens = geminiDefaultMaxOutputTokens
	}
	if len(descriptors) > 0 {
		payload.ToolConfig = &geminiToolConfig{
			FunctionCallingConfig: geminiFunctionCallingConfig{
				Mode: "AUTO",
			},
		}
		for _, d := range descriptors {
			payload.Tools = append(payload.Tools, geminiTool{
				FunctionDeclarations: []geminiFunctionDeclaration{{
					Name:        d.Name,
					Description: d.Description,
					Parameters:  d.Schema,
				}},
			})
		}
	}

	return json.Marshal(payload)
}

func geminiRole(role string) string {
	switch role {
	case "user", "tool":
		return "user"
	case "assistant":
		return "model"
	default:
		return "user"
	}
}

type geminiRequest struct {
	Model             string                 `json:"model"`
	SystemInstruction *geminiContent         `json:"systemInstruction,omitempty"`
	Contents          []geminiContent        `json:"contents"`
	Tools             []geminiTool           `json:"tools,omitempty"`
	ToolConfig        *geminiToolConfig      `json:"toolConfig,omitempty"`
	GenerationConfig  geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text             string                  `json:"text,omitempty"`
	FunctionCall     *geminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
}

type geminiFunctionCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

type geminiFunctionResponse struct {
	Name     string          `json:"name"`
	Response json.RawMessage `json:"response"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDeclaration `json:"functionDeclarations"`
}

type geminiFunctionDeclaration struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type geminiToolConfig struct {
	FunctionCallingConfig geminiFunctionCallingConfig `json:"functionCallingConfig"`
}

type geminiFunctionCallingConfig struct {
	Mode string `json:"mode"`
}

type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

type geminiCloudCodeStream struct {
	frames          []geminiStreamEvent
	toolDescriptors []ToolDescriptor
	idx             int
}

type geminiStreamEvent struct {
	Candidates    []geminiCandidate   `json:"candidates"`
	UsageMetadata geminiUsageMetadata `json:"usageMetadata"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
}

func newGeminiCloudCodeStream(raw []byte, descriptors []ToolDescriptor) *geminiCloudCodeStream {
	var events []geminiStreamEvent
	_ = json.Unmarshal(raw, &events)
	return &geminiCloudCodeStream{frames: events, toolDescriptors: descriptors}
}

func (s *geminiCloudCodeStream) Recv(_ context.Context) (Event, error) {
	if s.idx >= len(s.frames) {
		return Event{Kind: EventDone}, nil
	}
	frame := s.frames[s.idx]
	s.idx++
	if len(frame.Candidates) == 0 {
		return Event{Kind: EventToken, Token: ""}, nil
	}
	candidate := frame.Candidates[0]
	var text strings.Builder
	var toolCalls []ToolCall
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			text.WriteString(part.Text)
		}
		if part.FunctionCall != nil {
			toolCalls = append(toolCalls, ToolCall{
				Name:      part.FunctionCall.Name,
				Arguments: part.FunctionCall.Args,
			})
		}
	}
	if len(toolCalls) > 0 {
		return Event{Kind: EventDone, ToolCalls: toolCalls, FinishReason: "tool_calls"}, nil
	}
	if candidate.FinishReason != "" {
		return Event{Kind: EventDone, FinishReason: candidate.FinishReason}, nil
	}
	return Event{Kind: EventToken, Token: text.String()}, nil
}

func (s *geminiCloudCodeStream) SessionID() string { return "" }

func (s *geminiCloudCodeStream) Close() error { return nil }

func classifyGeminiCloudCodeError(status int, body string, header http.Header) ProviderErrorClassification {
	httpErr := newHTTPError(status, body, header)
	return ClassifyProviderError(httpErr)
}

func geminiCloudCodeProviderStatus() ProviderStatus {
	status := openAICompatibleProviderStatus("gemini_cloudcode", "")
	status.Provider = "gemini_cloudcode"
	status.Runtime = geminiCloudCodeAPIMode
	return status
}
