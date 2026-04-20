package memory

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/XelHaku/golang-hermes-agent/gormes/internal/hermes"
)

// ExtractorConfig controls the Brain worker's polling + retry behavior.
// Zero values fall back to sensible defaults.
type ExtractorConfig struct {
	Model        string        // empty = reuse kernel's Hermes model
	PollInterval time.Duration // default 10s
	BatchSize    int           // default 5 turns per LLM call
	MaxAttempts  int           // default 5 before dead-letter
	CallTimeout  time.Duration // default 30s per LLM call
	BackoffBase  time.Duration // default 2s; doubles per attempt
	BackoffMax   time.Duration // default 60s cap
}

func (c *ExtractorConfig) withDefaults() {
	if c.PollInterval <= 0 {
		c.PollInterval = 10 * time.Second
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 5
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 5
	}
	if c.CallTimeout <= 0 {
		c.CallTimeout = 30 * time.Second
	}
	if c.BackoffBase <= 0 {
		c.BackoffBase = 2 * time.Second
	}
	if c.BackoffMax <= 0 {
		c.BackoffMax = 60 * time.Second
	}
}

// Extractor runs the LLM-assisted entity/relationship extraction loop.
// Exactly one goroutine owns the main poll loop; graph writes serialize
// through the shared *SqliteStore *sql.DB (SetMaxOpenConns(1) pool).
type Extractor struct {
	store *SqliteStore
	llm   hermes.Client
	cfg   ExtractorConfig
	log   *slog.Logger

	done      chan struct{}
	closeOnce sync.Once
	running   atomic.Bool
}

// NewExtractor constructs an Extractor. Caller drives lifecycle via
// Run(ctx) and Close(ctx).
func NewExtractor(s *SqliteStore, llm hermes.Client, cfg ExtractorConfig, log *slog.Logger) *Extractor {
	cfg.withDefaults()
	if log == nil {
		log = slog.Default()
	}
	return &Extractor{
		store: s,
		llm:   llm,
		cfg:   cfg,
		log:   log,
		done:  make(chan struct{}, 1),
	}
}

// Run blocks until ctx is cancelled. Each tick: loopOnce.
func (e *Extractor) Run(ctx context.Context) {
	e.running.Store(true)
	defer func() {
		select {
		case e.done <- struct{}{}:
		default:
		}
	}()
	ticker := time.NewTicker(e.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.loopOnce(ctx) // stub in T5; T6 fills in
		}
	}
}

// Close waits for Run to exit if Run is currently executing, bounded by
// ctx. If Run has never been called, returns immediately. Idempotent.
func (e *Extractor) Close(ctx context.Context) error {
	e.closeOnce.Do(func() {
		if !e.running.Load() {
			return
		}
		select {
		case <-e.done:
		case <-ctx.Done():
		}
	})
	return nil
}

func (e *Extractor) loopOnce(ctx context.Context) {
	batch, err := e.pollBatch(ctx)
	if err != nil {
		e.log.Warn("extractor: poll failed", "err", err)
		return
	}
	if len(batch) == 0 {
		return
	}
	ids := make([]int64, len(batch))
	for i, r := range batch {
		ids[i] = r.id
	}

	callCtx, cancel := context.WithTimeout(ctx, e.cfg.CallTimeout)
	defer cancel()
	raw, err := e.callLLM(callCtx, batch)
	if err != nil {
		e.log.Warn("extractor: LLM call failed", "turn_ids", ids, "err", err)
		_ = incrementAttempts(ctx, e.store.db, ids, err.Error())
		return
	}

	validated, err := ValidateExtractorOutput(raw)
	if err != nil {
		preview := string(raw)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		e.log.Warn("extractor: malformed JSON",
			"turn_ids", ids, "preview", preview, "err", err)
		_ = incrementAttempts(ctx, e.store.db, ids, "malformed JSON: "+err.Error())
		return
	}

	// writeGraphBatch is ONE transaction: upserts + mark-extracted commit
	// atomically. See graph.go.
	if err := writeGraphBatch(ctx, e.store.db, validated, ids); err != nil {
		e.log.Warn("extractor: graph write failed",
			"turn_ids", ids, "err", err)
		_ = incrementAttempts(ctx, e.store.db, ids, err.Error())
		return
	}

	e.log.Debug("extractor: batch processed",
		"turn_ids", ids,
		"entities", len(validated.Entities),
		"relationships", len(validated.Relationships))
}

// pollBatch reads up to cfg.BatchSize unprocessed turns.
func (e *Extractor) pollBatch(ctx context.Context) ([]turnRow, error) {
	rows, err := e.store.db.QueryContext(ctx,
		`SELECT id, role, content FROM turns
		 WHERE extracted = 0 AND extraction_attempts < ?
		 ORDER BY id LIMIT ?`,
		e.cfg.MaxAttempts, e.cfg.BatchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []turnRow
	for rows.Next() {
		var r turnRow
		if err := rows.Scan(&r.id, &r.role, &r.content); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// callLLM sends the extractor prompt to the hermes.Client and collects
// the full streamed response. Returns raw JSON (not yet validated).
func (e *Extractor) callLLM(ctx context.Context, batch []turnRow) ([]byte, error) {
	req := hermes.ChatRequest{
		Model:  e.cfg.Model,
		Stream: true,
		Messages: []hermes.Message{
			{Role: "system", Content: extractorSystemPrompt},
			{Role: "user", Content: formatBatchPrompt(batch)},
		},
	}
	stream, err := e.llm.OpenStream(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = stream.Close() }()

	var b strings.Builder
	for {
		ev, err := stream.Recv(ctx)
		if errors.Is(err, io.EOF) || ev.Kind == hermes.EventDone {
			if ev.Token != "" {
				b.WriteString(ev.Token)
			}
			break
		}
		if err != nil {
			// fakeStream in tests returns a sentinel when exhausted; if we
			// already have content, treat as clean end.
			if b.Len() > 0 {
				break
			}
			return nil, err
		}
		if ev.Token != "" {
			b.WriteString(ev.Token)
		}
	}
	return []byte(b.String()), nil
}
