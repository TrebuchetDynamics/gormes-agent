package cron

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/TrebuchetDynamics/gormes-agent/internal/core/subagent"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	hermesclient "github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// KernelAPI is the narrow slice of *kernel.Kernel the Executor needs.
// Defined as an interface here so tests can swap in a fake without
// importing the full kernel package's internals.
type KernelAPI interface {
	Submit(e kernel.PlatformEvent) error
	// Subscribe returns an independent render stream so concurrent cron runs (and
	// the gateway) sharing one kernel do not steal each other's frames.
	Subscribe() (<-chan kernel.RenderFrame, func())
	// ConfigModel returns the model name baked into the kernel at construction
	// time, used for pre-flight model-resolution checks in cron.
	ConfigModel() string
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
	Kernel        KernelAPI
	JobStore      *Store
	RunStore      *RunStore
	DurableLedger *subagent.DurableLedger
	Sink          DeliverySink
	LiveDelivery  LiveDeliveryAdapter
	Directory     DeliveryTargetDirectory
	CallTimeout   time.Duration // default 60s when zero
	// CronApprovalMode is the Hermes approvals.cron_mode value applied to
	// tool execution for cron-fired kernel turns. Empty defaults to deny.
	CronApprovalMode string

	// SubprocessKiller is the optional reaper used by the per-run release
	// ledger to terminate tool subprocesses registered during the kernel
	// turn. When nil, registered PIDs are recorded as evidence with a
	// "no killer supplied" field instead of being killed.
	SubprocessKiller SubprocessKiller

	// RegisterRelease is the optional registration seam for the kernel /
	// inner runner. It is called once at the start of each Run with the
	// per-run ledger and the active job, so callers can register session
	// DB io.Closers, HTTP idle closables, and tool subprocess PIDs that
	// should be reaped at run end. Nil is allowed (Run will record a
	// cron_release_skipped_no_resource evidence entry).
	RegisterRelease func(ctx context.Context, ledger *RunReleaseLedger, job Job)

	// ScriptRunner executes an optional pre-run script and returns redacted
	// evidence for prompt injection. Nil uses ShellCronScriptRunner.
	ScriptRunner  CronScriptRunner
	ScriptsRoot   string
	ScriptTimeout time.Duration

	// InactivityTimeout bounds idle provider/kernel streams. Zero is resolved
	// from HERMES_CRON_TIMEOUT with the Hermes default of 10m; negative disables
	// the idle timer.
	InactivityTimeout time.Duration
	LookupEnv         func(string) string

	// Codex401Refresher is an injected auth-refresh seam for openai-codex cron
	// runs. When it returns true after a 401 Submit error, the executor retries
	// the same event once.
	Codex401Refresher func(context.Context, Job, error) (bool, error)

	// OperatorReportHome is the root used for durable OperatorRunReport
	// artifacts. Empty disables report writes unless GORMES_HOME is set.
	OperatorReportHome string
}

func (c *ExecutorConfig) withDefaults() {
	if c.CallTimeout <= 0 {
		c.CallTimeout = 60 * time.Second
	}
	if strings.TrimSpace(c.CronApprovalMode) == "" {
		c.CronApprovalMode = tools.CronApprovalModeDeny
	}
	if c.ScriptTimeout <= 0 {
		c.ScriptTimeout = 30 * time.Second
	}
	if strings.TrimSpace(c.ScriptsRoot) == "" {
		c.ScriptsRoot = defaultCronScriptsRoot()
	}
	if c.LookupEnv == nil {
		c.LookupEnv = os.Getenv
	}
	if c.InactivityTimeout == 0 {
		if timeout, enabled := ResolveCronInactivityTimeout(c.LookupEnv); enabled {
			c.InactivityTimeout = timeout
		} else {
			c.InactivityTimeout = -1
		}
	}
	if strings.TrimSpace(c.OperatorReportHome) == "" {
		c.OperatorReportHome = strings.TrimSpace(c.LookupEnv("GORMES_HOME"))
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
// mailbox). Discards the per-run release evidence; callers that need
// it should use RunWithRelease.
func (e *Executor) Run(ctx context.Context, job Job) {
	_, _ = e.RunWithRelease(ctx, job)
}

// RunWithRelease fires one job end-to-end and returns the per-run release
// ledger evidence and the kernel-side error (if any) from the turn. All
// four exit paths — success, kernel Submit error, parent ctx cancel, and
// CallTimeout — go through the same deferred ledger.Release so registered
// session DBs, HTTP idle closables, and tool subprocess PIDs are reaped
// exactly once. Release errors are recorded on the executor log and shown
// in the returned evidence, but they do not mask the kernel error
// returned to the caller.
func (e *Executor) RunWithRelease(ctx context.Context, job Job) (evidence []ReleaseEvidence, kernelErr error) {
	ledger := NewRunReleaseLedger()
	if e.cfg.RegisterRelease != nil {
		e.cfg.RegisterRelease(ctx, ledger, job)
	}
	defer func() {
		releaseEvidence, releaseErr := ledger.Release(e.cfg.SubprocessKiller)
		evidence = releaseEvidence
		if releaseErr != nil {
			e.log.Warn("cron: release ledger reported errors",
				"job_id", job.ID, "err", releaseErr)
		}
	}()

	kernelErr = e.runOneTurn(ctx, job)
	return
}

// runOneTurn executes the kernel turn and the post-turn delivery /
// recording bookkeeping. It returns the kernel-side error (Submit error,
// timeout, or ctx-cancel) so RunWithRelease can surface it to callers
// alongside the release evidence; success paths return nil.
func (e *Executor) runOneTurn(ctx context.Context, job Job) error {
	runtimeJob, envEvidence := ExpandCronJobEnvRefs(job, e.cfg.LookupEnv)
	if len(envEvidence) > 0 {
		e.log.Warn("cron: unresolved env refs", "job_id", job.ID, "evidence", envEvidence)
	}
	// Fail fast if no model can be resolved: per-job model (after env expansion)
	// takes precedence; if empty, the kernel falls back to its config model. An
	// empty model flowing to the provider surfaces as an opaque HTTP 400 —
	// mirror Hermes #43899 / fix(cron): resolve model.default + fail fast.
	if strings.TrimSpace(runtimeJob.Model) == "" && strings.TrimSpace(e.cfg.Kernel.ConfigModel()) == "" {
		return &cronNoModelError{JobID: job.ID, JobName: job.Name}
	}
	startedAt := time.Now().Unix()
	promptHash := shortHash(job.Prompt)
	restoreCWD := e.applyWorkdir(runtimeJob)
	defer restoreCWD()
	if runtimeJob.NoAgent {
		return e.runNoAgentJob(ctx, runtimeJob, startedAt, promptHash)
	}
	sessionID := fmt.Sprintf("cron:%s:%d", job.ID, startedAt)
	const durableWorker = "cron-executor"
	durableActive := e.startDurableCronRun(ctx, sessionID, job, promptHash, durableWorker)

	// Subscribe BEFORE Submit so we don't miss the final frame. An independent
	// subscription (not the shared Render channel) keeps this run's frames from
	// being stolen by the gateway or a sibling cron job on the same kernel.
	frames, unsubscribe := e.cfg.Kernel.Subscribe()
	defer unsubscribe()
	done := make(chan string, 1) // receives the final assistant text
	activity := make(chan kernel.RenderFrame, 1)
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
				select {
				case activity <- f:
				default:
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

	// Build and scan the fully assembled runtime prompt before Submit. This
	// mirrors Hermes' CronPromptInjectionBlocked branch: create/update-time scans
	// do not cover runtime context_from or pre-run script output.
	promptText := e.buildPromptForJob(ctx, runtimeJob)
	if finding, blocked := ScanPromptForCronThreat(promptText); blocked {
		blockErr := cronPromptInjectionBlockedError(finding)
		blockedDoc := cronPromptBlockedDocument(runtimeJob, startedAt, finding)
		blockedNotice := cronPromptBlockedDelivery(runtimeJob, finding)
		_ = e.cfg.Sink.Deliver(context.Background(), blockedNotice)
		run := Run{
			JobID:         job.ID,
			StartedAt:     startedAt,
			FinishedAt:    time.Now().Unix(),
			PromptHash:    promptHash,
			Status:        "error",
			Delivered:     true,
			OutputPreview: truncate(blockedDoc, 200),
			ErrorMsg:      blockErr.Error(),
		}
		e.failDurableCronRun(sessionID, durableWorker, durableActive, run.ErrorMsg)
		e.recordAndUpdateJob(ctx, runtimeJob, run, operatorReportContext{SessionID: sessionID})
		return blockErr
	}

	// Submit.
	event := kernel.PlatformEvent{
		Kind:             kernel.PlatformEventSubmit,
		Text:             promptText,
		SessionID:        sessionID,
		Model:            runtimeJob.Model,
		CronJobID:        job.ID,
		CronApprovalMode: e.cfg.CronApprovalMode,
	}
	submitErr := e.submitCronEvent(callCtx, runtimeJob, event)
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
		e.failDurableCronRun(sessionID, durableWorker, durableActive, run.ErrorMsg)
		e.recordAndUpdateJob(ctx, runtimeJob, run, operatorReportContext{SessionID: sessionID})
		return submitErr
	}

	// Wait for final text or timeout.
	var finalText string
	lastActivity := time.Now()
	lastActivityDesc := "submit"
	var inactivityTimer *time.Timer
	var inactivityC <-chan time.Time
	if e.cfg.InactivityTimeout > 0 {
		inactivityTimer = time.NewTimer(e.cfg.InactivityTimeout)
		defer inactivityTimer.Stop()
		inactivityC = inactivityTimer.C
	}
	for {
		select {
		case finalText = <-done:
			goto completed
		case frame := <-activity:
			lastActivity = time.Now()
			lastActivityDesc = frame.Phase.String()
			if inactivityTimer != nil {
				resetTimer(inactivityTimer, e.cfg.InactivityTimeout)
			}
		case <-inactivityC:
			err := cronInactivityError(job, time.Since(lastActivity), e.cfg.InactivityTimeout, lastActivityDesc)
			notice := "Cron job " + job.Name + " " + err.Error() + "."
			_ = e.cfg.Sink.Deliver(context.Background(), notice)
			run := Run{
				JobID:         job.ID,
				StartedAt:     startedAt,
				FinishedAt:    time.Now().Unix(),
				PromptHash:    promptHash,
				Status:        "timeout",
				Delivered:     true,
				OutputPreview: truncate(notice, 200),
				ErrorMsg:      err.Error(),
			}
			e.failDurableCronRun(sessionID, durableWorker, durableActive, run.ErrorMsg)
			e.recordAndUpdateJob(ctx, runtimeJob, run, operatorReportContext{SessionID: sessionID})
			return err
		case <-callCtx.Done():
			// Timeout or parent ctx cancel — deliver a short failure notice.
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
			e.failDurableCronRun(sessionID, durableWorker, durableActive, run.ErrorMsg)
			e.recordAndUpdateJob(ctx, runtimeJob, run, operatorReportContext{SessionID: sessionID})
			return callCtx.Err()
		}
	}

completed:
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
		e.completeDurableCronRun(sessionID, durableWorker, durableActive, run)
		e.recordAndUpdateJob(ctx, runtimeJob, run, operatorReportContext{SessionID: sessionID})
		return nil
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
		e.failDurableCronRun(sessionID, durableWorker, durableActive, run.ErrorMsg)
		e.recordAndUpdateJob(ctx, runtimeJob, run, operatorReportContext{SessionID: sessionID})
		return nil
	}

	// Normal delivery.
	content := PrepareCronDeliveryContent(finalText)
	deliveryPlan := PlanCronDeliveryForJob(runtimeJob, e.cfg.Directory)
	outcome := DeliverCronDeliveryPlan(
		context.Background(),
		deliveryPlan,
		content,
		e.cfg.LiveDelivery,
		e.cfg.Sink,
	)
	run := Run{
		JobID:         job.ID,
		StartedAt:     startedAt,
		FinishedAt:    finished,
		PromptHash:    promptHash,
		Status:        "success",
		OutputPreview: truncate(content.Text, 200),
	}
	run = applyDeliveryOutcome(run, outcome)
	e.completeDurableCronRun(sessionID, durableWorker, durableActive, run)
	e.recordAndUpdateJob(ctx, runtimeJob, run, operatorReportContext{SessionID: sessionID, DeliveryPlan: deliveryPlan, DeliveryOutcome: outcome})
	return nil
}

// cronNoModelError is returned by runOneTurn when neither the per-job model
// nor the kernel's config model resolves to a non-empty string. This surfaces
// an actionable error instead of letting an empty model reach the provider as
// an opaque HTTP 400 (Hermes #43899 parity).
type cronNoModelError struct {
	JobID   string
	JobName string
}

func (e *cronNoModelError) Error() string {
	name := e.JobName
	if name == "" {
		name = e.JobID
	}
	return fmt.Sprintf(
		"cron_no_model: job %q has no model configured (job.model is empty, "+
			"GORMES_MODEL env is unset, and the runtime config has no default model). "+
			"Set a per-job model via `gormes cronjob update job_id=%s model=<name>` "+
			"or configure a default with `gormes model <name>`.",
		name, e.JobID,
	)
}

const CronPromptInjectionBlockedCode = "cron_prompt_injection_blocked"

func cronPromptInjectionBlockedError(finding CronSafetyFinding) error {
	return fmt.Errorf("%s: %s", CronPromptInjectionBlockedCode, cronPromptScannerResult(finding))
}

func cronPromptBlockedDocument(job Job, startedAt int64, finding CronSafetyFinding) string {
	return fmt.Sprintf(
		"# Cron Job: %s\n\n"+
			"**Job ID:** %s\n"+
			"**Run Time:** %s\n"+
			"**Status:** BLOCKED\n\n"+
			"The assembled prompt (job prompt + runtime context) tripped the cron injection scanner and the agent was NOT run.\n\n"+
			"**Scanner result:** %s\n\n"+
			"Audit the job prompt, pre-run script output, context_from source jobs, and attached skill content for prompt-injection payloads or invisible-unicode markers.",
		job.Name,
		job.ID,
		time.Unix(startedAt, 0).Format("2006-01-02 15:04:05"),
		cronPromptScannerResult(finding),
	)
}

func cronPromptBlockedDelivery(job Job, finding CronSafetyFinding) string {
	return fmt.Sprintf(
		"⚠️ Cron job %q failed:\n%s\n\n**Status:** BLOCKED\nThe assembled prompt tripped the cron injection scanner and the agent was NOT run.\n\n**Scanner result:** %s",
		job.Name,
		CronPromptInjectionBlockedCode,
		cronPromptScannerResult(finding),
	)
}

func cronPromptScannerResult(finding CronSafetyFinding) string {
	if finding.Code == "invisible_unicode" && strings.TrimSpace(finding.Evidence) != "" {
		return fmt.Sprintf("Blocked: prompt contains invisible unicode %s (possible injection).", finding.Evidence)
	}
	if strings.TrimSpace(finding.ID) != "" {
		return fmt.Sprintf("Blocked: prompt matches threat pattern '%s'. Cron prompts must not contain injection or exfiltration payloads.", finding.ID)
	}
	if strings.TrimSpace(finding.Message) != "" {
		return "Blocked: " + finding.Message
	}
	return "Blocked: prompt failed cron safety scan."
}

func (e *Executor) runNoAgentJob(ctx context.Context, job Job, startedAt int64, promptHash string) error {
	finished := func() int64 { return time.Now().Unix() }
	if strings.TrimSpace(job.Script) == "" {
		err := errors.New("cron_no_agent_script_required")
		notice := fmt.Sprintf("Cron watchdog %q cannot run: no_agent=True requires a script.", job.Name)
		_ = e.cfg.Sink.Deliver(context.Background(), notice)
		e.recordAndUpdateJob(ctx, job, Run{
			JobID:         job.ID,
			StartedAt:     startedAt,
			FinishedAt:    finished(),
			PromptHash:    promptHash,
			Status:        "error",
			Delivered:     true,
			OutputPreview: truncate(notice, 200),
			ErrorMsg:      err.Error(),
		}, operatorReportContext{})
		return err
	}
	runner := e.cfg.ScriptRunner
	if runner == nil {
		runner = ShellCronScriptRunner{}
	}
	result := runner.RunCronScript(ctx, CronScriptRequest{
		Path:        job.Script,
		ScriptsRoot: e.cfg.ScriptsRoot,
		Workdir:     job.Workdir,
		Timeout:     e.cfg.ScriptTimeout,
	})
	output := strings.TrimSpace(result.Output)
	if result.Success {
		switch {
		case output == "":
			e.recordAndUpdateJob(ctx, job, Run{
				JobID:             job.ID,
				StartedAt:         startedAt,
				FinishedAt:        finished(),
				PromptHash:        promptHash,
				Status:            "suppressed",
				Delivered:         false,
				SuppressionReason: "empty",
			}, operatorReportContext{})
			return nil
		case cronNoAgentWakeAgentFalse(output):
			e.recordAndUpdateJob(ctx, job, Run{
				JobID:             job.ID,
				StartedAt:         startedAt,
				FinishedAt:        finished(),
				PromptHash:        promptHash,
				Status:            "suppressed",
				Delivered:         false,
				SuppressionReason: "silent",
				OutputPreview:     truncate(output, 200),
			}, operatorReportContext{})
			return nil
		default:
			content := PrepareCronDeliveryContent(output)
			deliveryPlan := PlanCronDeliveryForJob(job, e.cfg.Directory)
			outcome := DeliverCronDeliveryPlan(
				context.Background(),
				deliveryPlan,
				content,
				e.cfg.LiveDelivery,
				e.cfg.Sink,
			)
			run := Run{
				JobID:         job.ID,
				StartedAt:     startedAt,
				FinishedAt:    finished(),
				PromptHash:    promptHash,
				Status:        "success",
				OutputPreview: truncate(content.Text, 200),
			}
			run = applyDeliveryOutcome(run, outcome)
			e.recordAndUpdateJob(ctx, job, run, operatorReportContext{DeliveryPlan: deliveryPlan, DeliveryOutcome: outcome})
			return nil
		}
	}
	if output == "" {
		output = "script failed without output"
	}
	status := "error"
	if strings.Contains(strings.ToLower(output), "timed out") {
		status = "timeout"
	}
	notice := fmt.Sprintf("Cron watchdog %q script failed\n\n%s", job.Name, output)
	if status == "timeout" {
		notice = fmt.Sprintf("Cron watchdog %q script timed out\n\n%s", job.Name, output)
	}
	_ = e.cfg.Sink.Deliver(context.Background(), notice)
	e.recordAndUpdateJob(ctx, job, Run{
		JobID:         job.ID,
		StartedAt:     startedAt,
		FinishedAt:    finished(),
		PromptHash:    promptHash,
		Status:        status,
		Delivered:     true,
		OutputPreview: truncate(notice, 200),
		ErrorMsg:      output,
	}, operatorReportContext{})
	if status == "timeout" {
		return context.DeadlineExceeded
	}
	return errors.New(output)
}

func cronNoAgentWakeAgentFalse(output string) bool {
	var doc map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &doc); err != nil {
		return false
	}
	value, ok := doc["wakeAgent"]
	return ok && value == false
}

const CronEnvRefUnresolved = "cron_env_ref_unresolved"

type CronEnvRefEvidence struct {
	Code     string `json:"code"`
	Field    string `json:"field"`
	Variable string `json:"variable"`
}

func (e CronEnvRefEvidence) String() string {
	return e.Code + " field=" + e.Field + " variable=" + e.Variable
}

var cronEnvRefPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

func ExpandCronJobEnvRefs(job Job, lookup func(string) string) (Job, []CronEnvRefEvidence) {
	if lookup == nil {
		lookup = os.Getenv
	}
	out := job
	var evidence []CronEnvRefEvidence
	out.Model = expandCronEnvRefString("model", out.Model, lookup, &evidence)
	out.Provider = expandCronEnvRefString("provider", out.Provider, lookup, &evidence)
	out.Deliver = expandCronEnvRefString("deliver", out.Deliver, lookup, &evidence)
	out.Workdir = expandCronEnvRefString("workdir", out.Workdir, lookup, &evidence)
	out.Script = expandCronEnvRefString("script", out.Script, lookup, &evidence)
	if out.Origin != nil {
		origin := *out.Origin
		origin.Platform = expandCronEnvRefString("origin.platform", origin.Platform, lookup, &evidence)
		origin.ChatID = expandCronEnvRefString("origin.chat_id", origin.ChatID, lookup, &evidence)
		origin.ThreadID = expandCronEnvRefString("origin.thread_id", origin.ThreadID, lookup, &evidence)
		out.Origin = &origin
	}
	return out, evidence
}

func expandCronEnvRefString(field, value string, lookup func(string) string, evidence *[]CronEnvRefEvidence) string {
	if value == "" || !strings.Contains(value, "${") {
		return value
	}
	return cronEnvRefPattern.ReplaceAllStringFunc(value, func(token string) string {
		match := cronEnvRefPattern.FindStringSubmatch(token)
		if len(match) != 2 {
			return token
		}
		name := match[1]
		resolved := lookup(name)
		if resolved == "" {
			*evidence = append(*evidence, CronEnvRefEvidence{
				Code:     CronEnvRefUnresolved,
				Field:    field,
				Variable: name,
			})
			return token
		}
		return resolved
	})
}

func (e *Executor) buildPromptForJob(ctx context.Context, job Job) string {
	job.Prompt = e.promptWithScript(ctx, job)
	return BuildPromptForJob(ctx, job, e.cfg.RunStore, e.log)
}

func (e *Executor) promptWithScript(ctx context.Context, job Job) string {
	if strings.TrimSpace(job.Script) == "" {
		return job.Prompt
	}
	runner := e.cfg.ScriptRunner
	if runner == nil {
		runner = ShellCronScriptRunner{}
	}
	result := runner.RunCronScript(ctx, CronScriptRequest{
		Path:        job.Script,
		ScriptsRoot: e.cfg.ScriptsRoot,
		Workdir:     job.Workdir,
		Timeout:     e.cfg.ScriptTimeout,
	})
	output := strings.TrimSpace(result.Output)
	if result.Success {
		if output == "" {
			output = "script produced no output"
		}
		return "## Script Output\n" +
			"The following data was collected by a pre-run script. Use it as context for your analysis.\n\n" +
			"```\n" + output + "\n```\n\n" +
			job.Prompt
	}
	if output == "" {
		output = "script failed without output"
	}
	return "## Script Error\n" +
		"The data-collection script failed. Report this to the user.\n\n" +
		"```\n" + output + "\n```\n\n" +
		job.Prompt
}

func (e *Executor) applyWorkdir(job Job) func() {
	workdir := strings.TrimSpace(job.Workdir)
	if workdir == "" {
		return func() {}
	}
	info, err := os.Stat(workdir)
	if err != nil || !info.IsDir() {
		e.log.Warn("cron: configured workdir unavailable; running without it", "job_id", job.ID, "workdir", workdir, "err", err)
		return func() {}
	}
	prior, hadPrior := os.LookupEnv("TERMINAL_CWD")
	_ = os.Setenv("TERMINAL_CWD", workdir)
	return func() {
		if hadPrior {
			_ = os.Setenv("TERMINAL_CWD", prior)
			return
		}
		_ = os.Unsetenv("TERMINAL_CWD")
	}
}

func (e *Executor) submitCronEvent(ctx context.Context, job Job, event kernel.PlatformEvent) error {
	err := e.cfg.Kernel.Submit(event)
	if err == nil {
		return nil
	}
	if e.cfg.Codex401Refresher == nil || !cronShouldRefreshCodex401(job, err) {
		return err
	}
	ok, refreshErr := e.cfg.Codex401Refresher(ctx, job, err)
	if refreshErr != nil {
		return fmt.Errorf("codex auth refresh: %w", refreshErr)
	}
	if !ok {
		return err
	}
	retryErr := e.cfg.Kernel.Submit(event)
	if retryErr != nil {
		return retryErr
	}
	return nil
}

func cronShouldRefreshCodex401(job Job, err error) bool {
	if err == nil {
		return false
	}
	provider := strings.ToLower(strings.TrimSpace(job.Provider))
	if provider != "openai-codex" && provider != "codex" {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "401") || strings.Contains(text, "unauthorized")
}

type CronScriptRequest struct {
	Path        string
	ScriptsRoot string
	Workdir     string
	Timeout     time.Duration
}

type CronScriptResult struct {
	Success bool
	Output  string
}

type CronScriptRunner interface {
	RunCronScript(context.Context, CronScriptRequest) CronScriptResult
}

type CronScriptRunnerFunc func(context.Context, CronScriptRequest) CronScriptResult

func (f CronScriptRunnerFunc) RunCronScript(ctx context.Context, req CronScriptRequest) CronScriptResult {
	return f(ctx, req)
}

type ShellCronScriptRunner struct{}

func (ShellCronScriptRunner) RunCronScript(ctx context.Context, req CronScriptRequest) CronScriptResult {
	path, err := resolveCronScriptPath(req.Path, req.ScriptsRoot)
	if err != nil {
		return CronScriptResult{Output: err.Error()}
	}
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	argv := cronScriptArgv(path)
	cmd := exec.CommandContext(runCtx, argv[0], argv[1:]...)
	cmd.Dir = filepath.Dir(path)
	raw, runErr := cmd.CombinedOutput()
	output := strings.TrimSpace(redactCronScriptText(string(raw)))
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		return CronScriptResult{Output: fmt.Sprintf("Script timed out after %s: %s", timeout, path)}
	}
	if runErr != nil {
		if output == "" {
			output = runErr.Error()
		}
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			return CronScriptResult{Output: fmt.Sprintf("Script exited with code %d\n%s", exitErr.ExitCode(), output)}
		}
		return CronScriptResult{Output: "Script execution failed: " + output}
	}
	return CronScriptResult{Success: true, Output: output}
}

func resolveCronScriptPath(rawPath, scriptsRoot string) (string, error) {
	scriptsRoot = strings.TrimSpace(scriptsRoot)
	if scriptsRoot == "" {
		scriptsRoot = defaultCronScriptsRoot()
	}
	root, err := filepath.Abs(scriptsRoot)
	if err != nil {
		return "", fmt.Errorf("script root invalid: %w", err)
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", fmt.Errorf("script root unavailable: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("script root unavailable: %w", err)
	}
	raw := strings.TrimSpace(rawPath)
	if raw == "" {
		return "", errors.New("Script path is empty")
	}
	var candidate string
	if filepath.IsAbs(raw) {
		candidate = raw
	} else {
		candidate = filepath.Join(root, raw)
	}
	cleaned, err := filepath.Abs(candidate)
	if err != nil {
		return "", fmt.Errorf("Script path invalid: %w", err)
	}
	resolved := cleaned
	if eval, err := filepath.EvalSymlinks(cleaned); err == nil {
		resolved = eval
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
		return "", fmt.Errorf("Blocked: script path resolves outside the scripts directory (%s): %q", root, rawPath)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("Script not found: %s", resolved)
	}
	if info.IsDir() {
		return "", fmt.Errorf("Script path is not a file: %s", resolved)
	}
	return resolved, nil
}

func cronScriptArgv(path string) []string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".sh", ".bash":
		return []string{"/bin/bash", path}
	default:
		if python, err := exec.LookPath("python3"); err == nil {
			return []string{python, path}
		}
		return []string{"python", path}
	}
}

func defaultCronScriptsRoot() string {
	if home := strings.TrimSpace(os.Getenv("GORMES_HOME")); home != "" {
		return filepath.Join(home, "scripts")
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".gormes", "scripts")
	}
	return filepath.Join(".", "scripts")
}

func redactCronScriptText(text string) string {
	return cronScriptSecretPattern.ReplaceAllString(text, "${1}=[REDACTED]")
}

var cronScriptSecretPattern = regexp.MustCompile(`(?i)\b(token|api[_-]?key|secret|password)=\S+`)

func ResolveCronInactivityTimeout(lookup func(string) string) (time.Duration, bool) {
	if lookup == nil {
		lookup = os.Getenv
	}
	raw := strings.TrimSpace(lookup("HERMES_CRON_TIMEOUT"))
	if raw == "" {
		return 10 * time.Minute, true
	}
	seconds, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 10 * time.Minute, true
	}
	if seconds <= 0 {
		return 0, false
	}
	return time.Duration(seconds * float64(time.Second)), true
}

func resetTimer(timer *time.Timer, d time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(d)
}

func cronInactivityError(_ Job, idleFor, limit time.Duration, lastActivity string) error {
	return fmt.Errorf("idle for %s (limit %s) after last activity %s", idleFor.Round(time.Millisecond), limit.Round(time.Millisecond), displayCronActivity(lastActivity))
}

func displayCronActivity(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return value
}

func (e *Executor) startDurableCronRun(ctx context.Context, id string, job Job, promptHash, workerID string) bool {
	if e.cfg.DurableLedger == nil {
		return false
	}
	progress, err := durableCronProgress(job, promptHash, "queued")
	if err != nil {
		e.log.Warn("cron: durable ledger progress encode failed", "job_id", job.ID, "err", err)
		return false
	}
	if _, err := e.cfg.DurableLedger.Submit(ctx, subagent.DurableJobSubmission{
		ID:       id,
		Kind:     subagent.WorkKindCronJob,
		Progress: progress,
	}); err != nil {
		e.log.Warn("cron: durable ledger submit failed", "job_id", job.ID, "err", err)
		return false
	}
	_, ok, err := e.cfg.DurableLedger.ClaimJob(ctx, id, subagent.DurableClaim{
		WorkerID:  workerID,
		LockUntil: time.Now().UTC().Add(e.cfg.CallTimeout + time.Minute),
	})
	if err != nil || !ok {
		e.log.Warn("cron: durable ledger claim failed", "job_id", job.ID, "ok", ok, "err", err)
		return false
	}
	progress, err = durableCronProgress(job, promptHash, "submitted")
	if err != nil {
		e.log.Warn("cron: durable ledger progress encode failed", "job_id", job.ID, "err", err)
		return true
	}
	if ok, err := e.cfg.DurableLedger.UpdateProgress(ctx, id, workerID, progress); err != nil || !ok {
		e.log.Warn("cron: durable ledger progress update failed", "job_id", job.ID, "ok", ok, "err", err)
	}
	return true
}

func (e *Executor) completeDurableCronRun(id, workerID string, active bool, run Run) {
	if e.cfg.DurableLedger == nil || !active {
		return
	}
	raw, err := json.Marshal(map[string]any{
		"delivered": run.Delivered,
		"status":    run.Status,
	})
	if err != nil {
		e.log.Warn("cron: durable ledger result encode failed", "job_id", run.JobID, "err", err)
		return
	}
	if _, ok, err := e.cfg.DurableLedger.Complete(context.Background(), id, workerID, raw); err != nil || !ok {
		e.log.Warn("cron: durable ledger complete failed", "job_id", run.JobID, "ok", ok, "err", err)
	}
}

func (e *Executor) failDurableCronRun(id, workerID string, active bool, errorText string) {
	if e.cfg.DurableLedger == nil || !active {
		return
	}
	if _, ok, err := e.cfg.DurableLedger.Fail(context.Background(), id, workerID, errorText); err != nil || !ok {
		e.log.Warn("cron: durable ledger fail failed", "ledger_job_id", id, "ok", ok, "err", err)
	}
}

func durableCronProgress(job Job, promptHash, phase string) (json.RawMessage, error) {
	return json.Marshal(map[string]any{
		"cron_job_id": job.ID,
		"job_name":    job.Name,
		"phase":       phase,
		"prompt_hash": promptHash,
	})
}

type operatorReportContext struct {
	SessionID       string
	DeliveryPlan    DeliveryPlan
	DeliveryOutcome DeliveryOutcome
}

func (e *Executor) recordAndUpdateJob(ctx context.Context, job Job, run Run, reportCtx ...operatorReportContext) {
	var report operatorReportContext
	if len(reportCtx) > 0 {
		report = reportCtx[0]
	}
	completion, err := e.cfg.JobStore.ApplyRunCompletion(job, run, runCompletionNow(run), CronNextRunDecision)
	if err != nil {
		e.log.Warn("cron: failed to update job after run", "job_id", job.ID, "err", err)
		completion = cronRunCompletionForJob(job, run, runCompletionNow(run), CronNextRunDecision)
	}
	recordedRun := completion.Run
	id, err := e.cfg.RunStore.RecordRunWithID(ctx, recordedRun)
	if err != nil {
		e.log.Warn("cron: failed to record run", "job_id", job.ID, "err", err)
		return
	}
	recordedRun.ID = id
	e.writeOperatorRunReport(job, recordedRun, report)
}

func (e *Executor) writeOperatorRunReport(job Job, run Run, reportCtx operatorReportContext) {
	home := strings.TrimSpace(e.cfg.OperatorReportHome)
	if home == "" {
		return
	}
	report := BuildOperatorRunReport(OperatorRunReportInput{
		Job:             job,
		Run:             run,
		HomeDir:         home,
		RuntimeEvidence: map[string]any{"provider": job.Provider, "model": job.Model},
		DeliveryPlan:    reportCtx.DeliveryPlan,
		DeliveryOutcome: reportCtx.DeliveryOutcome,
		SessionID:       reportCtx.SessionID,
	})
	path := OperatorRunReportPath(home, report)
	if err := WriteOperatorRunReport(path, report); err != nil {
		e.log.Warn("cron: failed to write operator run report", "job_id", job.ID, "run_id", run.ID, "err", err)
	}
}

func runCompletionNow(run Run) time.Time {
	at := run.FinishedAt
	if at == 0 {
		at = run.StartedAt
	}
	if at == 0 {
		return time.Now()
	}
	return time.Unix(at, 0).UTC()
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
	return cutBytesAtRuneBoundary(s, n) + "…"
}

// cutBytesAtRuneBoundary returns the longest prefix of s that is at most n
// bytes and does not split a multibyte UTF-8 sequence.
func cutBytesAtRuneBoundary(s string, n int) string {
	if n >= len(s) {
		return s
	}
	if n < 0 {
		return ""
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}
