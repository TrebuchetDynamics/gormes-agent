package llm

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPicoClawProviderRegression_OpenRouterReasoningHidden(t *testing.T) {
	stream := newChatStream(io.NopCloser(strings.NewReader(joinSSEData(
		`{"choices":[{"delta":{"reasoning_content":"private chain of thought"}}]}`,
		`{"choices":[{"delta":{"content":"<think>do not show</think>PONG NEMOTRON FREE"}}]}`,
		`{"choices":[{"delta":{"content":"\n<reasoning>more hidden</reasoning> Done"}}]}`,
		`{"choices":[{"finish_reason":"stop"}],"usage":{"prompt_tokens":11,"completion_tokens":7}}`,
		`[DONE]`,
	))), "", nil)
	defer stream.Close()

	visible, reasoning, raw := collectProviderStream(t, stream)
	for _, leaked := range []string{"private chain", "do not show", "more hidden"} {
		if strings.Contains(visible, leaked) {
			t.Fatalf("assistant-visible content leaked reasoning %q: %q", leaked, visible)
		}
	}
	if !strings.Contains(visible, "PONG NEMOTRON FREE") || !strings.Contains(visible, "Done") {
		t.Fatalf("visible content = %q, want sanitized final answer text", visible)
	}
	if !strings.Contains(reasoning, "private chain of thought") {
		t.Fatalf("reasoning events = %q, want reasoning_content routed away from visible tokens", reasoning)
	}
	if !strings.Contains(raw, "reasoning_content") || !strings.Contains(raw, "do not show") {
		t.Fatalf("raw stream audit = %q, want original reasoning-bearing frames preserved", raw)
	}
}

func TestPicoClawProviderRegression_CodexOutputItemDoneYieldsAssistantText(t *testing.T) {
	stream, err := newCodexResponsesSSEStream(context.Background(), io.NopCloser(strings.NewReader(joinSSEData(
		`{"type":"response.output_text.delta","delta":"ignored partial "}`,
		`{"type":"response.output_item.done","item":{"type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"streamed codex final"}]}}`,
		`{"type":"response.completed","response":{"status":"completed","output":[],"usage":{"input_tokens":12,"output_tokens":3,"total_tokens":15}}}`,
		`[DONE]`,
	))), ProviderRequest{})
	if err != nil {
		t.Fatalf("newCodexResponsesSSEStream() error = %v", err)
	}
	defer stream.Close()

	visible, _, _ := collectProviderStream(t, stream)
	if visible != "streamed codex final" {
		t.Fatalf("visible content = %q, want response.output_item.done assistant text", visible)
	}
}

func TestPicoClawProviderRegression_Auth401NamesCredentialSource(t *testing.T) {
	const rawKey = "sk-live-secret-do-not-print"
	var sawAuthHeader bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got == "Bearer "+rawKey {
			sawAuthHeader = true
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error":{"message":"Invalid API Key %s","code":"invalid_api_key"}}`, rawKey)
	}))
	defer srv.Close()

	client := NewHTTPClientWithProvider(srv.URL, rawKey, "openrouter")
	_, err := client.OpenStream(context.Background(), ChatRequest{
		Model:    "openrouter/test-model",
		Messages: []Message{{Role: "user", Content: "ping"}},
	})
	if err == nil {
		t.Fatal("OpenStream() error = nil, want 401")
	}
	if !sawAuthHeader {
		t.Fatal("provider request did not send the configured bearer credential")
	}

	diag := BuildProviderFailureDiagnostic(ProviderFailureDiagnosticInput{
		Provider:         "openrouter",
		Model:            "openrouter/test-model",
		CredentialSource: "env:OPENROUTER_API_KEY=" + rawKey,
		Err:              err,
	})
	if diag.Provider != "openrouter" || diag.Model != "openrouter/test-model" {
		t.Fatalf("diagnostic identity = %+v, want provider/model named", diag)
	}
	if diag.CredentialSource == "" || !strings.Contains(diag.CredentialSource, "OPENROUTER_API_KEY") {
		t.Fatalf("credential source = %q, want safe source evidence", diag.CredentialSource)
	}
	if diag.Kind != ProviderErrorAuth || diag.Status != http.StatusUnauthorized || diag.Class != ClassFatal {
		t.Fatalf("diagnostic classification = %+v, want fatal auth 401", diag)
	}
	if diag.NextAction == "" {
		t.Fatalf("next action is empty in diagnostic: %+v", diag)
	}
	rendered := fmt.Sprintf("%+v", diag)
	if strings.Contains(rendered, rawKey) {
		t.Fatalf("diagnostic leaked raw credential: %s", rendered)
	}
}

func TestPicoClawProviderRegression_LMStudioLocalModelRouting(t *testing.T) {
	const localModel = "local-qwen-not-returned-by-model-list"
	var capturedModel string
	var modelListHit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			modelListHit = true
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"object":"list","data":[{"id":"different-loaded-model","object":"model","owned_by":"lmstudio"}]}`)
		case "/v1/chat/completions":
			body, _ := io.ReadAll(r.Body)
			capturedModel = jsonValue(body, "model")
			if got := r.Header.Get("Authorization"); got != "" {
				t.Fatalf("LM Studio Authorization header = %q, want empty local auth", got)
			}
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			bw := bufio.NewWriter(w)
			fmt.Fprint(bw, joinSSEData(
				`{"choices":[{"delta":{"content":"local ok"}}]}`,
				`{"choices":[{"finish_reason":"stop"}]}`,
				`[DONE]`,
			))
			bw.Flush()
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := NewLMStudioAdapter(srv.URL + "/v1").Client()
	stream, err := client.OpenStream(context.Background(), ChatRequest{
		Model:    localModel,
		Messages: []Message{{Role: "user", Content: "ping"}},
	})
	if err != nil {
		t.Fatalf("LM Studio OpenStream() error = %v", err)
	}
	defer stream.Close()

	visible, _, _ := collectProviderStream(t, stream)
	if visible != "local ok" {
		t.Fatalf("visible content = %q, want local chat completion", visible)
	}
	if capturedModel != localModel {
		t.Fatalf("captured model = %q, want %q", capturedModel, localModel)
	}
	if modelListHit {
		t.Fatal("LM Studio chat routing consulted /models before dispatch and risks false negatives")
	}
}

func TestPicoClawProviderRegression_RetryableLLMFailureUsesClassifiedRetry(t *testing.T) {
	classifier := &DefaultChainErrorClassifier{MaxRetriesPerProvider: 2}

	retryable := ClassifyProviderError(&HTTPError{Status: http.StatusInternalServerError, Body: `{"error":{"message":"upstream failed"}}`})
	if retryable.Kind != ProviderErrorRetryable || retryable.Class != ClassRetryable {
		t.Fatalf("retryable classification = %+v, want retryable server failure", retryable)
	}
	if got := classifier.Decide(retryable, 1); got != ChainDecisionRetry {
		t.Fatalf("attempt 1 decision = %q, want retry", got)
	}
	if got := classifier.Decide(retryable, 2); got != ChainDecisionFallback {
		t.Fatalf("attempt 2 decision = %q, want fallback after retry budget", got)
	}

	auth := ClassifyProviderError(&HTTPError{Status: http.StatusUnauthorized, Body: `{"error":{"message":"invalid api key"}}`})
	if got := classifier.Decide(auth, 1); got != ChainDecisionAbort {
		t.Fatalf("auth decision = %q, want abort without retry", got)
	}
}

func collectProviderStream(t *testing.T, stream Stream) (visible, reasoning, raw string) {
	t.Helper()
	var visibleParts []string
	var reasoningParts []string
	var rawParts []string
	for {
		event, err := stream.Recv(context.Background())
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Recv() error = %v", err)
		}
		switch event.Kind {
		case EventToken:
			visibleParts = append(visibleParts, event.Token)
		case EventReasoning:
			reasoningParts = append(reasoningParts, event.Reasoning)
		}
		if len(event.Raw) > 0 {
			rawParts = append(rawParts, string(event.Raw))
		}
	}
	return strings.Join(visibleParts, ""), strings.Join(reasoningParts, "\n"), strings.Join(rawParts, "\n")
}

func joinSSEData(frames ...string) string {
	var b strings.Builder
	for _, frame := range frames {
		b.WriteString("data: ")
		b.WriteString(frame)
		b.WriteString("\n\n")
	}
	return b.String()
}

func jsonValue(body []byte, key string) string {
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return ""
	}
	value, _ := obj[key].(string)
	return value
}
