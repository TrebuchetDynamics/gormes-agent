package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

type browserHarnessChromedpLiveTransport struct {
	mu            sync.Mutex
	browserCtx    context.Context
	targetCtx     context.Context
	allocCancel   context.CancelFunc
	browserCancel context.CancelFunc
	sessionState  browserHarnessTargetSessionState
}

func newBrowserHarnessChromedpLiveTransport(parent context.Context, endpoint string) (BrowserHarnessCDPTransport, error) {
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(parent, endpoint)
	ctx, cancel := chromedp.NewContext(allocCtx)
	if err := chromedp.Run(ctx); err != nil {
		cancel()
		allocCancel()
		return nil, fmt.Errorf("dial %q: %w", "[redacted]", err)
	}
	transport := &browserHarnessChromedpLiveTransport{
		browserCtx:    ctx,
		targetCtx:     ctx,
		allocCancel:   allocCancel,
		browserCancel: cancel,
		sessionState:  newBrowserHarnessTargetSessionStateFromEnv(),
	}
	transport.restoreTarget(ctx)
	return transport, nil
}

func (t *browserHarnessChromedpLiveTransport) SendCommand(ctx context.Context, method string, params any) (json.RawMessage, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	c := chromedp.FromContext(t.browserCtx)
	if c == nil || c.Browser == nil {
		return nil, fmt.Errorf("chromedp browser not initialised")
	}
	var result json.RawMessage
	if browserHarnessCDPCommandScope(method) == browserHarnessCDPScopeBrowser {
		runCtx, cancel := browserHarnessContextWithOptionalDeadline(t.browserCtx, ctx)
		defer cancel()
		if err := c.Browser.Execute(runCtx, method, params, &result); err != nil {
			return nil, err
		}
		if method == "Target.createTarget" {
			t.switchToCreatedTarget(ctx, result)
		}
		return result, nil
	}

	runCtx, cancel := browserHarnessContextWithOptionalDeadline(t.targetCtx, ctx)
	defer cancel()
	err := chromedp.Run(runCtx, chromedp.ActionFunc(func(actionCtx context.Context) error {
		return cdp.Execute(actionCtx, method, params, &result)
	}))
	if err != nil {
		return nil, err
	}
	return result, nil
}

type browserHarnessCDPScope string

const (
	browserHarnessCDPScopeBrowser browserHarnessCDPScope = "browser"
	browserHarnessCDPScopeTarget  browserHarnessCDPScope = "target"
)

func browserHarnessCDPCommandScope(method string) browserHarnessCDPScope {
	domain, _, _ := strings.Cut(strings.TrimSpace(method), ".")
	switch domain {
	case "Browser", "Target":
		return browserHarnessCDPScopeBrowser
	default:
		return browserHarnessCDPScopeTarget
	}
}

func browserHarnessContextWithOptionalDeadline(base context.Context, caller context.Context) (context.Context, context.CancelFunc) {
	if deadline, ok := caller.Deadline(); ok {
		return context.WithDeadline(base, deadline)
	}
	return base, func() {}
}

func (t *browserHarnessChromedpLiveTransport) switchToCreatedTarget(ctx context.Context, raw json.RawMessage) {
	var createResult struct {
		TargetID target.ID `json:"targetId"`
	}
	if err := json.Unmarshal(raw, &createResult); err != nil || createResult.TargetID == "" {
		return
	}
	targetCtx, _ := chromedp.NewContext(t.browserCtx, chromedp.WithTargetID(createResult.TargetID))
	runCtx, cancel := browserHarnessContextWithOptionalDeadline(targetCtx, ctx)
	defer cancel()
	if err := chromedp.Run(runCtx); err != nil {
		return
	}
	t.targetCtx = targetCtx
	_ = t.sessionState.save(createResult.TargetID)
}

func (t *browserHarnessChromedpLiveTransport) restoreTarget(ctx context.Context) {
	targetID := t.sessionState.load()
	if targetID == "" {
		return
	}
	targetCtx, _ := chromedp.NewContext(t.browserCtx, chromedp.WithTargetID(targetID))
	runCtx, cancel := browserHarnessContextWithOptionalDeadline(targetCtx, ctx)
	defer cancel()
	if err := chromedp.Run(runCtx); err != nil {
		_ = os.Remove(t.sessionState.path())
		return
	}
	t.targetCtx = targetCtx
}

type browserHarnessTargetSessionState struct {
	dir         string
	sessionName string
}

func newBrowserHarnessTargetSessionStateFromEnv() browserHarnessTargetSessionState {
	return browserHarnessTargetSessionState{
		dir:         browserHarnessTargetStateDir(),
		sessionName: sanitizeBrowserHarnessTargetSessionName(os.Getenv("BU_NAME")),
	}
}

func browserHarnessTargetStateDir() string {
	if dir := strings.TrimSpace(os.Getenv("GORMES_BROWSER_HARNESS_STATE_DIR")); dir != "" {
		return dir
	}
	if dir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR")); dir != "" {
		return filepath.Join(dir, "gormes-browser-harness")
	}
	return filepath.Join(os.TempDir(), "gormes-browser-harness")
}

func sanitizeBrowserHarnessTargetSessionName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = defaultBrowserTaskID
	}
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_' || r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	name := strings.Trim(b.String(), "_-")
	if name == "" {
		return defaultBrowserTaskID
	}
	if len(name) > 80 {
		name = strings.TrimRight(name[:80], "_-")
	}
	if name == "" {
		return defaultBrowserTaskID
	}
	return name
}

func (s browserHarnessTargetSessionState) path() string {
	name := s.sessionName
	if name == "" {
		name = defaultBrowserTaskID
	}
	return filepath.Join(s.dir, name+".json")
}

func (s browserHarnessTargetSessionState) load() target.ID {
	raw, err := os.ReadFile(s.path())
	if err != nil {
		return ""
	}
	var payload struct {
		TargetID target.ID `json:"target_id"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	return payload.TargetID
}

func (s browserHarnessTargetSessionState) save(targetID target.ID) error {
	if targetID == "" {
		return nil
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	payload := struct {
		TargetID target.ID `json:"target_id"`
	}{TargetID: targetID}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	path := s.path()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
