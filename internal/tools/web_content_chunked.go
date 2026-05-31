package tools

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// ChunkedWebContentProcessorConfig holds configuration for chunked processing.
type ChunkedWebContentProcessorConfig struct {
	// MaxContentSize is the maximum content size to accept.
	// Content above this size returns an error.
	MaxContentSize int
	// ChunkThreshold is the content length above which chunked processing is used.
	ChunkThreshold int
	// ChunkSize is the target size for each chunk.
	ChunkSize int
	// MaxOutputChars limits the final output.
	MaxOutputChars int
	// MaxParallelism limits concurrent goroutines for parallel chunk processing.
	MaxParallelism int
	// MinLength is the minimum content length to process.
	// Content below this is returned unchanged.
	MinLength int
}

// DefaultChunkedWebContentProcessorConfig returns the default configuration.
func DefaultChunkedWebContentProcessorConfig() ChunkedWebContentProcessorConfig {
	return ChunkedWebContentProcessorConfig{
		MaxContentSize: 2_000_000,
		ChunkThreshold: 500_000,
		ChunkSize:      100_000,
		MaxOutputChars: 5000,
		MaxParallelism: 3,
		MinLength:      5000,
	}
}

// ChunkedWebContentProcessor implements WebContentProcessor with chunked LLM processing
// for large content. It splits content into chunks, summarizes each in parallel,
// then synthesizes a final summary.
type ChunkedWebContentProcessor struct {
	client llm.Client
	model  string
	cfg    ChunkedWebContentProcessorConfig
}

// NewChunkedWebContentProcessor creates a new ChunkedWebContentProcessor.
// Returns nil if client is nil.
func NewChunkedWebContentProcessor(client llm.Client, model string, cfg ChunkedWebContentProcessorConfig) WebContentProcessor {
	if client == nil {
		return nil
	}
	if cfg.MaxContentSize <= 0 {
		cfg.MaxContentSize = 2_000_000
	}
	if cfg.ChunkThreshold <= 0 {
		cfg.ChunkThreshold = 500_000
	}
	if cfg.ChunkSize <= 0 {
		cfg.ChunkSize = 100_000
	}
	if cfg.MaxOutputChars <= 0 {
		cfg.MaxOutputChars = 5000
	}
	if cfg.MaxParallelism <= 0 {
		cfg.MaxParallelism = 3
	}
	if cfg.MinLength <= 0 {
		cfg.MinLength = 5000
	}
	return &ChunkedWebContentProcessor{
		client: client,
		model:  model,
		cfg:    cfg,
	}
}

// ProcessWebContent implements WebContentProcessor.
func (p *ChunkedWebContentProcessor) ProcessWebContent(ctx context.Context, req WebContentProcessRequest) (string, error) {
	content := req.Content
	contentLen := len(content)

	// Refuse content above max size
	if contentLen > p.cfg.MaxContentSize {
		return "", errors.New("content exceeds maximum size limit")
	}

	// Return original content if below minimum length
	if contentLen < p.cfg.MinLength {
		return content, nil
	}

	// Single-pass processing if content fits within threshold
	if contentLen <= p.cfg.ChunkThreshold {
		result, err := p.singlePassProcess(ctx, req)
		return limitWebProcessedOutput(result, p.cfg.MaxOutputChars), err
	}

	// Chunked processing for large content
	result, err := p.chunkedProcess(ctx, req)
	return limitWebProcessedOutput(result, p.cfg.MaxOutputChars), err
}

// limitWebProcessedOutput enforces MaxOutputChars on processed LLM output.
func limitWebProcessedOutput(output string, maxChars int) string {
	if maxChars <= 0 {
		return output
	}
	runes := []rune(output)
	if len(runes) <= maxChars {
		return output
	}
	return string(runes[:maxChars])
}

// singlePassProcess handles content below the chunk threshold.
func (p *ChunkedWebContentProcessor) singlePassProcess(ctx context.Context, req WebContentProcessRequest) (string, error) {
	stream, err := p.client.OpenStream(ctx, llm.ChatRequest{
		Model:  p.model,
		Stream: true,
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: "You are an expert content analyst. Produce a concise markdown summary that preserves key facts, figures, quotes, code snippets, and actionable details.",
			},
			{
				Role: "user",
				Content: "Title: " + req.Title + "\nSource: " + req.URL +
					"\n\nCONTENT TO PROCESS:\n" + req.Content +
					"\n\nCreate a well-organized markdown summary. Preserve important specifics and avoid introductions.",
			},
		},
		MaxTokens: 20000,
	})
	if err != nil {
		return "", err
	}
	defer stream.Close() //nolint:errcheck // best-effort close

	return p.readStreamToString(ctx, stream)
}

// chunkedProcess handles large content by chunking, parallel summarization, and synthesis.
func (p *ChunkedWebContentProcessor) chunkedProcess(ctx context.Context, req WebContentProcessRequest) (string, error) {
	// Split content into chunks
	chunks := p.splitIntoChunks(req.Content)
	if len(chunks) == 0 {
		return req.Content, nil
	}

	// Summarize chunks in parallel
	summaries, err := p.summarizeChunks(ctx, req, chunks)
	if err != nil {
		return "", err
	}

	if err := requireSuccessfulChunkSummaries(summaries); err != nil {
		return "", err
	}

	// If only one chunk succeeded, return it directly
	if len(summaries) == 1 {
		return summaries[0].summary, nil
	}

	// If multiple chunks, synthesize them
	return p.synthesizeSummaries(ctx, req, summaries)
}

func requireSuccessfulChunkSummaries(summaries []chunkSummary) error {
	if len(summaries) == 0 {
		return errors.New("no summaries available")
	}
	return nil
}

// chunkSummary holds the result of summarizing one chunk.
type chunkSummary struct {
	index   int
	summary string
	err     error
}

// splitIntoChunks splits content into chunks of approximately ChunkSize.
// It tries to split at sentence boundaries for natural chunking.
func (p *ChunkedWebContentProcessor) splitIntoChunks(content string) []string {
	if len(content) <= p.cfg.ChunkSize {
		return []string{content}
	}

	var chunks []string
	remaining := content

	for len(remaining) > p.cfg.ChunkSize {
		// Find the actual split point
		splitAt := findGoodSplitPoint(remaining, p.cfg.ChunkSize)

		// Take the chunk
		chunks = append(chunks, remaining[:splitAt])

		// Move to remaining
		remaining = remaining[splitAt:]

		// Skip leading whitespace for next chunk to avoid orphaned whitespace
		remaining = strings.TrimLeft(remaining, " \n\t\r")
	}

	// Don't forget the last chunk
	if len(remaining) > 0 {
		chunks = append(chunks, remaining)
	}

	return chunks
}

// findGoodSplitPoint finds a good place to split content near the target size.
// It prefers splitting at sentence boundaries (period followed by space/newline)
// that are close to targetSize. It searches forward from targetSize, then
// backward if no good point is found forward.
func findGoodSplitPoint(content string, targetSize int) int {
	contentLen := len(content)
	if contentLen <= targetSize {
		return contentLen
	}

	// Define search window around targetSize
	// First, look forward from targetSize (preferring content after targetSize)
	forwardEnd := targetSize + 2000
	if forwardEnd > contentLen {
		forwardEnd = contentLen
	}

	// Search FORWARD from targetSize for a sentence boundary
	for i := targetSize; i < forwardEnd; i++ {
		if content[i] == '.' && i+1 < contentLen {
			c := content[i+1]
			if c == ' ' || c == '\n' || c == '\t' {
				return i + 1
			}
		}
	}

	// Also check if there's a good word boundary (space) near targetSize going forward
	for i := targetSize; i < forwardEnd && i < contentLen; i++ {
		if content[i] == ' ' {
			// Good space boundary found after targetSize
			return i
		}
	}

	// If no good forward split, look backward from targetSize
	backwardStart := targetSize - 2000
	if backwardStart < 0 {
		backwardStart = 0
	}

	// Search BACKWARD from targetSize for a sentence boundary
	for i := targetSize - 1; i >= backwardStart; i-- {
		if content[i] == '.' && i+1 < contentLen {
			c := content[i+1]
			if c == ' ' || c == '\n' || c == '\t' {
				return i + 1
			}
		}
	}

	// Search backward for a space (word boundary)
	for i := targetSize - 1; i >= backwardStart; i-- {
		if content[i] == ' ' {
			return i
		}
	}

	// Fallback: split at targetSize (or end if content is shorter)
	if targetSize < contentLen {
		return targetSize
	}
	return contentLen
}

// summarizeChunks processes chunks in parallel and returns successful summaries.
func (p *ChunkedWebContentProcessor) summarizeChunks(ctx context.Context, req WebContentProcessRequest, chunks []string) ([]chunkSummary, error) {
	results := make(chan chunkSummary, len(chunks))
	var wg sync.WaitGroup

	// Use a semaphore to limit concurrency
	sem := make(chan struct{}, p.cfg.MaxParallelism)

	for i, chunk := range chunks {
		wg.Add(1)
		go func(index int, content string) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			summary, err := p.summarizeChunk(ctx, req, index, content)
			results <- chunkSummary{index: index, summary: summary, err: err}
		}(i, chunk)
	}

	// Close results when done
	go func() {
		wg.Wait()
		close(results)
	}()

	var summaries []chunkSummary
	for r := range results {
		if r.err == nil && r.summary != "" {
			summaries = append(summaries, r)
		}
	}

	return summaries, nil
}

// summarizeChunk produces a summary for one chunk using a chunk-specific prompt.
func (p *ChunkedWebContentProcessor) summarizeChunk(ctx context.Context, req WebContentProcessRequest, chunkIndex int, chunkContent string) (string, error) {
	stream, err := p.client.OpenStream(ctx, llm.ChatRequest{
		Model:  p.model,
		Stream: true,
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: "You are an expert content analyst processing a SECTION of a larger document. Do NOT write introductions or conclusions. Focus on extracting ALL key facts, figures, data points. Preserve quotes and code snippets verbatim. Use bullet points.",
			},
			{
				Role: "user",
				Content: "Title: " + req.Title + "\nSource: " + req.URL +
					"\n\nSECTION (" + itoa(chunkIndex+1) + ") TO PROCESS:\n" + chunkContent +
					"\n\nExtract all key information from this section. Do not add preamble.",
			},
		},
		MaxTokens: 20000,
	})
	if err != nil {
		return "", err
	}
	defer stream.Close() //nolint:errcheck // best-effort close

	return p.readStreamToString(ctx, stream)
}

// synthesizeSummaries combines multiple chunk summaries into one cohesive summary.
func (p *ChunkedWebContentProcessor) synthesizeSummaries(ctx context.Context, req WebContentProcessRequest, summaries []chunkSummary) (string, error) {
	// Build the combined summaries text, sorted by chunk index
	sorted := make([]chunkSummary, len(summaries))
	copy(sorted, summaries)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].index < sorted[i].index {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	var summaryTexts []string
	for _, s := range sorted {
		summaryTexts = append(summaryTexts, s.summary)
	}

	combined := strings.Join(summaryTexts, "\n\n---\n\n")

	stream, err := p.client.OpenStream(ctx, llm.ChatRequest{
		Model:  p.model,
		Stream: true,
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: "You have been given summaries of different sections of a large document. Synthesize these into ONE cohesive, comprehensive summary that removes redundancy, preserves all key facts, and is well-organized with clear structure.",
			},
			{
				Role: "user",
				Content: "Title: " + req.Title + "\nSource: " + req.URL +
					"\n\nSECTION SUMMARIES TO SYNTHESIZE:\n" + combined +
					"\n\nCreate one unified, well-organized summary that integrates all section summaries. Remove redundancy and preserve all important information.",
			},
		},
		MaxTokens: 20000,
	})
	if err != nil {
		// Fallback: concatenate summaries if synthesis fails
		return p.fallbackSummaries(summaries)
	}
	defer stream.Close() //nolint:errcheck // best-effort close

	result, err := p.readStreamToString(ctx, stream)
	if err != nil {
		return p.fallbackSummaries(summaries)
	}
	return result, nil
}

// fallbackSummaries concatenates chunk summaries when synthesis fails.
// Input summaries are sorted by index before concatenation.
func (p *ChunkedWebContentProcessor) fallbackSummaries(summaries []chunkSummary) (string, error) {
	if len(summaries) == 0 {
		return "", errors.New("no summaries available")
	}

	// Sort by original chunk index
	sorted := make([]chunkSummary, len(summaries))
	copy(sorted, summaries)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].index < sorted[i].index {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	var b strings.Builder
	for i, s := range sorted {
		if i > 0 {
			b.WriteString("\n\n---\n\n")
		}
		b.WriteString(s.summary)
	}
	return b.String(), nil
}

// readStreamToString reads all tokens from a stream and returns them as a string.
func (p *ChunkedWebContentProcessor) readStreamToString(ctx context.Context, stream llm.Stream) (string, error) {
	var out strings.Builder
	for {
		ev, err := stream.Recv(ctx)
		if err != nil {
			if err == io.EOF && out.Len() > 0 {
				return out.String(), nil
			}
			return "", err
		}
		switch ev.Kind {
		case llm.EventToken:
			out.WriteString(ev.Token)
		case llm.EventDone:
			return out.String(), nil
		}
	}
}

// itoa converts an int to a string without importing strconv.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}
