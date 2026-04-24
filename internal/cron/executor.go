package cron

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	hermesclient "github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

// KernelAPI is the narrow slice of *kernel.Kernel the Executor needs.
// Defined as an interface here so tests can swap in a fake without
// importing the full kernel package's internals.
type KernelAPI interface {
	Submit(e kernel.PlatformEvent) error
	Render() <-chan kernel.RenderFrame
}

// Runner is the narrow interface the Scheduler uses to fire a job.
// The real *Executor satisfies it; tests inject fakes.
type Runner interface {
	Run(ctx context.Context, job Job)
}

// ExecutorConfig is the set of live dependencies. Callers construct it
// once at startup (cmd/gormes/telegram.go) and pass the same Executor
// to the Scheduler.
type ExecutorConfig struct {
	Kernel      KernelAPI
	JobStore    *Store
	RunStore    *RunStore
	Sink        DeliverySink
	CallTimeout time.Duration // default 60s when zero
}

func (c *ExecutorConfig) withDefaults() {
	if c.CallTimeout <= 0 {
		c.CallTimeout = 60 * time.Second
	}
}

// Executor bridges a scheduler tick into the kernel and records what
// happened.
type Executor struct {
	cfg ExecutorConfig
	log *slog.Logger
}

// NewExecutor constructs a ready-to-use Executor. Pass nil for log to
// use slog.Default().
func NewExecutor(cfg ExecutorConfig, log *slog.Logger) *Executor {
	cfg.withDefaults()
	if log == nil {
		log = slog.Default()
	}
	return &Executor{cfg: cfg, log: log}
}

// Run fires one job end-to-end. Blocks until the turn completes or
// times out. Safe to call concurrently (the kernel serialises via its
// mailbox).
func (e *Executor) Run(ctx context.Context, job Job) {
	startedAt := time.Now().Unix()
	sessionID := fmt.Sprintf("cron:%s:%d", job.ID, startedAt)
	promptHash := shortHash(job.Prompt)

	// Subscribe BEFORE Submit so we don't miss the final frame.
	frames := e.cfg.Kernel.Render()
	done := make(chan string, 1) // receives the final assistant text
	callCtx, cancel := context.WithTimeout(ctx, e.cfg.CallTimeout)
	defer cancel()
	go func() {
		for {
			select {
			case f, ok := <-frames:
				if !ok {
					return
				}
				if f.SessionID != sessionID {
					continue
				}
				if f.Phase != kernel.PhaseIdle {
					continue
				}
				// Find the last assistant message in History.
				text := lastAssistantText(f.History)
				select {
				case done <- text:
				default:
				}
				return
			case <-callCtx.Done():
				return
			}
		}
	}()

	// Submit.
	submitErr := e.cfg.Kernel.Submit(kernel.PlatformEvent{
		Kind:      kernel.PlatformEventSubmit,
		Text:      BuildPrompt(job.Prompt),
		SessionID: sessionID,
		CronJobID: job.ID,
	})
	if submitErr != nil {
		run := Run{
			JobID:      job.ID,
			StartedAt:  startedAt,
			FinishedAt: time.Now().Unix(),
			PromptHash: promptHash,
			Status:     "error",
			Delivered:  false,
			ErrorMsg:   submitErr.Error(),
		}
		e.recordAndUpdateJob(ctx, job, run)
		return
	}

	// Wait for final text or timeout.
	var finalText string
	select {
	case finalText = <-done:
	case <-callCtx.Done():
		// Timeout — deliver a short failure notice.
		notice := fmt.Sprintf("Cron job %s timed out after %s.", job.Name, e.cfg.CallTimeout)
		_ = e.cfg.Sink.Deliver(context.Background(), notice)
		run := Run{
			JobID:         job.ID,
			StartedAt:     startedAt,
			FinishedAt:    time.Now().Unix(),
			PromptHash:    promptHash,
			Status:        "timeout",
			Delivered:     true,
			OutputPreview: truncate(notice, 200),
			ErrorMsg:      "context deadline exceeded",
		}
		e.recordAndUpdateJob(ctx, job, run)
		return
	}

	finished := time.Now().Unix()

	// [SILENT] suppression?
	if DetectSilent(finalText) {
		run := Run{
			JobID:             job.ID,
			StartedAt:         startedAt,
			FinishedAt:        finished,
			PromptHash:        promptHash,
			Status:            "suppressed",
			Delivered:         false,
			SuppressionReason: "silent",
		}
		e.recordAndUpdateJob(ctx, job, run)
		return
	}

	// Empty response? Deliver failure notice.
	if isEmpty(finalText) {
		notice := fmt.Sprintf("Cron job %s returned empty output.", job.Name)
		_ = e.cfg.Sink.Deliver(context.Background(), notice)
		run := Run{
			JobID:             job.ID,
			StartedAt:         startedAt,
			FinishedAt:        finished,
			PromptHash:        promptHash,
			Status:            "error",
			Delivered:         true,
			SuppressionReason: "empty",
			OutputPreview:     truncate(notice, 200),
			ErrorMsg:          "agent returned empty response",
		}
		e.recordAndUpdateJob(ctx, job, run)
		return
	}

	// Normal delivery.
	delivErr := e.cfg.Sink.Deliver(context.Background(), finalText)
	run := Run{
		JobID:         job.ID,
		StartedAt:     startedAt,
		FinishedAt:    finished,
		PromptHash:    promptHash,
		Status:        "success",
		Delivered:     delivErr == nil,
		OutputPreview: truncate(finalText, 200),
	}
	if delivErr != nil {
		run.ErrorMsg = fmt.Sprintf("delivery: %v", delivErr)
	}
	e.recordAndUpdateJob(ctx, job, run)
}

func (e *Executor) recordAndUpdateJob(ctx context.Context, job Job, run Run) {
	if err := e.cfg.RunStore.RecordRun(ctx, run); err != nil {
		e.log.Warn("cron: failed to record run", "job_id", job.ID, "err", err)
	}
	job.LastRunUnix = run.StartedAt
	job.LastStatus = run.Status
	if err := e.cfg.JobStore.Update(job); err != nil {
		e.log.Warn("cron: failed to update job after run", "job_id", job.ID, "err", err)
	}
}

// lastAssistantText walks history backwards and returns the first
// assistant message's content. Empty string when no assistant message
// exists (shouldn't happen in practice).
func lastAssistantText(history []hermesclient.Message) string {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "assistant" {
			return history[i].Content
		}
	}
	return ""
}

func shortHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8]) // 16-char prefix
}

func isEmpty(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
	}
	return true
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
