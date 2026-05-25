package kernel

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestExtensionsRegisterHookPointsAndRunInOrder(t *testing.T) {
	wantHooks := []ExtensionHook{
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
	if got := AllExtensionHooks(); !reflect.DeepEqual(got, wantHooks) {
		t.Fatalf("AllExtensionHooks = %#v, want %#v", got, wantHooks)
	}

	chain := NewExtensionChain(ExtensionChainOptions{DefaultTimeout: time.Second})
	var order []string
	mustRegisterExtension(t, chain, ExtensionRegistration{
		Name:  "first",
		Hooks: []ExtensionHook{ExtensionHookPromptBefore},
		Handler: func(_ context.Context, hook ExtensionHook, data *ExtensionData) error {
			order = append(order, "first:"+string(hook))
			data.Values["prompt"] = "first"
			return nil
		},
	})
	mustRegisterExtension(t, chain, ExtensionRegistration{
		Name:  "second",
		Hooks: []ExtensionHook{ExtensionHookPromptBefore},
		Handler: func(_ context.Context, hook ExtensionHook, data *ExtensionData) error {
			order = append(order, "second:"+string(hook)+":"+data.Values["prompt"].(string))
			data.Values["prompt"] = "second"
			return nil
		},
	})

	data := &ExtensionData{Values: map[string]any{}}
	report := chain.Run(context.Background(), ExtensionHookPromptBefore, data)

	if got, want := order, []string{"first:prompt_before", "second:prompt_before:first"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %#v, want %#v", got, want)
	}
	if got := data.Values["prompt"]; got != "second" {
		t.Fatalf("data prompt = %v, want second", got)
	}
	if report.Hook != ExtensionHookPromptBefore {
		t.Fatalf("report hook = %q, want prompt_before", report.Hook)
	}
	if len(report.Results) != 2 {
		t.Fatalf("report results = %d, want 2: %#v", len(report.Results), report.Results)
	}
	for _, result := range report.Results {
		if result.Status != ExtensionStatusCompleted {
			t.Fatalf("result = %#v, want completed", result)
		}
		if result.Elapsed <= 0 {
			t.Fatalf("result elapsed = %s, want >0", result.Elapsed)
		}
	}
}

func TestExtensionsIsolatePanicTimeoutErrorAndSkip(t *testing.T) {
	chain := NewExtensionChain(ExtensionChainOptions{DefaultTimeout: 10 * time.Millisecond})
	mustRegisterExtension(t, chain, ExtensionRegistration{
		Name:  "panic-ext",
		Hooks: []ExtensionHook{ExtensionHookToolBefore},
		Handler: func(context.Context, ExtensionHook, *ExtensionData) error {
			panic("bad plugin")
		},
	})
	mustRegisterExtension(t, chain, ExtensionRegistration{
		Name:  "timeout-ext",
		Hooks: []ExtensionHook{ExtensionHookToolBefore},
		Handler: func(ctx context.Context, _ ExtensionHook, _ *ExtensionData) error {
			<-ctx.Done()
			return ctx.Err()
		},
	})
	mustRegisterExtension(t, chain, ExtensionRegistration{
		Name:  "error-ext",
		Hooks: []ExtensionHook{ExtensionHookToolBefore},
		Handler: func(context.Context, ExtensionHook, *ExtensionData) error {
			return errors.New("bad hook")
		},
	})
	mustRegisterExtension(t, chain, ExtensionRegistration{
		Name:     "disabled-ext",
		Hooks:    []ExtensionHook{ExtensionHookToolBefore},
		Disabled: true,
		Handler: func(context.Context, ExtensionHook, *ExtensionData) error {
			t.Fatal("disabled extension handler ran")
			return nil
		},
	})

	report := chain.Run(context.Background(), ExtensionHookToolBefore, &ExtensionData{})

	wantStatuses := map[string]ExtensionStatus{
		"panic-ext":    ExtensionStatusPanic,
		"timeout-ext":  ExtensionStatusTimeout,
		"error-ext":    ExtensionStatusError,
		"disabled-ext": ExtensionStatusSkipped,
	}
	if len(report.Results) != len(wantStatuses) {
		t.Fatalf("results = %d, want %d: %#v", len(report.Results), len(wantStatuses), report.Results)
	}
	for _, result := range report.Results {
		want, ok := wantStatuses[result.Name]
		if !ok {
			t.Fatalf("unexpected result: %#v", result)
		}
		if result.Hook != ExtensionHookToolBefore {
			t.Fatalf("%s hook = %q, want tool_before", result.Name, result.Hook)
		}
		if result.Status != want {
			t.Fatalf("%s status = %q, want %q: %#v", result.Name, result.Status, want, result)
		}
		if result.Status != ExtensionStatusSkipped && result.Elapsed <= 0 {
			t.Fatalf("%s elapsed = %s, want >0", result.Name, result.Elapsed)
		}
	}
	if got := resultByName(t, report, "panic-ext").Panic; !strings.Contains(got, "bad plugin") {
		t.Fatalf("panic evidence = %q, want bad plugin", got)
	}
	if got := resultByName(t, report, "timeout-ext").Error; !strings.Contains(got, "timed out") {
		t.Fatalf("timeout evidence = %q, want timed out", got)
	}
	if got := resultByName(t, report, "error-ext").Error; !strings.Contains(got, "bad hook") {
		t.Fatalf("error evidence = %q, want bad hook", got)
	}
	if got := resultByName(t, report, "disabled-ext").Error; !strings.Contains(got, "disabled") {
		t.Fatalf("skip evidence = %q, want disabled", got)
	}
}

func TestExtensionsRejectInvalidRegistration(t *testing.T) {
	chain := NewExtensionChain(ExtensionChainOptions{})
	if err := chain.Register(ExtensionRegistration{Name: "missing hooks", Handler: func(context.Context, ExtensionHook, *ExtensionData) error { return nil }}); err == nil {
		t.Fatal("Register missing hooks succeeded, want error")
	}
	if err := chain.Register(ExtensionRegistration{Name: "missing handler", Hooks: []ExtensionHook{ExtensionHookAgentInit}}); err == nil {
		t.Fatal("Register missing handler succeeded, want error")
	}
	if err := chain.Register(ExtensionRegistration{Name: "unknown", Hooks: []ExtensionHook{"future_hook"}, Handler: func(context.Context, ExtensionHook, *ExtensionData) error { return nil }}); err == nil {
		t.Fatal("Register unknown hook succeeded, want error")
	}
}

func TestExtensionUINoopReturnsTypedUnavailableEvidence(t *testing.T) {
	ui := NewNoopExtensionUI("print mode")
	for name, res := range map[string]ExtensionUIResult{
		"set status":    ui.SetStatus("demo", "running"),
		"clear status":  ui.ClearStatus("demo"),
		"set widget":    ui.SetWidget("demo", []string{"line"}, ExtensionUIWidgetOptions{}),
		"clear widget":  ui.ClearWidget("demo"),
		"set footer":    ui.SetFooter([]string{"footer"}),
		"clear footer":  ui.ClearFooter(),
		"set working":   ui.SetWorkingIndicator(ExtensionUIWorkingIndicator{Text: "working", Frames: []string{"●"}}),
		"clear working": ui.ClearWorkingIndicator(),
	} {
		t.Run(name, func(t *testing.T) {
			if res.Status != ExtensionUIUnavailable {
				t.Fatalf("status = %q, want unavailable: %#v", res.Status, res)
			}
			if !strings.Contains(res.Evidence, "print mode") {
				t.Fatalf("evidence = %q, want print mode", res.Evidence)
			}
		})
	}
}

func mustRegisterExtension(t *testing.T, chain *ExtensionChain, reg ExtensionRegistration) {
	t.Helper()
	if err := chain.Register(reg); err != nil {
		t.Fatalf("Register(%s): %v", reg.Name, err)
	}
}

func resultByName(t *testing.T, report ExtensionRunReport, name string) ExtensionResult {
	t.Helper()
	for _, result := range report.Results {
		if result.Name == name {
			return result
		}
	}
	t.Fatalf("missing result %q in %#v", name, report.Results)
	return ExtensionResult{}
}
