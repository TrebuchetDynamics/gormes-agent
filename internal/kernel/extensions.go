package kernel

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type ExtensionHook string

const (
	ExtensionHookAgentInit            ExtensionHook = "agent_init"
	ExtensionHookMonologueStart       ExtensionHook = "monologue_start"
	ExtensionHookMonologueEnd         ExtensionHook = "monologue_end"
	ExtensionHookMessageLoopStart     ExtensionHook = "message_loop_start"
	ExtensionHookMessageLoopEnd       ExtensionHook = "message_loop_end"
	ExtensionHookBeforeMainLLMCall    ExtensionHook = "before_main_llm_call"
	ExtensionHookPromptBefore         ExtensionHook = "prompt_before"
	ExtensionHookPromptAfter          ExtensionHook = "prompt_after"
	ExtensionHookResponseStreamChunk  ExtensionHook = "response_stream_chunk"
	ExtensionHookReasoningStreamChunk ExtensionHook = "reasoning_stream_chunk"
	ExtensionHookToolBefore           ExtensionHook = "tool_before"
	ExtensionHookToolAfter            ExtensionHook = "tool_after"
	ExtensionHookContextDeleted       ExtensionHook = "context_deleted"
)

var extensionHooks = []ExtensionHook{
	ExtensionHookAgentInit,
	ExtensionHookMonologueStart,
	ExtensionHookMonologueEnd,
	ExtensionHookMessageLoopStart,
	ExtensionHookMessageLoopEnd,
	ExtensionHookBeforeMainLLMCall,
	ExtensionHookPromptBefore,
	ExtensionHookPromptAfter,
	ExtensionHookResponseStreamChunk,
	ExtensionHookReasoningStreamChunk,
	ExtensionHookToolBefore,
	ExtensionHookToolAfter,
	ExtensionHookContextDeleted,
}

var extensionHookSet = func() map[ExtensionHook]struct{} {
	set := make(map[ExtensionHook]struct{}, len(extensionHooks))
	for _, hook := range extensionHooks {
		set[hook] = struct{}{}
	}
	return set
}()

func AllExtensionHooks() []ExtensionHook {
	out := make([]ExtensionHook, len(extensionHooks))
	copy(out, extensionHooks)
	return out
}

type ExtensionStatus string

const (
	ExtensionStatusCompleted ExtensionStatus = "completed"
	ExtensionStatusError     ExtensionStatus = "error"
	ExtensionStatusTimeout   ExtensionStatus = "timeout"
	ExtensionStatusPanic     ExtensionStatus = "panic"
	ExtensionStatusSkipped   ExtensionStatus = "skipped"
)

type ExtensionData struct {
	Values map[string]any
}

type ExtensionHandler func(context.Context, ExtensionHook, *ExtensionData) error

type ExtensionRegistration struct {
	Name     string
	Hooks    []ExtensionHook
	Handler  ExtensionHandler
	Timeout  time.Duration
	Disabled bool
}

type ExtensionChainOptions struct {
	DefaultTimeout time.Duration
}

type ExtensionChain struct {
	mu             sync.RWMutex
	defaultTimeout time.Duration
	extensions     map[ExtensionHook][]extensionRegistration
}

type extensionRegistration struct {
	name     string
	hooks    []ExtensionHook
	handler  ExtensionHandler
	timeout  time.Duration
	disabled bool
}

type ExtensionRunReport struct {
	Hook    ExtensionHook
	Results []ExtensionResult
	Elapsed time.Duration
}

type ExtensionResult struct {
	Name    string
	Hook    ExtensionHook
	Status  ExtensionStatus
	Elapsed time.Duration
	Error   string
	Panic   string
}

func NewExtensionChain(opts ExtensionChainOptions) *ExtensionChain {
	timeout := opts.DefaultTimeout
	if timeout <= 0 {
		timeout = 500 * time.Millisecond
	}
	return &ExtensionChain{
		defaultTimeout: timeout,
		extensions:     make(map[ExtensionHook][]extensionRegistration),
	}
}

func (c *ExtensionChain) Register(reg ExtensionRegistration) error {
	if c == nil {
		return errors.New("extension_chain_nil")
	}
	name := strings.TrimSpace(reg.Name)
	if name == "" {
		return errors.New("extension_name_required")
	}
	if len(reg.Hooks) == 0 {
		return fmt.Errorf("extension_hooks_required: %s", name)
	}
	if reg.Handler == nil {
		return fmt.Errorf("extension_handler_required: %s", name)
	}
	cleanHooks := make([]ExtensionHook, 0, len(reg.Hooks))
	for _, hook := range reg.Hooks {
		if _, ok := extensionHookSet[hook]; !ok {
			return fmt.Errorf("extension_unknown_hook: %s", hook)
		}
		cleanHooks = append(cleanHooks, hook)
	}
	registered := extensionRegistration{
		name:     name,
		hooks:    cleanHooks,
		handler:  reg.Handler,
		timeout:  reg.Timeout,
		disabled: reg.Disabled,
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, hook := range cleanHooks {
		c.extensions[hook] = append(c.extensions[hook], registered)
	}
	return nil
}

func (c *ExtensionChain) Run(ctx context.Context, hook ExtensionHook, data *ExtensionData) ExtensionRunReport {
	started := time.Now()
	report := ExtensionRunReport{Hook: hook}
	if c == nil {
		return report
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if data == nil {
		data = &ExtensionData{}
	}
	if data.Values == nil {
		data.Values = map[string]any{}
	}
	c.mu.RLock()
	extensions := append([]extensionRegistration(nil), c.extensions[hook]...)
	c.mu.RUnlock()
	for _, ext := range extensions {
		report.Results = append(report.Results, c.runOne(ctx, hook, ext, data))
	}
	report.Elapsed = elapsedSince(started)
	return report
}

func (c *ExtensionChain) runOne(ctx context.Context, hook ExtensionHook, ext extensionRegistration, data *ExtensionData) ExtensionResult {
	result := ExtensionResult{Name: ext.name, Hook: hook}
	if ext.disabled {
		result.Status = ExtensionStatusSkipped
		result.Error = "extension disabled"
		return result
	}

	timeout := ext.timeout
	if timeout <= 0 {
		timeout = c.defaultTimeout
	}
	extCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	started := time.Now()
	done := make(chan ExtensionResult, 1)
	go func() {
		r := ExtensionResult{Name: ext.name, Hook: hook}
		defer func() {
			r.Elapsed = elapsedSince(started)
			if recovered := recover(); recovered != nil {
				r.Status = ExtensionStatusPanic
				r.Panic = fmt.Sprint(recovered)
			}
			done <- r
		}()
		if err := ext.handler(extCtx, hook, data); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				r.Status = ExtensionStatusTimeout
				r.Error = "extension hook timed out"
				return
			}
			r.Status = ExtensionStatusError
			r.Error = err.Error()
			return
		}
		r.Status = ExtensionStatusCompleted
	}()

	select {
	case result = <-done:
		return result
	case <-extCtx.Done():
		result.Elapsed = elapsedSince(started)
		if errors.Is(extCtx.Err(), context.DeadlineExceeded) {
			result.Status = ExtensionStatusTimeout
			result.Error = "extension hook timed out"
			return result
		}
		result.Status = ExtensionStatusError
		result.Error = extCtx.Err().Error()
		return result
	}
}

func elapsedSince(started time.Time) time.Duration {
	elapsed := time.Since(started)
	if elapsed <= 0 {
		return time.Nanosecond
	}
	return elapsed
}
