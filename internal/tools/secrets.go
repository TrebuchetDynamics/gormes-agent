package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

const (
	SecretRefSourceEnv  = "env"
	SecretRefSourceFile = "file"
	SecretRefSourceExec = "exec"

	DefaultSecretProviderAlias = "default"

	SecretsEvidenceApplied      = "secrets_applied"
	SecretsEvidenceConfigured   = "secrets_configured"
	SecretsEvidenceReloaded     = "secrets_reloaded"
	SecretsEvidenceUnavailable  = "secrets_unavailable"
	SecretsEvidenceAuditPassed  = "secrets_audit_passed"
	SecretsEvidenceAuditFinding = "secrets_audit_finding"

	SecretsFindingPlaintext       = "plaintext_secret"
	SecretsFindingUnresolvedRef   = "unresolved_secret_ref"
	SecretsFindingPrecedenceDrift = "precedence_drift"
	SecretsFindingInvalidTarget   = "invalid_secret_target"
)

type SecretRef struct {
	Source   string `json:"source"`
	Provider string `json:"provider"`
	ID       string `json:"id"`
}

type SecretRefEvidence struct {
	Code     string `json:"code"`
	Source   string `json:"source,omitempty"`
	Provider string `json:"provider,omitempty"`
	ID       string `json:"id,omitempty"`
	Redacted bool   `json:"redacted"`
}

type SecretResolver interface {
	ResolveSecretString(ref SecretRef) (string, SecretRefEvidence, error)
}

type SecretResolverFunc func(ref SecretRef) (string, SecretRefEvidence, error)

func (f SecretResolverFunc) ResolveSecretString(ref SecretRef) (string, SecretRefEvidence, error) {
	return f(ref)
}

type SecretsPlan struct {
	Targets []SecretTarget `json:"targets"`
}

type SecretTarget struct {
	Path      string    `json:"path"`
	Ref       SecretRef `json:"ref"`
	Required  bool      `json:"required"`
	Plaintext string    `json:"plaintext,omitempty"`
}

type SecretsRuntimeEntry struct {
	Path     string            `json:"path"`
	Ref      SecretRef         `json:"ref"`
	Resolved bool              `json:"resolved"`
	Value    string            `json:"-"`
	Evidence SecretRefEvidence `json:"evidence"`
}

type SecretsRuntimeSnapshot struct {
	Generation int                            `json:"generation"`
	Entries    map[string]SecretsRuntimeEntry `json:"entries"`
	Redacted   bool                           `json:"redacted"`
}

type SecretsRuntimeControllerConfig struct {
	Resolver        SecretResolver
	InitialSnapshot *SecretsRuntimeSnapshot
}

type SecretsRuntimeController struct {
	mu       sync.RWMutex
	resolver SecretResolver
	snapshot SecretsRuntimeSnapshot
}

func NewSecretsRuntimeController(cfg SecretsRuntimeControllerConfig) *SecretsRuntimeController {
	resolver := cfg.Resolver
	if resolver == nil {
		resolver = unavailableSecretResolver{}
	}
	snapshot := cloneSecretsRuntimeSnapshot(cfg.InitialSnapshot)
	if snapshot.Entries == nil {
		snapshot.Entries = map[string]SecretsRuntimeEntry{}
	}
	snapshot.Redacted = true
	return &SecretsRuntimeController{resolver: resolver, snapshot: snapshot}
}

func (c *SecretsRuntimeController) Snapshot() SecretsRuntimeSnapshot {
	if c == nil {
		return SecretsRuntimeSnapshot{Entries: map[string]SecretsRuntimeEntry{}, Redacted: true}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return cloneSecretsRuntimeSnapshot(&c.snapshot)
}

func (c *SecretsRuntimeController) Apply(ctx context.Context, plan SecretsPlan) (SecretsApplyResult, error) {
	return c.swapSnapshot(ctx, plan, SecretsEvidenceApplied)
}

func (c *SecretsRuntimeController) Reload(ctx context.Context, plan SecretsPlan) (SecretsApplyResult, error) {
	return c.swapSnapshot(ctx, plan, SecretsEvidenceReloaded)
}

type SecretsApplyResult struct {
	Code     string                 `json:"code"`
	Snapshot SecretsRuntimeSnapshot `json:"snapshot"`
	Findings []SecretsAuditFinding  `json:"findings,omitempty"`
	Redacted bool                   `json:"redacted"`
}

func (c *SecretsRuntimeController) swapSnapshot(ctx context.Context, plan SecretsPlan, successCode string) (SecretsApplyResult, error) {
	if c == nil {
		return SecretsApplyResult{Code: SecretsEvidenceUnavailable, Redacted: true}, fmt.Errorf("%s: nil runtime controller", SecretsEvidenceUnavailable)
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	next, findings := buildSecretsSnapshot(ctx, c.resolver, plan, c.snapshot.Generation+1)
	if hasBlockingSecretsFinding(findings) {
		current := cloneSecretsRuntimeSnapshot(&c.snapshot)
		result := SecretsApplyResult{Code: SecretsEvidenceUnavailable, Snapshot: current, Findings: findings, Redacted: true}
		return result, fmt.Errorf("%s path=%s", SecretsEvidenceUnavailable, firstSecretsFindingPath(findings))
	}
	c.snapshot = next
	return SecretsApplyResult{Code: successCode, Snapshot: cloneSecretsRuntimeSnapshot(&c.snapshot), Redacted: true}, nil
}

type SecretsAuditRequest struct {
	Resolver         SecretResolver
	Plan             SecretsPlan
	PreviousSnapshot *SecretsRuntimeSnapshot
}

type SecretsAuditResult struct {
	Code     string                `json:"code"`
	OK       bool                  `json:"ok"`
	Findings []SecretsAuditFinding `json:"findings"`
	Redacted bool                  `json:"redacted"`
}

type SecretsAuditFinding struct {
	Code     string            `json:"code"`
	Path     string            `json:"path,omitempty"`
	Severity string            `json:"severity"`
	Message  string            `json:"message"`
	Ref      SecretRef         `json:"ref,omitempty"`
	Evidence SecretRefEvidence `json:"evidence,omitempty"`
	Redacted bool              `json:"redacted"`
}

func AuditSecrets(ctx context.Context, req SecretsAuditRequest) SecretsAuditResult {
	resolver := req.Resolver
	if resolver == nil {
		resolver = unavailableSecretResolver{}
	}
	var findings []SecretsAuditFinding
	for _, target := range req.Plan.Targets {
		target = normalizeSecretTarget(target)
		if target.Path == "" {
			findings = append(findings, invalidSecretTargetFinding(target, "secret target path is required"))
			continue
		}
		if strings.TrimSpace(target.Plaintext) != "" {
			findings = append(findings, SecretsAuditFinding{
				Code:     SecretsFindingPlaintext,
				Path:     target.Path,
				Severity: "error",
				Message:  "plaintext secret is present; replace it with a SecretRef",
				Ref:      target.Ref,
				Redacted: true,
			})
		}
		if _, evidence, err := resolver.ResolveSecretString(target.Ref); err != nil {
			findings = append(findings, SecretsAuditFinding{
				Code:     SecretsFindingUnresolvedRef,
				Path:     target.Path,
				Severity: secretsFindingSeverity(target.Required),
				Message:  "SecretRef could not be resolved",
				Ref:      target.Ref,
				Evidence: evidence,
				Redacted: true,
			})
		}
		if req.PreviousSnapshot != nil {
			if previous, ok := req.PreviousSnapshot.Entries[target.Path]; ok && !sameSecretRef(previous.Ref, target.Ref) {
				findings = append(findings, SecretsAuditFinding{
					Code:     SecretsFindingPrecedenceDrift,
					Path:     target.Path,
					Severity: "warn",
					Message:  "active runtime SecretRef differs from the planned SecretRef",
					Ref:      target.Ref,
					Evidence: SecretRefEvidence{Code: SecretsFindingPrecedenceDrift, Source: previous.Ref.Source, Provider: previous.Ref.Provider, ID: previous.Ref.ID, Redacted: true},
					Redacted: true,
				})
			}
		}
	}
	if len(findings) > 0 {
		return SecretsAuditResult{Code: SecretsEvidenceAuditFinding, OK: false, Findings: findings, Redacted: true}
	}
	return SecretsAuditResult{Code: SecretsEvidenceAuditPassed, OK: true, Findings: []SecretsAuditFinding{}, Redacted: true}
}

type SecretsConfigureRequest struct {
	Resolver SecretResolver
	Path     string
	Required bool
	Ref      SecretRef
}

type SecretsConfigureResult struct {
	Code        string            `json:"code"`
	Target      SecretTarget      `json:"target"`
	PreflightOK bool              `json:"preflight_ok"`
	Evidence    SecretRefEvidence `json:"evidence"`
	Redacted    bool              `json:"redacted"`
}

func ConfigureSecretRef(ctx context.Context, req SecretsConfigureRequest) (SecretsConfigureResult, error) {
	resolver := req.Resolver
	if resolver == nil {
		resolver = unavailableSecretResolver{}
	}
	target := normalizeSecretTarget(SecretTarget{Path: req.Path, Ref: req.Ref, Required: req.Required})
	if target.Path == "" {
		result := SecretsConfigureResult{Code: SecretsEvidenceUnavailable, Target: target, Redacted: true}
		return result, fmt.Errorf("%s path=<empty>", SecretsEvidenceUnavailable)
	}
	_, evidence, err := resolver.ResolveSecretString(target.Ref)
	result := SecretsConfigureResult{
		Code:        SecretsEvidenceConfigured,
		Target:      target,
		PreflightOK: err == nil,
		Evidence:    evidence,
		Redacted:    true,
	}
	if err != nil && target.Required {
		result.Code = SecretsEvidenceUnavailable
		return result, fmt.Errorf("%s path=%s", SecretsEvidenceUnavailable, target.Path)
	}
	return result, nil
}

func buildSecretsSnapshot(ctx context.Context, resolver SecretResolver, plan SecretsPlan, generation int) (SecretsRuntimeSnapshot, []SecretsAuditFinding) {
	if resolver == nil {
		resolver = unavailableSecretResolver{}
	}
	snapshot := SecretsRuntimeSnapshot{
		Generation: generation,
		Entries:    make(map[string]SecretsRuntimeEntry, len(plan.Targets)),
		Redacted:   true,
	}
	var findings []SecretsAuditFinding
	for _, target := range plan.Targets {
		target = normalizeSecretTarget(target)
		if target.Path == "" {
			findings = append(findings, invalidSecretTargetFinding(target, "secret target path is required"))
			continue
		}
		value, evidence, err := resolver.ResolveSecretString(target.Ref)
		entry := SecretsRuntimeEntry{
			Path:     target.Path,
			Ref:      target.Ref,
			Resolved: err == nil,
			Evidence: evidence,
		}
		if err == nil {
			entry.Value = value
			snapshot.Entries[target.Path] = entry
			continue
		}
		if target.Required {
			findings = append(findings, SecretsAuditFinding{
				Code:     SecretsFindingUnresolvedRef,
				Path:     target.Path,
				Severity: "error",
				Message:  "SecretRef could not be resolved",
				Ref:      target.Ref,
				Evidence: evidence,
				Redacted: true,
			})
		} else {
			snapshot.Entries[target.Path] = entry
		}
	}
	return snapshot, findings
}

func normalizeSecretTarget(target SecretTarget) SecretTarget {
	target.Path = strings.TrimSpace(target.Path)
	target.Ref.Source = strings.ToLower(strings.TrimSpace(target.Ref.Source))
	target.Ref.Provider = strings.TrimSpace(target.Ref.Provider)
	if target.Ref.Provider == "" {
		target.Ref.Provider = DefaultSecretProviderAlias
	}
	target.Ref.ID = strings.TrimSpace(target.Ref.ID)
	return target
}

func invalidSecretTargetFinding(target SecretTarget, message string) SecretsAuditFinding {
	return SecretsAuditFinding{
		Code:     SecretsFindingInvalidTarget,
		Path:     target.Path,
		Severity: "error",
		Message:  message,
		Ref:      target.Ref,
		Redacted: true,
	}
}

func secretsFindingSeverity(required bool) string {
	if required {
		return "error"
	}
	return "warn"
}

func hasBlockingSecretsFinding(findings []SecretsAuditFinding) bool {
	for _, finding := range findings {
		if finding.Severity == "error" {
			return true
		}
	}
	return false
}

func firstSecretsFindingPath(findings []SecretsAuditFinding) string {
	for _, finding := range findings {
		if finding.Path != "" {
			return finding.Path
		}
	}
	return "<unknown>"
}

func sameSecretRef(a, b SecretRef) bool {
	aTarget := normalizeSecretTarget(SecretTarget{Ref: a})
	bTarget := normalizeSecretTarget(SecretTarget{Ref: b})
	return aTarget.Ref.Source == bTarget.Ref.Source &&
		aTarget.Ref.Provider == bTarget.Ref.Provider &&
		aTarget.Ref.ID == bTarget.Ref.ID
}

type unavailableSecretResolver struct{}

func (unavailableSecretResolver) ResolveSecretString(ref SecretRef) (string, SecretRefEvidence, error) {
	evidence := SecretRefEvidence{
		Code:     SecretsEvidenceUnavailable,
		Source:   ref.Source,
		Provider: ref.Provider,
		ID:       ref.ID,
		Redacted: true,
	}
	return "", evidence, fmt.Errorf("%s source=%s provider=%s id=%s", SecretsEvidenceUnavailable, ref.Source, ref.Provider, ref.ID)
}

func cloneSecretsRuntimeSnapshot(snapshot *SecretsRuntimeSnapshot) SecretsRuntimeSnapshot {
	out := SecretsRuntimeSnapshot{Redacted: true}
	if snapshot == nil {
		out.Entries = map[string]SecretsRuntimeEntry{}
		return out
	}
	out.Generation = snapshot.Generation
	out.Redacted = true
	out.Entries = make(map[string]SecretsRuntimeEntry, len(snapshot.Entries))
	for path, entry := range snapshot.Entries {
		out.Entries[path] = entry
	}
	return out
}
