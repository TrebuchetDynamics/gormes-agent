package tools

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	defaultBrowserUseBaseURL                    = "https://api.browser-use.com/api/v3"
	defaultBrowserUseManagedTimeoutMinutes      = 5
	defaultBrowserUseManagedProxyCountryCode    = "us"
	defaultBrowserHarnessCommand                = "browser-harness"
	defaultBrowserHarnessTimeout                = 2 * time.Minute
	defaultBrowserUseHTTPBodyLimit              = 4 * 1024
	defaultBrowserUseRedactedErrorPreviewLength = 512

	BrowserProviderBrowserUse  = "browser-use"
	BrowserProviderBrowserbase = "browserbase"
	BrowserProviderFirecrawl   = "firecrawl"

	BrowserUseEvidenceUnconfigured        = "browser_use_unconfigured"
	BrowserUseEvidenceSessionCreated      = "browser_use_session_created"
	BrowserUseEvidenceSessionCreateFailed = "browser_use_session_create_failed"
	BrowserUseEvidenceCleanupOK           = "browser_use_cleanup_ok"
	BrowserUseEvidenceCleanupSkipped      = "browser_use_cleanup_skipped"
	BrowserUseEvidenceCleanupFailed       = "browser_use_cleanup_failed"

	BrowserHarnessEvidenceCommandOK     = "browser_harness_command_ok"
	BrowserHarnessEvidenceUnavailable   = "browser_harness_unavailable"
	BrowserHarnessEvidenceCommandFailed = "browser_harness_command_failed"
)

// BrowserCloudProviderCandidate is the minimal provider-selection record used
// by browser runtime callers before they own concrete provider implementations.
type BrowserCloudProviderCandidate struct {
	Name       string
	Configured bool
	Evidence   string
}

// SelectBrowserCloudProvider keeps Browser Use preferred over Browserbase and
// Firecrawl when more than one cloud provider is configured. Hermes historically
// grew provider-specific branches; this helper gives Gormes one deterministic
// selection point for later Browserbase/Firecrawl bridge work.
func SelectBrowserCloudProvider(candidates ...BrowserCloudProviderCandidate) BrowserCloudProviderCandidate {
	byName := map[string]BrowserCloudProviderCandidate{}
	for _, candidate := range candidates {
		name := strings.ToLower(strings.TrimSpace(candidate.Name))
		if name == "" || !candidate.Configured {
			continue
		}
		candidate.Name = name
		byName[name] = candidate
	}
	for _, preferred := range []string{BrowserProviderBrowserUse, BrowserProviderBrowserbase, BrowserProviderFirecrawl} {
		if candidate, ok := byName[preferred]; ok {
			return candidate
		}
	}
	for _, candidate := range candidates {
		if candidate.Configured {
			candidate.Name = strings.ToLower(strings.TrimSpace(candidate.Name))
			return candidate
		}
	}
	return BrowserCloudProviderCandidate{Evidence: BrowserUseEvidenceUnconfigured}
}

// BrowserUseConfig is the fakeable Browser Use cloud credential and request
// default contract. Direct mode uses BROWSER_USE_API_KEY; managed mode uses the
// gateway origin/token shape Hermes receives from the managed tool gateway.
type BrowserUseConfig struct {
	APIKey           string
	BaseURL          string
	Managed          bool
	GatewayOrigin    string
	GatewayToken     string
	ProfileName      string
	ProfileID        string
	ProxyCountryCode string
	TimeoutMinutes   int
}

// BrowserUseConfigFromEnv resolves only direct Browser Use env credentials.
// Managed gateway discovery stays explicit so unit tests never need process env
// mutation or network-backed discovery.
func BrowserUseConfigFromEnv(lookup func(string) string) BrowserUseConfig {
	if lookup == nil {
		lookup = os.Getenv
	}
	return BrowserUseConfig{
		APIKey: strings.TrimSpace(lookup("BROWSER_USE_API_KEY")),
	}
}

// BrowserUseSessionRequest is one task-scoped Browser Use cloud session create
// request. Profile values are sent to Browser Use when configured, but the
// result records only safe booleans so transcripts do not leak profile state.
type BrowserUseSessionRequest struct {
	TaskID           string
	ProfileName      string
	ProfileID        string
	ProxyCountryCode string
	TimeoutMinutes   int
}

// BrowserUseSessionInputs records the non-secret session input shape for audit
// evidence. Profile values are redacted into presence bits.
type BrowserUseSessionInputs struct {
	TimeoutMinutes   int
	ProxyCountryCode string
	ProfileNameSet   bool
	ProfileIDSet     bool
}

// BrowserUseSession mirrors Hermes' BrowserUseProvider session shape while
// retaining Gormes evidence and redaction state.
type BrowserUseSession struct {
	SessionName            string
	ProviderSessionID      string
	CompatibilitySessionID string
	CDPURL                 string
	LiveURL                string
	Features               map[string]bool
	ExternalCallID         string
	Inputs                 BrowserUseSessionInputs
	Evidence               string
	Redacted               bool
}

// BrowserUseCleanupResult is the typed stop/cleanup outcome. The raw session
// id is intentionally not echoed back to callers that may log this result.
type BrowserUseCleanupResult struct {
	Stopped  bool
	Evidence string
	Redacted bool
}

// BrowserUseHTTPClient is satisfied by *http.Client and by unit-test fakes.
type BrowserUseHTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// BrowserUseProviderBridge is a small Go port of Hermes'
// BrowserUseProvider lifecycle logic. It never performs work at construction;
// tests inject BrowserUseHTTPClient fakes to avoid live DNS/cloud calls.
type BrowserUseProviderBridge struct {
	cfg       BrowserUseConfig
	client    BrowserUseHTTPClient
	newCreate func(taskID string) string
}

// NewBrowserUseProviderBridge constructs a Browser Use lifecycle bridge.
func NewBrowserUseProviderBridge(cfg BrowserUseConfig, client BrowserUseHTTPClient) *BrowserUseProviderBridge {
	if client == nil {
		client = http.DefaultClient
	}
	return &BrowserUseProviderBridge{
		cfg:       cfg,
		client:    client,
		newCreate: newBrowserUseCreateKey,
	}
}

// Configured reports whether the bridge has either direct Browser Use API
// credentials or managed gateway state.
func (b *BrowserUseProviderBridge) Configured() bool {
	_, _, ok := b.resolvedConfig()
	return ok
}

// CreateSession POSTs /browsers through the injected client and maps Browser
// Use's id/cdpUrl/connectUrl/liveUrl response into the shared session result.
func (b *BrowserUseProviderBridge) CreateSession(ctx context.Context, req BrowserUseSessionRequest) (BrowserUseSession, error) {
	baseURL, apiKey, ok := b.resolvedConfig()
	if !ok {
		return BrowserUseSession{}, newBrowserUseError(BrowserUseEvidenceUnconfigured, "Browser Use requires BROWSER_USE_API_KEY or managed gateway credentials")
	}
	body, inputs, err := b.createPayload(req)
	if err != nil {
		return BrowserUseSession{}, newBrowserUseError(BrowserUseEvidenceSessionCreateFailed, err.Error())
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/browsers", bytes.NewReader(body))
	if err != nil {
		return BrowserUseSession{}, newBrowserUseError(BrowserUseEvidenceSessionCreateFailed, err.Error())
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Browser-Use-API-Key", apiKey)
	if b.cfg.Managed {
		httpReq.Header.Set("X-Idempotency-Key", b.createKey(req.TaskID))
	}

	resp, err := b.client.Do(httpReq)
	if err != nil {
		return BrowserUseSession{}, newBrowserUseError(BrowserUseEvidenceSessionCreateFailed, b.redact(req, "", err.Error()))
	}
	defer resp.Body.Close()

	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, defaultBrowserUseHTTPBodyLimit))
	if readErr != nil {
		return BrowserUseSession{}, newBrowserUseError(BrowserUseEvidenceSessionCreateFailed, b.redact(req, "", readErr.Error()))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := fmt.Sprintf("Browser Use create failed: HTTP %d %s", resp.StatusCode, string(respBody))
		return BrowserUseSession{}, newBrowserUseError(BrowserUseEvidenceSessionCreateFailed, b.redact(req, "", msg))
	}

	var payload browserUseCreateResponse
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return BrowserUseSession{}, newBrowserUseError(BrowserUseEvidenceSessionCreateFailed, b.redact(req, "", "decode Browser Use create response: "+err.Error()))
	}
	cdpURL := strings.TrimSpace(payload.CDPURL)
	if cdpURL == "" {
		cdpURL = strings.TrimSpace(payload.ConnectURL)
	}
	if strings.TrimSpace(payload.ID) == "" || cdpURL == "" {
		return BrowserUseSession{}, newBrowserUseError(BrowserUseEvidenceSessionCreateFailed, "Browser Use create response missing id or cdpUrl/connectUrl")
	}
	features := map[string]bool{"browser_use": true}
	for k, v := range payload.Features {
		if strings.TrimSpace(k) != "" {
			features[k] = v
		}
	}

	return BrowserUseSession{
		SessionName:            buildBrowserUseSessionName(req.TaskID, payload.ID),
		ProviderSessionID:      payload.ID,
		CompatibilitySessionID: payload.ID,
		CDPURL:                 cdpURL,
		LiveURL:                strings.TrimSpace(payload.LiveURL),
		Features:               features,
		ExternalCallID:         resp.Header.Get("X-External-Call-Id"),
		Inputs:                 inputs,
		Evidence:               BrowserUseEvidenceSessionCreated,
		Redacted:               true,
	}, nil
}

// StopSession PATCHes action=stop for a Browser Use browser id.
func (b *BrowserUseProviderBridge) StopSession(ctx context.Context, sessionID string) (BrowserUseCleanupResult, error) {
	return b.stopSession(ctx, sessionID, true)
}

// EmergencyCleanup uses the same Browser Use stop action while tolerating empty
// session ids. It remains fakeable and does not panic if credentials vanished.
func (b *BrowserUseProviderBridge) EmergencyCleanup(ctx context.Context, sessionID string) (BrowserUseCleanupResult, error) {
	return b.stopSession(ctx, sessionID, false)
}

func (b *BrowserUseProviderBridge) stopSession(ctx context.Context, sessionID string, requireConfigured bool) (BrowserUseCleanupResult, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return BrowserUseCleanupResult{Evidence: BrowserUseEvidenceCleanupSkipped, Redacted: true}, nil
	}
	baseURL, apiKey, ok := b.resolvedConfig()
	if !ok {
		result := BrowserUseCleanupResult{Evidence: BrowserUseEvidenceCleanupFailed, Redacted: true}
		err := newBrowserUseError(BrowserUseEvidenceCleanupFailed, "Browser Use cleanup requires credentials")
		if requireConfigured {
			return result, err
		}
		return result, nil
	}
	body := []byte(`{"action":"stop"}`)
	u := baseURL + "/browsers/" + url.PathEscape(sessionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u, bytes.NewReader(body))
	if err != nil {
		return BrowserUseCleanupResult{Evidence: BrowserUseEvidenceCleanupFailed, Redacted: true}, newBrowserUseError(BrowserUseEvidenceCleanupFailed, b.redact(BrowserUseSessionRequest{}, sessionID, err.Error()))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Browser-Use-API-Key", apiKey)

	resp, err := b.client.Do(req)
	if err != nil {
		return BrowserUseCleanupResult{Evidence: BrowserUseEvidenceCleanupFailed, Redacted: true}, newBrowserUseError(BrowserUseEvidenceCleanupFailed, b.redact(BrowserUseSessionRequest{}, sessionID, err.Error()))
	}
	defer resp.Body.Close()
	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, defaultBrowserUseHTTPBodyLimit))
	if readErr != nil {
		return BrowserUseCleanupResult{Evidence: BrowserUseEvidenceCleanupFailed, Redacted: true}, newBrowserUseError(BrowserUseEvidenceCleanupFailed, b.redact(BrowserUseSessionRequest{}, sessionID, readErr.Error()))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := fmt.Sprintf("Browser Use cleanup failed: HTTP %d %s", resp.StatusCode, string(respBody))
		return BrowserUseCleanupResult{Evidence: BrowserUseEvidenceCleanupFailed, Redacted: true}, newBrowserUseError(BrowserUseEvidenceCleanupFailed, b.redact(BrowserUseSessionRequest{}, sessionID, msg))
	}
	return BrowserUseCleanupResult{Stopped: true, Evidence: BrowserUseEvidenceCleanupOK, Redacted: true}, nil
}

func (b *BrowserUseProviderBridge) resolvedConfig() (baseURL, apiKey string, ok bool) {
	if b == nil {
		return "", "", false
	}
	cfg := b.cfg
	if cfg.Managed {
		apiKey = strings.TrimSpace(firstNonEmpty(cfg.GatewayToken, cfg.APIKey))
		baseURL = strings.TrimRight(strings.TrimSpace(firstNonEmpty(cfg.GatewayOrigin, cfg.BaseURL)), "/")
		return baseURL, apiKey, baseURL != "" && apiKey != ""
	}
	apiKey = strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return "", "", false
	}
	baseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBrowserUseBaseURL
	}
	return baseURL, apiKey, true
}

func (b *BrowserUseProviderBridge) createPayload(req BrowserUseSessionRequest) ([]byte, BrowserUseSessionInputs, error) {
	profileName := firstNonEmpty(req.ProfileName, b.cfg.ProfileName)
	profileID := firstNonEmpty(req.ProfileID, b.cfg.ProfileID)
	timeout := firstPositive(req.TimeoutMinutes, b.cfg.TimeoutMinutes)
	proxy := firstNonEmpty(req.ProxyCountryCode, b.cfg.ProxyCountryCode)
	if b.cfg.Managed {
		if timeout <= 0 {
			timeout = defaultBrowserUseManagedTimeoutMinutes
		}
		if proxy == "" {
			proxy = defaultBrowserUseManagedProxyCountryCode
		}
	}

	payload := map[string]any{}
	if timeout > 0 {
		payload["timeout"] = timeout
	}
	if proxy != "" {
		payload["proxyCountryCode"] = proxy
	}
	if profileName != "" {
		payload["profileName"] = profileName
	}
	if profileID != "" {
		payload["profileId"] = profileID
	}
	body, err := json.Marshal(payload)
	return body, BrowserUseSessionInputs{
		TimeoutMinutes:   timeout,
		ProxyCountryCode: proxy,
		ProfileNameSet:   profileName != "",
		ProfileIDSet:     profileID != "",
	}, err
}

func (b *BrowserUseProviderBridge) createKey(taskID string) string {
	if b != nil && b.newCreate != nil {
		return b.newCreate(taskID)
	}
	return newBrowserUseCreateKey(taskID)
}

func (b *BrowserUseProviderBridge) redact(req BrowserUseSessionRequest, sessionID, text string) string {
	secrets := []string{
		b.cfg.APIKey,
		b.cfg.GatewayToken,
		b.cfg.ProfileName,
		b.cfg.ProfileID,
		req.ProfileName,
		req.ProfileID,
		sessionID,
	}
	return redactBrowserBridgeText(text, secrets...)
}

type browserUseCreateResponse struct {
	ID         string          `json:"id"`
	CDPURL     string          `json:"cdpUrl"`
	ConnectURL string          `json:"connectUrl"`
	LiveURL    string          `json:"liveUrl"`
	Features   map[string]bool `json:"features"`
}

type BrowserUseError struct {
	Evidence string
	Message  string
}

func (e *BrowserUseError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func newBrowserUseError(evidence, message string) *BrowserUseError {
	return &BrowserUseError{Evidence: evidence, Message: truncateBrowserBridgeText(message, defaultBrowserUseRedactedErrorPreviewLength)}
}

func BrowserUseErrorEvidence(err error) string {
	var typed *BrowserUseError
	if errors.As(err, &typed) {
		return typed.Evidence
	}
	return ""
}

// BrowserHarnessCommandRequest describes one browser-harness execution. The
// bridge always invokes the external command as argv: browser-harness -c CODE.
type BrowserHarnessCommandRequest struct {
	Command   string
	Code      string
	TaskID    string
	Action    BrowserAction
	Env       map[string]string
	Timeout   time.Duration
	MediaType string
	Budget    ToolResultBudgetConfig
}

// BrowserHarnessProcessResult is the raw fakeable process output.
type BrowserHarnessProcessResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// BrowserHarnessProcessRunner hides os/exec behind a test fake.
type BrowserHarnessProcessRunner interface {
	Run(ctx context.Context, argv []string, env map[string]string) (BrowserHarnessProcessResult, error)
}

// BrowserHarnessCommandResult is the bounded, transcript-ready command result.
type BrowserHarnessCommandResult struct {
	Argv     []string
	Env      map[string]string
	Envelope BrowserResultEnvelope
	Evidence string
	Redacted bool
}

// BrowserHarnessBridge maps browser-harness into Gormes' browser result
// envelope contract without importing browser-harness Python packages.
type BrowserHarnessBridge struct {
	Command string
	Runner  BrowserHarnessProcessRunner
}

func (b BrowserHarnessBridge) Run(ctx context.Context, req BrowserHarnessCommandRequest) (BrowserHarnessCommandResult, error) {
	command := strings.TrimSpace(firstNonEmpty(req.Command, b.Command, defaultBrowserHarnessCommand))
	code := req.Code
	argv := []string{command, "-c", code}
	action := req.Action
	if action.Kind == "" {
		action = BrowserAction{Kind: BrowserActionSnapshot, TaskID: req.TaskID}
	}
	if action.TaskID == "" {
		action.TaskID = req.TaskID
	}
	env := buildBrowserHarnessEnv(req)
	safeEnv := redactEnv(env)
	result := BrowserHarnessCommandResult{
		Argv:     append([]string(nil), argv...),
		Env:      safeEnv,
		Redacted: true,
	}
	if strings.TrimSpace(code) == "" {
		result.Evidence = BrowserHarnessEvidenceCommandFailed
		result.Envelope = browserHarnessUnavailableEnvelope(action, "missing_code", result.Evidence)
		return result, errors.New("browser-harness: code is required")
	}
	runner := b.Runner
	if runner == nil {
		runner = browserHarnessExecRunner{}
	}
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = defaultBrowserHarnessTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	processResult, err := runner.Run(runCtx, argv, env)
	raw := combineBrowserHarnessOutput(processResult)
	raw = []byte(redactBrowserBridgeSecretsOnly(string(raw), secretEnvValues(env)...))
	if req.MediaType == "" {
		req.MediaType = "text/plain"
	}
	envelope, envelopeErr := BuildBrowserResultEnvelope(BrowserResultInput{
		Action:    action,
		Output:    raw,
		MediaType: req.MediaType,
		Budget:    req.Budget,
	})
	if envelopeErr != nil && err == nil {
		err = envelopeErr
	}
	if err != nil {
		if isBrowserHarnessUnavailable(err) {
			result.Evidence = BrowserHarnessEvidenceUnavailable
			result.Envelope = browserHarnessUnavailableEnvelope(action, err.Error(), result.Evidence)
			return result, err
		}
		result.Evidence = BrowserHarnessEvidenceCommandFailed
		envelope.Evidence = BrowserHarnessEvidenceCommandFailed
		if envelope.Tool.Code == "" {
			envelope.Tool.Code = BrowserHarnessEvidenceCommandFailed
		}
		result.Envelope = envelope
		return result, err
	}
	result.Evidence = BrowserHarnessEvidenceCommandOK
	result.Envelope = envelope
	return result, nil
}

type browserHarnessExecRunner struct{}

func (browserHarnessExecRunner) Run(ctx context.Context, argv []string, env map[string]string) (BrowserHarnessProcessResult, error) {
	if len(argv) == 0 || strings.TrimSpace(argv[0]) == "" {
		return BrowserHarnessProcessResult{}, exec.ErrNotFound
	}
	path, err := exec.LookPath(argv[0])
	if err != nil {
		return BrowserHarnessProcessResult{}, err
	}
	cmd := exec.CommandContext(ctx, path, argv[1:]...)
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	return BrowserHarnessProcessResult{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		ExitCode: browserHarnessExitCode(err),
	}, err
}

func buildBrowserHarnessEnv(req BrowserHarnessCommandRequest) map[string]string {
	env := map[string]string{}
	for k, v := range req.Env {
		env[k] = v
	}
	env["BU_NAME"] = sanitizeBrowserHarnessName(req.TaskID)
	return env
}

func sanitizeBrowserHarnessName(taskID string) string {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		taskID = defaultBrowserTaskID
	}
	var b strings.Builder
	for _, r := range taskID {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case r == '_' || r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	name := strings.Trim(b.String(), "_-")
	if name == "" {
		name = defaultBrowserTaskID
	}
	name = "gormes_" + compactUnderscores(name)
	if len(name) > 63 {
		name = strings.TrimRight(name[:63], "_-")
	}
	return name
}

func buildBrowserUseSessionName(taskID, providerSessionID string) string {
	hash := sha256.Sum256([]byte(providerSessionID))
	suffix := hex.EncodeToString(hash[:4])
	return sanitizeBrowserHarnessName(taskID) + "_" + suffix
}

func newBrowserUseCreateKey(taskID string) string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err == nil {
		return "browser-use-session-create:" + hex.EncodeToString(buf[:])
	}
	hash := sha256.Sum256([]byte(time.Now().UTC().Format(time.RFC3339Nano) + ":" + taskID))
	return "browser-use-session-create:" + hex.EncodeToString(hash[:8])
}

func combineBrowserHarnessOutput(result BrowserHarnessProcessResult) []byte {
	out := append([]byte(nil), result.Stdout...)
	if len(result.Stderr) > 0 {
		if len(out) > 0 && !bytes.HasSuffix(out, []byte("\n")) {
			out = append(out, '\n')
		}
		out = append(out, []byte("[stderr]\n")...)
		out = append(out, result.Stderr...)
	}
	if len(out) == 0 && result.ExitCode != 0 {
		out = []byte(fmt.Sprintf("browser-harness exited with code %d", result.ExitCode))
	}
	return out
}

func browserHarnessUnavailableEnvelope(action BrowserAction, reason, evidence string) BrowserResultEnvelope {
	envelope := BuildBrowserUnavailableResult(action, redactBrowserBridgeText(reason))
	envelope.Evidence = evidence
	envelope.Tool.Code = evidence
	return envelope
}

func isBrowserHarnessUnavailable(err error) bool {
	return errors.Is(err, exec.ErrNotFound) || strings.Contains(strings.ToLower(err.Error()), "executable file not found")
}

func browserHarnessExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

func redactEnv(env map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range env {
		if isSecretEnvKey(k) && v != "" {
			out[k] = "[redacted]"
			continue
		}
		out[k] = v
	}
	return out
}

func secretEnvValues(env map[string]string) []string {
	var values []string
	for k, v := range env {
		if isSecretEnvKey(k) {
			values = append(values, v)
		}
	}
	return values
}

func isSecretEnvKey(key string) bool {
	key = strings.ToUpper(key)
	for _, needle := range []string{"KEY", "TOKEN", "SECRET", "PASSWORD", "CREDENTIAL"} {
		if strings.Contains(key, needle) {
			return true
		}
	}
	return false
}

var browserBridgeURLPattern = regexp.MustCompile(`(?i)\b(?:https?|wss?)://[^\s"'<>]+`)

func redactBrowserBridgeText(text string, secrets ...string) string {
	text = redactBrowserBridgeSecretsOnly(text, secrets...)
	text = browserBridgeURLPattern.ReplaceAllString(text, "[redacted-url]")
	return truncateBrowserBridgeText(text, defaultBrowserUseRedactedErrorPreviewLength)
}

func redactBrowserBridgeSecretsOnly(text string, secrets ...string) string {
	for _, secret := range secrets {
		secret = strings.TrimSpace(secret)
		if secret != "" {
			text = strings.ReplaceAll(text, secret, "[redacted]")
		}
	}
	return text
}

func truncateBrowserBridgeText(text string, max int) string {
	if max <= 0 || len(text) <= max {
		return text
	}
	head := []byte(text)[:max]
	for len(head) > 0 && !utf8.Valid(head) {
		head = head[:len(head)-1]
	}
	return string(head) + "...[truncated]"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func compactUnderscores(s string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range s {
		if r == '_' {
			if lastUnderscore {
				continue
			}
			lastUnderscore = true
		} else {
			lastUnderscore = false
		}
		b.WriteRune(r)
	}
	return b.String()
}
