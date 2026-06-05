package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func codexResponsesStreamingBody(body []byte) ([]byte, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, err
	}
	obj["stream"] = json.RawMessage("true")
	return json.Marshal(obj)
}

func newCodexResponsesSSEStream(ctx context.Context, body io.ReadCloser, req ProviderRequest) (Stream, error) {
	defer body.Close()

	reader := newSSEReader(body)
	var completed *codexResponsesResponse
	var outputItems []codexResponsesOutputItem
	var outputText strings.Builder
	var reasoningText strings.Builder

	for {
		frame, err := reader.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		data := strings.TrimSpace(frame.data)
		if data == "" {
			continue
		}
		if data == "[DONE]" {
			break
		}

		event, err := decodeCodexResponsesSSEEvent(data)
		if err != nil {
			continue
		}
		eventType := strings.TrimSpace(event.Type)
		if eventType == "" {
			eventType = strings.TrimSpace(frame.event)
		}

		switch eventType {
		case "response.output_text.delta":
			outputText.WriteString(event.Delta)
		case "response.output_text.done":
			if event.Text != "" {
				outputText.Reset()
				outputText.WriteString(event.Text)
			}
		case "response.reasoning_summary_text.delta", "response.reasoning_text.delta":
			reasoningText.WriteString(event.Delta)
		case "response.reasoning_summary_text.done", "response.reasoning_text.done":
			if event.Text != "" {
				reasoningText.Reset()
				reasoningText.WriteString(event.Text)
			}
		case "response.output_item.done":
			if event.Item.Type != "" {
				outputItems = append(outputItems, event.Item)
			}
		case "response.completed":
			completed = &event.Response
		case "response.failed":
			return nil, fmt.Errorf("codex responses stream failed: %s", eventErrorText(event))
		}
	}

	response := codexResponsesResponse{Status: "completed"}
	if completed != nil {
		response = *completed
	}
	if len(response.Output) == 0 && len(outputItems) > 0 {
		response.Output = outputItems
	}
	if strings.TrimSpace(response.OutputText) == "" && outputText.Len() > 0 {
		response.OutputText = outputText.String()
	}
	if len(response.Output) == 0 && response.OutputText == "" && reasoningText.Len() > 0 {
		response.Output = []codexResponsesOutputItem{{
			Type: "reasoning",
			Summary: []codexResponsesOutputContent{{
				Type: "summary_text",
				Text: reasoningText.String(),
			}},
		}}
	}

	normalized, err := normalizeCodexResponsesResponseWithTools(response, req.ToolDescriptors)
	if err != nil {
		return nil, err
	}
	return newStaticProviderStream(normalized.Events), nil
}

type codexResponsesSSEEvent struct {
	Type     string                     `json:"type"`
	Delta    string                     `json:"delta"`
	Text     string                     `json:"text"`
	Item     codexResponsesOutputItem   `json:"item"`
	Response codexResponsesResponse     `json:"response"`
	Error    codexResponsesStreamError  `json:"error"`
	Usage    codexResponsesUsage        `json:"usage"`
	Output   []codexResponsesOutputItem `json:"output"`
}

type codexResponsesStreamError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	Type    string `json:"type"`
}

func decodeCodexResponsesSSEEvent(data string) (codexResponsesSSEEvent, error) {
	var event codexResponsesSSEEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return codexResponsesSSEEvent{}, err
	}
	if event.Response.Status == "" && (len(event.Output) > 0 || event.Usage != (codexResponsesUsage{})) {
		event.Response = codexResponsesResponse{
			Status: "completed",
			Output: event.Output,
			Usage:  event.Usage,
		}
	}
	return event, nil
}

func eventErrorText(event codexResponsesSSEEvent) string {
	for _, value := range []string{event.Error.Message, event.Error.Code, event.Error.Type} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "unknown"
}
