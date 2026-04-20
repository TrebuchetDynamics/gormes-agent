package memory

import (
	"context"
	"log/slog"
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

// loopOnce is the stub; Task 6 replaces it with the real body.
func (e *Extractor) loopOnce(ctx context.Context) {
	// Intentionally empty — poll is a no-op until T6.
}
