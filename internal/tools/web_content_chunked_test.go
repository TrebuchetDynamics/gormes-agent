package tools

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestChunkedWebContentProcessor_Process_SmallContent(t *testing.T) {
	mockClient := llm.NewMockClient()
	mockClient.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "Small content summary"},
		{Kind: llm.EventDone},
	}, "test-session")

	cfg := ChunkedWebContentProcessorConfig{
		MaxContentSize: 2_000_000,
		ChunkThreshold: 500_000,
		ChunkSize:      100_000,
		MaxOutputChars: 5000,
		MaxParallelism: 3,
		MinLength:      10,
	}
	processor := NewChunkedWebContentProcessor(mockClient, "test-model", cfg)
	if processor == nil {
		t.Fatal("expected non-nil processor")
	}

	// Use content that's above MinLength but below ChunkThreshold (single-pass)
	result, err := processor.ProcessWebContent(context.Background(), WebContentProcessRequest{
		URL:     "https://example.com",
		Title:   "Test",
		Content: strings.Repeat("x", 1000), // 1000 chars - single pass
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Small content summary" {
		t.Errorf("result = %q, want %q", result, "Small content summary")
	}
}

func TestChunkedWebContentProcessor_Process_BelowMinLength(t *testing.T) {
	mockClient := llm.NewMockClient()
	cfg := DefaultChunkedWebContentProcessorConfig()
	processor := NewChunkedWebContentProcessor(mockClient, "test-model", cfg)
	if processor == nil {
		t.Fatal("expected non-nil processor")
	}

	// Content below MinLength (5000 by default) should be returned unchanged
	shortContent := "Too short"
	result, err := processor.ProcessWebContent(context.Background(), WebContentProcessRequest{
		URL:     "https://example.com",
		Title:   "Test",
		Content: shortContent,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != shortContent {
		t.Errorf("result = %q, want %q", result, shortContent)
	}

	// Verify no LLM calls were made
	if len(mockClient.Requests()) != 0 {
		t.Errorf("expected no LLM calls for short content, got %d", len(mockClient.Requests()))
	}
}

func TestChunkedWebContentProcessor_Process_ExceedsMaxSize(t *testing.T) {
	mockClient := llm.NewMockClient()
	cfg := DefaultChunkedWebContentProcessorConfig()
	processor := NewChunkedWebContentProcessor(mockClient, "test-model", cfg)
	if processor == nil {
		t.Fatal("expected non-nil processor")
	}

	// Create content that exceeds MaxContentSize
	largeContent := strings.Repeat("x", 2_500_000)
	_, err := processor.ProcessWebContent(context.Background(), WebContentProcessRequest{
		URL:     "https://example.com",
		Title:   "Test",
		Content: largeContent,
	})
	if err == nil {
		t.Fatal("expected error for content exceeding max size")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestChunkedWebContentProcessor_Process_NilClient(t *testing.T) {
	cfg := DefaultChunkedWebContentProcessorConfig()
	processor := NewChunkedWebContentProcessor(nil, "test-model", cfg)
	if processor != nil {
		t.Error("expected nil processor for nil client")
	}
}

func TestChunkedWebContentProcessor_Process_ChunkedFlow(t *testing.T) {
	mockClient := llm.NewMockClient()

	// Content that creates exactly 3 chunks with ChunkSize=50 and threshold=100
	// Each "Word. " is 5 chars. 15 * 5 = 75 chars per sentence.
	// We need enough content to create 3 chunks.
	// With targetSize=50, we should get chunks at sentence boundaries.
	// Let's use 3 sentences which should create 3 chunks.
	// "Word. Word. Word. Word. Word. " = 30 chars - this won't split into 3 chunks
	// Let's use a larger content that definitely creates multiple chunks
	content := strings.Repeat("This is a test. ", 20) // 240 chars, ~4-5 chunks with ChunkSize=50

	// Provide enough mock streams (5 chunks + 1 synthesis)
	for i := 0; i < 6; i++ {
		mockClient.Script([]llm.Event{
			{Kind: llm.EventToken, Token: "Summary"},
			{Kind: llm.EventDone},
		}, "session")
	}

	cfg := ChunkedWebContentProcessorConfig{
		MaxContentSize: 2_000_000,
		ChunkThreshold: 50, // Low threshold to force chunking
		ChunkSize:      50, // Small chunks
		MaxOutputChars: 5000,
		MaxParallelism: 3,
		MinLength:      10,
	}
	processor := NewChunkedWebContentProcessor(mockClient, "test-model", cfg)
	if processor == nil {
		t.Fatal("expected non-nil processor")
	}

	result, err := processor.ProcessWebContent(context.Background(), WebContentProcessRequest{
		URL:     "https://example.com",
		Title:   "Test Document",
		Content: content,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Result should be "Summary" from synthesis (since all chunks succeed)
	if result != "Summary" {
		t.Errorf("result = %q, want %q", result, "Summary")
	}

	// Should have 6 LLM calls (5 chunks + 1 synthesis)
	requests := mockClient.Requests()
	if len(requests) != 6 {
		t.Errorf("expected 6 LLM calls, got %d", len(requests))
	}
}

func TestChunkedWebContentProcessor_Process_SingleChunkSuccess(t *testing.T) {
	mockClient := llm.NewMockClient()

	// Only 1 chunk succeeds, so no synthesis needed
	mockClient.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "Single chunk summary"},
		{Kind: llm.EventDone},
	}, "session-1")

	cfg := ChunkedWebContentProcessorConfig{
		MaxContentSize: 2_000_000,
		ChunkThreshold: 100,
		ChunkSize:      50,
		MaxOutputChars: 5000,
		MaxParallelism: 3,
		MinLength:      10,
	}
	processor := NewChunkedWebContentProcessor(mockClient, "test-model", cfg)

	// Content that results in only 1 chunk - small enough to not need splitting
	content := "Small content" // 13 chars
	result, err := processor.ProcessWebContent(context.Background(), WebContentProcessRequest{
		URL:     "https://example.com",
		Title:   "Test",
		Content: content,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Single chunk summary" {
		t.Errorf("result = %q, want %q", result, "Single chunk summary")
	}
}

func TestChunkedWebContentProcessor_Process_AllChunkSummariesFailDoesNotSynthesize(t *testing.T) {
	mockClient := llm.NewMockClient()

	// Both chunk summarization streams fail. A later success would hide the bug if
	// the processor tries to synthesize with an empty summary set.
	mockClient.Script([]llm.Event{}, "chunk-1")
	mockClient.Script([]llm.Event{}, "chunk-2")
	mockClient.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "invalid synthesis without summaries"},
		{Kind: llm.EventDone},
	}, "synthesis-should-not-run")

	cfg := ChunkedWebContentProcessorConfig{
		MaxContentSize: 2_000_000,
		ChunkThreshold: 5,
		ChunkSize:      10,
		MaxOutputChars: 5000,
		MaxParallelism: 1,
		MinLength:      1,
	}
	processor := NewChunkedWebContentProcessor(mockClient, "test-model", cfg)

	_, err := processor.ProcessWebContent(context.Background(), WebContentProcessRequest{
		URL:     "https://example.com",
		Title:   "Test",
		Content: "abcdefghijabcdefghij",
	})
	if err == nil {
		t.Fatal("expected error when all chunk summaries fail")
	}
	if !strings.Contains(err.Error(), "no summaries available") {
		t.Fatalf("error = %v, want no summaries available", err)
	}
	if requests := mockClient.Requests(); len(requests) != 2 {
		t.Fatalf("expected only chunk attempts and no synthesis request, got %d requests", len(requests))
	}
}

func TestChunkedWebContentProcessor_Process_PartialChunkSummaryFailureDoesNotSynthesize(t *testing.T) {
	mockClient := llm.NewMockClient()

	mockClient.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "first chunk summary"},
		{Kind: llm.EventDone},
	}, "chunk-1")
	mockClient.Script([]llm.Event{}, "chunk-2")
	mockClient.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "invalid synthesis with missing chunk"},
		{Kind: llm.EventDone},
	}, "synthesis-should-not-run")

	cfg := ChunkedWebContentProcessorConfig{
		MaxContentSize: 2_000_000,
		ChunkThreshold: 5,
		ChunkSize:      10,
		MaxOutputChars: 5000,
		MaxParallelism: 1,
		MinLength:      1,
	}
	processor := NewChunkedWebContentProcessor(mockClient, "test-model", cfg)

	_, err := processor.ProcessWebContent(context.Background(), WebContentProcessRequest{
		URL:     "https://example.com",
		Title:   "Test",
		Content: "abcdefghijabcdefghij",
	})
	if err == nil {
		t.Fatal("expected error when any chunk summary fails")
	}
	if !strings.Contains(err.Error(), "chunk summary failed") {
		t.Fatalf("error = %v, want chunk summary failed", err)
	}
	if requests := mockClient.Requests(); len(requests) != 2 {
		t.Fatalf("expected only chunk attempts and no synthesis request, got %d requests", len(requests))
	}
}

func TestChunkedWebContentProcessor_Process_WhitespaceOnlyChunkSummaryDoesNotSynthesize(t *testing.T) {
	mockClient := llm.NewMockClient()

	mockClient.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "first chunk summary"},
		{Kind: llm.EventDone},
	}, "chunk-1")
	mockClient.Script([]llm.Event{
		{Kind: llm.EventToken, Token: " \n\t "},
		{Kind: llm.EventDone},
	}, "chunk-2")
	mockClient.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "invalid synthesis with blank chunk"},
		{Kind: llm.EventDone},
	}, "synthesis-should-not-run")

	cfg := ChunkedWebContentProcessorConfig{
		MaxContentSize: 2_000_000,
		ChunkThreshold: 5,
		ChunkSize:      10,
		MaxOutputChars: 5000,
		MaxParallelism: 1,
		MinLength:      1,
	}
	processor := NewChunkedWebContentProcessor(mockClient, "test-model", cfg)

	_, err := processor.ProcessWebContent(context.Background(), WebContentProcessRequest{
		URL:     "https://example.com",
		Title:   "Test",
		Content: "abcdefghijabcdefghij",
	})
	if err == nil {
		t.Fatal("expected error when a chunk summary is whitespace-only")
	}
	if !strings.Contains(err.Error(), "chunk summary failed") {
		t.Fatalf("error = %v, want chunk summary failed", err)
	}
	if requests := mockClient.Requests(); len(requests) != 2 {
		t.Fatalf("expected only chunk attempts and no synthesis request, got %d requests", len(requests))
	}
}

func TestChunkedWebContentProcessor_Process_SynthesisFailureFallback(t *testing.T) {
	mockClient := llm.NewMockClient()

	// Chunk summaries succeed
	mockClient.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "First chunk summary"},
		{Kind: llm.EventDone},
	}, "session-1")
	mockClient.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "Second chunk summary"},
		{Kind: llm.EventDone},
	}, "session-2")
	// Synthesis fails (empty events = EOF error)
	mockClient.Script([]llm.Event{}, "session-3")

	cfg := ChunkedWebContentProcessorConfig{
		MaxContentSize: 2_000_000,
		ChunkThreshold: 5,
		ChunkSize:      20,
		MaxOutputChars: 5000,
		MaxParallelism: 1,
		MinLength:      10,
	}
	processor := NewChunkedWebContentProcessor(mockClient, "test-model", cfg)

	content := "abcdefghijabcdefghijabcdefghijabcdefghij"
	result, err := processor.ProcessWebContent(context.Background(), WebContentProcessRequest{
		URL:     "https://example.com",
		Title:   "Test",
		Content: content,
	})
	// Should fallback to concatenation even on synthesis error
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "First chunk summary") || !strings.Contains(result, "Second chunk summary") {
		t.Errorf("result = %q, expected to contain both chunk summaries", result)
	}
}

func TestChunkedWebContentProcessor_Process_EnforcesMaxOutputChars(t *testing.T) {
	mockClient := llm.NewMockClient()
	mockClient.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "1234567890"},
		{Kind: llm.EventDone},
	}, "session")

	cfg := ChunkedWebContentProcessorConfig{
		MaxContentSize: 2_000_000,
		ChunkThreshold: 500_000,
		ChunkSize:      100_000,
		MaxOutputChars: 5,
		MaxParallelism: 3,
		MinLength:      10,
	}
	processor := NewChunkedWebContentProcessor(mockClient, "test-model", cfg)

	result, err := processor.ProcessWebContent(context.Background(), WebContentProcessRequest{
		URL:     "https://example.com",
		Title:   "Test",
		Content: strings.Repeat("x", 1000),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "12345" {
		t.Errorf("result = %q, want max-output truncated %q", result, "12345")
	}
}

func TestChunkedWebContentProcessor_Process_EmptyContent(t *testing.T) {
	mockClient := llm.NewMockClient()
	cfg := DefaultChunkedWebContentProcessorConfig()
	processor := NewChunkedWebContentProcessor(mockClient, "test-model", cfg)

	result, err := processor.ProcessWebContent(context.Background(), WebContentProcessRequest{
		URL:     "https://example.com",
		Title:   "Test",
		Content: "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty content is below MinLength, so returned as-is
	if result != "" {
		t.Errorf("result = %q, want empty string", result)
	}
}

func TestChunkedWebContentProcessor_DefaultConfig(t *testing.T) {
	mockClient := llm.NewMockClient()
	cfg := DefaultChunkedWebContentProcessorConfig()

	if cfg.MaxContentSize != 2_000_000 {
		t.Errorf("MaxContentSize = %d, want 2000000", cfg.MaxContentSize)
	}
	if cfg.ChunkThreshold != 500_000 {
		t.Errorf("ChunkThreshold = %d, want 500000", cfg.ChunkThreshold)
	}
	if cfg.ChunkSize != 100_000 {
		t.Errorf("ChunkSize = %d, want 100000", cfg.ChunkSize)
	}
	if cfg.MaxOutputChars != 5000 {
		t.Errorf("MaxOutputChars = %d, want 5000", cfg.MaxOutputChars)
	}
	if cfg.MaxParallelism != 3 {
		t.Errorf("MaxParallelism = %d, want 3", cfg.MaxParallelism)
	}
	if cfg.MinLength != 5000 {
		t.Errorf("MinLength = %d, want 5000", cfg.MinLength)
	}

	processor := NewChunkedWebContentProcessor(mockClient, "test-model", cfg)
	if processor == nil {
		t.Fatal("expected non-nil processor")
	}
}

func TestChunkedWebContentProcessor_ConfigZeros(t *testing.T) {
	mockClient := llm.NewMockClient()
	// Passing zero values should use defaults
	cfg := ChunkedWebContentProcessorConfig{}
	processor := NewChunkedWebContentProcessor(mockClient, "test-model", cfg)
	if processor == nil {
		t.Fatal("expected non-nil processor")
	}
	// Verify defaults are applied by checking it processes without error
	result, err := processor.ProcessWebContent(context.Background(), WebContentProcessRequest{
		URL:     "https://example.com",
		Title:   "Test",
		Content: "Tiny",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// "Tiny" is below MinLength, so returned unchanged
	if result != "Tiny" {
		t.Errorf("result = %q, want %q", result, "Tiny")
	}
}

func TestChunkedWebContentProcessor_NormalizeConfigUsesSharedDefaults(t *testing.T) {
	cfg := normalizeChunkedWebContentProcessorConfig(ChunkedWebContentProcessorConfig{
		MaxContentSize: 42,
		ChunkSize:      17,
	})
	defaults := DefaultChunkedWebContentProcessorConfig()

	if cfg.MaxContentSize != 42 {
		t.Fatalf("MaxContentSize = %d, want explicit override 42", cfg.MaxContentSize)
	}
	if cfg.ChunkSize != 17 {
		t.Fatalf("ChunkSize = %d, want explicit override 17", cfg.ChunkSize)
	}
	if cfg.ChunkThreshold != defaults.ChunkThreshold {
		t.Fatalf("ChunkThreshold = %d, want default %d", cfg.ChunkThreshold, defaults.ChunkThreshold)
	}
	if cfg.MaxOutputChars != defaults.MaxOutputChars {
		t.Fatalf("MaxOutputChars = %d, want default %d", cfg.MaxOutputChars, defaults.MaxOutputChars)
	}
	if cfg.MaxParallelism != defaults.MaxParallelism {
		t.Fatalf("MaxParallelism = %d, want default %d", cfg.MaxParallelism, defaults.MaxParallelism)
	}
	if cfg.MinLength != defaults.MinLength {
		t.Fatalf("MinLength = %d, want default %d", cfg.MinLength, defaults.MinLength)
	}
}

func TestChunkedWebContentProcessor_ParallelismLimit(t *testing.T) {
	mockClient := llm.NewMockClient()

	// Add many chunk streams
	for i := 0; i < 10; i++ {
		mockClient.Script([]llm.Event{
			{Kind: llm.EventToken, Token: "Summary"},
			{Kind: llm.EventDone},
		}, "session")
	}
	// Plus synthesis
	mockClient.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "Final"},
		{Kind: llm.EventDone},
	}, "session")

	cfg := ChunkedWebContentProcessorConfig{
		MaxContentSize: 2_000_000,
		ChunkThreshold: 50,
		ChunkSize:      10,
		MaxOutputChars: 5000,
		MaxParallelism: 3, // Only 3 concurrent
		MinLength:      5,
	}
	processor := NewChunkedWebContentProcessor(mockClient, "test-model", cfg)

	// Content that creates 10 chunks
	content := strings.Repeat("abcdefghij", 10)
	_, err := processor.ProcessWebContent(context.Background(), WebContentProcessRequest{
		URL:     "https://example.com",
		Title:   "Test",
		Content: content,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChunkedWebContentProcessor_RequestContents(t *testing.T) {
	mockClient := llm.NewMockClient()
	mockClient.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "Summary"},
		{Kind: llm.EventDone},
	}, "session")

	cfg := ChunkedWebContentProcessorConfig{
		MaxContentSize: 2_000_000,
		ChunkThreshold: 500_000,
		ChunkSize:      100_000,
		MaxOutputChars: 5000,
		MaxParallelism: 3,
		MinLength:      10,
	}
	processor := NewChunkedWebContentProcessor(mockClient, "test-model", cfg)

	content := strings.Repeat("x", 10000)
	_, err := processor.ProcessWebContent(context.Background(), WebContentProcessRequest{
		URL:     "https://example.com/page",
		Title:   "Test Page Title",
		Content: content,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requests := mockClient.Requests()
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}

	req := requests[0]
	if req.Model != "test-model" {
		t.Errorf("Model = %q, want %q", req.Model, "test-model")
	}
	if len(req.Messages) != 2 {
		t.Fatalf("Messages count = %d, want 2", len(req.Messages))
	}
	if req.Messages[0].Role != "system" {
		t.Errorf("first message role = %q, want %q", req.Messages[0].Role, "system")
	}
	if req.Messages[1].Role != "user" {
		t.Errorf("second message role = %q, want %q", req.Messages[1].Role, "user")
	}
	if !strings.Contains(req.Messages[1].Content, "Test Page Title") {
		t.Errorf("user content missing title")
	}
	if !strings.Contains(req.Messages[1].Content, "https://example.com/page") {
		t.Errorf("user content missing URL")
	}
}

func TestChunkedWebContentProcessor_ChunkSystemPrompt(t *testing.T) {
	mockClient := llm.NewMockClient()

	cfg := ChunkedWebContentProcessorConfig{
		MaxContentSize: 2_000_000,
		ChunkThreshold: 50,
		ChunkSize:      20,
		MaxOutputChars: 5000,
		MaxParallelism: 3,
		MinLength:      10,
	}
	processor := NewChunkedWebContentProcessor(mockClient, "test-model", cfg)

	// Provide 3 chunk streams plus synthesis.
	for i := 0; i < 4; i++ {
		mockClient.Script([]llm.Event{
			{Kind: llm.EventToken, Token: "Summary"},
			{Kind: llm.EventDone},
		}, "session")
	}

	// Need content that splits into multiple chunks
	content := strings.Repeat("abcdefghij", 6)
	_, err := processor.ProcessWebContent(context.Background(), WebContentProcessRequest{
		URL:     "https://example.com",
		Title:   "Test",
		Content: content,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requests := mockClient.Requests()
	if len(requests) < 1 {
		t.Fatalf("expected at least 1 request, got %d", len(requests))
	}
	// First chunk request should have chunk-specific system prompt
	chunkPrompt := requests[0].Messages[0].Content
	if !strings.Contains(chunkPrompt, "SECTION of a larger document") {
		t.Errorf("chunk prompt missing 'SECTION of a larger document': %s", chunkPrompt)
	}
	if !strings.Contains(chunkPrompt, "Do NOT write introductions") {
		t.Errorf("chunk prompt missing 'Do NOT write introductions': %s", chunkPrompt)
	}
}

func TestChunkedWebContentProcessor_SynthesisSystemPrompt(t *testing.T) {
	mockClient := llm.NewMockClient()

	// We need enough chunks to trigger synthesis.
	// With ChunkSize=20 and content ~150 chars, we'll get ~7-8 chunks.
	// Plus 1 synthesis call.
	for i := 0; i < 9; i++ {
		mockClient.Script([]llm.Event{
			{Kind: llm.EventToken, Token: "Chunk"},
			{Kind: llm.EventDone},
		}, "session")
	}

	cfg := ChunkedWebContentProcessorConfig{
		MaxContentSize: 2_000_000,
		ChunkThreshold: 50,
		ChunkSize:      20,
		MaxOutputChars: 5000,
		MaxParallelism: 3,
		MinLength:      10,
	}
	processor := NewChunkedWebContentProcessor(mockClient, "test-model", cfg)

	// Content that creates multiple chunks requiring synthesis
	content := strings.Repeat("Word. ", 30) // ~150 chars
	_, err := processor.ProcessWebContent(context.Background(), WebContentProcessRequest{
		URL:     "https://example.com",
		Title:   "Test",
		Content: content,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requests := mockClient.Requests()
	// Last request should be synthesis (9th call)
	if len(requests) < 9 {
		t.Fatalf("expected 9+ requests, got %d", len(requests))
	}
	synthesisPrompt := requests[8].Messages[0].Content
	if !strings.Contains(strings.ToUpper(synthesisPrompt), "SYNTHESIZE") {
		t.Errorf("synthesis prompt missing 'SYNTHESIZE': %s", synthesisPrompt)
	}
	if !strings.Contains(strings.ToLower(synthesisPrompt), "cohesive") {
		t.Errorf("synthesis prompt missing 'cohesive': %s", synthesisPrompt)
	}
}

func TestChunkedWebContentProcessor_SinglePassSystemPrompt(t *testing.T) {
	mockClient := llm.NewMockClient()
	mockClient.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "Summary"},
		{Kind: llm.EventDone},
	}, "session")

	cfg := ChunkedWebContentProcessorConfig{
		MaxContentSize: 2_000_000,
		ChunkThreshold: 500_000,
		ChunkSize:      100_000,
		MaxOutputChars: 5000,
		MaxParallelism: 3,
		MinLength:      10,
	}
	processor := NewChunkedWebContentProcessor(mockClient, "test-model", cfg)

	content := strings.Repeat("x", 10000) // Large enough to process, but single-pass
	_, err := processor.ProcessWebContent(context.Background(), WebContentProcessRequest{
		URL:     "https://example.com",
		Title:   "Test",
		Content: content,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requests := mockClient.Requests()
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	// Single-pass should have standard system prompt
	prompt := requests[0].Messages[0].Content
	if !strings.Contains(prompt, "expert content analyst") {
		t.Errorf("single-pass prompt missing 'expert content analyst': %s", prompt)
	}
	if strings.Contains(prompt, "SECTION") {
		t.Errorf("single-pass prompt should not contain 'SECTION': %s", prompt)
	}
}

func TestChunkedWebContentProcessor_ChunkSplitting(t *testing.T) {
	cfg := ChunkedWebContentProcessorConfig{
		MaxContentSize: 2_000_000,
		ChunkThreshold: 500_000,
		ChunkSize:      100,
		MaxOutputChars: 5000,
		MaxParallelism: 3,
		MinLength:      10,
	}
	processor := &ChunkedWebContentProcessor{cfg: cfg}

	// Test content that doesn't need splitting
	chunks := processor.splitIntoChunks("Short content")
	if len(chunks) != 1 {
		t.Errorf("short content: got %d chunks, want 1", len(chunks))
	}

	// Test content larger than chunk size
	longContent := strings.Repeat("This is a sentence. ", 50) // ~1500 chars
	chunks = processor.splitIntoChunks(longContent)
	if len(chunks) < 2 {
		t.Errorf("long content: got %d chunks, want 2+", len(chunks))
	}

	// Verify each chunk is within reasonable size
	for i, chunk := range chunks {
		if len(chunk) > cfg.ChunkSize+2000 { // Allow some overflow for natural split points
			t.Errorf("chunk %d length %d exceeds ChunkSize %d by too much", i, len(chunk), cfg.ChunkSize)
		}
	}
}

func TestChunkedWebContentProcessor_ChunkSplittingKeepsValidUTF8(t *testing.T) {
	cfg := ChunkedWebContentProcessorConfig{
		ChunkSize: 1,
	}
	processor := &ChunkedWebContentProcessor{cfg: cfg}

	chunks := processor.splitIntoChunks("🙂🙂")
	if len(chunks) != 2 {
		t.Fatalf("got %d chunks, want 2", len(chunks))
	}
	for i, chunk := range chunks {
		if !utf8.ValidString(chunk) {
			t.Fatalf("chunk %d is not valid UTF-8: %q", i, chunk)
		}
	}
	if got := strings.Join(chunks, ""); got != "🙂🙂" {
		t.Fatalf("joined chunks = %q, want original content", got)
	}
}

func TestChunkedWebContentProcessor_LimitWebProcessedOutput(t *testing.T) {
	tests := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{name: "within limit", in: "abc", max: 3, want: "abc"},
		{name: "over limit", in: "abcdef", max: 3, want: "abc"},
		{name: "disabled", in: "abcdef", max: 0, want: "abcdef"},
		{name: "unicode chars", in: "åß∂ƒ", max: 2, want: "åß"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := limitWebProcessedOutput(tt.in, tt.max); got != tt.want {
				t.Fatalf("limitWebProcessedOutput(%q, %d) = %q, want %q", tt.in, tt.max, got, tt.want)
			}
		})
	}
}

func TestChunkedWebContentProcessor_FallbackSorting(t *testing.T) {
	cfg := ChunkedWebContentProcessorConfig{}
	processor := &ChunkedWebContentProcessor{cfg: cfg}

	summaries := []chunkSummary{
		{index: 2, summary: "Third"},
		{index: 0, summary: "First"},
		{index: 1, summary: "Second"},
	}

	result, err := processor.fallbackSummaries(summaries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be sorted by index
	expected := "First\n\n---\n\nSecond\n\n---\n\nThird"
	if result != expected {
		t.Errorf("result = %q, want %q", result, expected)
	}
}

func TestChunkedWebContentProcessor_JoinChunkSummariesSortsWithoutMutating(t *testing.T) {
	summaries := []chunkSummary{
		{index: 2, summary: "Third"},
		{index: 0, summary: "First"},
		{index: 1, summary: "Second"},
	}

	result := joinChunkSummaries(summaries)
	want := "First\n\n---\n\nSecond\n\n---\n\nThird"
	if result != want {
		t.Fatalf("joinChunkSummaries() = %q, want %q", result, want)
	}
	if summaries[0].summary != "Third" || summaries[1].summary != "First" || summaries[2].summary != "Second" {
		t.Fatalf("joinChunkSummaries mutated input order: %#v", summaries)
	}
}

func TestChunkedWebContentProcessor_FallbackEmptySummaries(t *testing.T) {
	cfg := ChunkedWebContentProcessorConfig{}
	processor := &ChunkedWebContentProcessor{cfg: cfg}

	_, err := processor.fallbackSummaries([]chunkSummary{})
	if err == nil {
		t.Error("expected error for empty summaries")
	}
}
