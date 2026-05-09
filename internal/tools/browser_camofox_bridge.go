package tools

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	BrowserProviderCamofox                    = "browser-camofox"
	BrowserProviderFeatureCamofox             = "camofox"
	BrowserProviderFeatureManagedPersistence  = "managed_persistence"
	BrowserProviderEvidenceCamofoxSoftCleanup = "camofox_soft_cleanup"
)

// CamofoxProviderConfig captures the Camofox REST backend mode switches. CDP
// stays explicit because Hermes gives a live /browser connect endpoint priority
// over CAMOFOX_URL.
type CamofoxProviderConfig struct {
	BaseURL            string
	CDPURL             string
	ManagedPersistence bool
	StateRoot          string
}

// CamofoxProviderBridge is a fakeable Camofox REST backend seam. It owns only
// mode, session identity, and cleanup behavior; action dispatch remains on the
// browser harness contract.
type CamofoxProviderBridge struct {
	cfg      CamofoxProviderConfig
	client   BrowserProviderHTTPClient
	mu       sync.Mutex
	sessions map[string]camofoxTrackedSession
}

// CamofoxIdentity is the profile/task identity Hermes sends to the Camofox
// REST server.
type CamofoxIdentity struct {
	UserID     string
	SessionKey string
}

type camofoxTrackedSession struct {
	identity CamofoxIdentity
	tabID    string
	managed  bool
}

func NewCamofoxProviderBridge(cfg CamofoxProviderConfig, client BrowserProviderHTTPClient) *CamofoxProviderBridge {
	if client == nil {
		client = http.DefaultClient
	}
	cfg.BaseURL = trimCamofoxBaseURL(cfg.BaseURL)
	cfg.CDPURL = strings.TrimSpace(cfg.CDPURL)
	return &CamofoxProviderBridge{cfg: cfg, client: client, sessions: map[string]camofoxTrackedSession{}}
}

func CamofoxProviderConfigFromEnv(lookup func(string) string) CamofoxProviderConfig {
	if lookup == nil {
		lookup = os.Getenv
	}
	home := strings.TrimSpace(lookup("GORMES_HOME"))
	stateRoot := ""
	if home != "" {
		stateRoot = filepath.Join(home, "browser_auth", "camofox")
	}
	return CamofoxProviderConfig{
		BaseURL:   trimCamofoxBaseURL(lookup("CAMOFOX_URL")),
		CDPURL:    strings.TrimSpace(lookup("BROWSER_CDP_URL")),
		StateRoot: stateRoot,
	}
}

func (c *CamofoxProviderBridge) Configured() bool {
	return c != nil && strings.TrimSpace(c.cfg.BaseURL) != "" && strings.TrimSpace(c.cfg.CDPURL) == ""
}

func CamofoxManagedIdentity(stateRoot, taskID string) CamofoxIdentity {
	stateRoot = strings.TrimSpace(stateRoot)
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		taskID = "default"
	}
	userDigest := uuid5Hex("camofox-user:" + stateRoot)
	sessionDigest := uuid5Hex("camofox-session:" + stateRoot + ":" + taskID)
	return CamofoxIdentity{
		UserID:     "hermes_" + userDigest[:10],
		SessionKey: "task_" + sessionDigest[:16],
	}
}

func (c *CamofoxProviderBridge) CreateSession(ctx context.Context, req BrowserProviderSessionRequest) (BrowserProviderSession, error) {
	if !c.Configured() {
		return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceUnconfigured, "Camofox requires CAMOFOX_URL and no BROWSER_CDP_URL override")
	}
	taskID := normalizeBrowserTaskID(req.TaskID)
	identity := c.identityForTask(taskID)
	body, err := json.Marshal(map[string]string{
		"userId":     identity.UserID,
		"sessionKey": identity.SessionKey,
		"url":        "about:blank",
	})
	if err != nil {
		return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceCreateFailed, err.Error())
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/tabs", bytes.NewReader(body))
	if err != nil {
		return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceCreateFailed, c.redact("", err.Error()))
	}
	httpReq.Header.Set("Content-Type", "application/json")
	respBody, status, err := doBrowserProviderRequest(c.client, httpReq)
	if err != nil {
		return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceCreateFailed, c.redact("", err.Error()))
	}
	if status < 200 || status >= 300 {
		msg := fmt.Sprintf("Camofox tab create failed: HTTP %d %s", status, string(respBody))
		return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceCreateFailed, c.redact("", msg))
	}
	var payload camofoxCreateTabResponse
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceCreateFailed, c.redact("", "decode Camofox tab response: "+err.Error()))
	}
	if strings.TrimSpace(payload.TabID) == "" {
		return BrowserProviderSession{}, newBrowserProviderError(BrowserProviderEvidenceCreateFailed, "Camofox tab response missing tabId")
	}
	c.mu.Lock()
	c.sessions[taskID] = camofoxTrackedSession{identity: identity, tabID: payload.TabID, managed: c.cfg.ManagedPersistence}
	c.mu.Unlock()
	return BrowserProviderSession{
		ProviderName:           BrowserProviderCamofox,
		SessionName:            buildBrowserUseSessionName(taskID, payload.TabID),
		ProviderSessionID:      payload.TabID,
		CompatibilitySessionID: identity.SessionKey,
		Features: map[string]bool{
			BrowserProviderFeatureCamofox:            true,
			BrowserProviderFeatureManagedPersistence: c.cfg.ManagedPersistence,
		},
		Evidence: []string{BrowserProviderEvidenceSessionCreated},
		Redacted: true,
	}, nil
}

func (c *CamofoxProviderBridge) SoftCleanup(taskID string) BrowserProviderCleanupResult {
	if c == nil {
		return BrowserProviderCleanupResult{Evidence: BrowserProviderEvidenceCleanupSkipped, Redacted: true}
	}
	taskID = normalizeBrowserTaskID(taskID)
	c.mu.Lock()
	session, ok := c.sessions[taskID]
	if ok && session.managed {
		delete(c.sessions, taskID)
	}
	c.mu.Unlock()
	if !ok || !session.managed {
		return BrowserProviderCleanupResult{Evidence: BrowserProviderEvidenceCleanupSkipped, Redacted: true}
	}
	return BrowserProviderCleanupResult{Evidence: BrowserProviderEvidenceCamofoxSoftCleanup, Redacted: true}
}

func (c *CamofoxProviderBridge) CloseTaskSession(ctx context.Context, taskID string) (BrowserProviderCleanupResult, error) {
	if c == nil {
		return BrowserProviderCleanupResult{Stopped: true, Evidence: BrowserProviderEvidenceCleanupOK, Redacted: true}, nil
	}
	taskID = normalizeBrowserTaskID(taskID)
	c.mu.Lock()
	session, ok := c.sessions[taskID]
	if ok {
		delete(c.sessions, taskID)
	}
	c.mu.Unlock()
	if !ok {
		return BrowserProviderCleanupResult{Stopped: true, Evidence: BrowserProviderEvidenceCleanupOK, Redacted: true}, nil
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.cfg.BaseURL+"/sessions/"+url.PathEscape(session.identity.UserID), nil)
	if err != nil {
		return BrowserProviderCleanupResult{Stopped: true, Evidence: BrowserProviderEvidenceCleanupFailed, Redacted: true}, newBrowserProviderError(BrowserProviderEvidenceCleanupFailed, c.redact(session.identity.UserID, err.Error()))
	}
	respBody, status, err := doBrowserProviderRequest(c.client, httpReq)
	if err != nil {
		return BrowserProviderCleanupResult{Stopped: true, Evidence: BrowserProviderEvidenceCleanupFailed, Redacted: true}, newBrowserProviderError(BrowserProviderEvidenceCleanupFailed, c.redact(session.identity.UserID, err.Error()))
	}
	if status < 200 || status >= 300 {
		msg := fmt.Sprintf("Camofox cleanup failed: HTTP %d %s", status, string(respBody))
		return BrowserProviderCleanupResult{Stopped: true, Evidence: BrowserProviderEvidenceCleanupFailed, Redacted: true}, newBrowserProviderError(BrowserProviderEvidenceCleanupFailed, c.redact(session.identity.UserID, msg))
	}
	return BrowserProviderCleanupResult{Stopped: true, Evidence: BrowserProviderEvidenceCleanupOK, Redacted: true}, nil
}

func trimCamofoxBaseURL(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}

type camofoxCreateTabResponse struct {
	TabID string `json:"tabId"`
}

func (c *CamofoxProviderBridge) identityForTask(taskID string) CamofoxIdentity {
	if c.cfg.ManagedPersistence {
		return CamofoxManagedIdentity(c.cfg.StateRoot, taskID)
	}
	return CamofoxIdentity{
		UserID:     "hermes_" + randomCamofoxHex(10),
		SessionKey: "task_" + randomCamofoxHex(16),
	}
}

func (c *CamofoxProviderBridge) redact(sessionID, text string) string {
	if c == nil {
		return redactBrowserProviderText(text, sessionID)
	}
	return redactBrowserProviderText(text, c.cfg.BaseURL, c.cfg.StateRoot, sessionID)
}

func uuid5Hex(name string) string {
	namespace, _ := hex.DecodeString("6ba7b8119dad11d180b400c04fd430c8")
	h := sha1.New()
	_, _ = h.Write(namespace)
	_, _ = h.Write([]byte(name))
	sum := h.Sum(nil)
	uuid := make([]byte, 16)
	copy(uuid, sum[:16])
	uuid[6] = (uuid[6] & 0x0f) | 0x50
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return hex.EncodeToString(uuid)
}

func randomCamofoxHex(chars int) string {
	if chars <= 0 {
		return ""
	}
	bytesNeeded := (chars + 1) / 2
	buf := make([]byte, bytesNeeded)
	if _, err := randRead(buf); err != nil {
		return strings.Repeat("0", chars)
	}
	return hex.EncodeToString(buf)[:chars]
}

var randRead = func(buf []byte) (int, error) {
	return rand.Reader.Read(buf)
}
