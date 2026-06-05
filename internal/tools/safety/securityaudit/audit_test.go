package securityaudit

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestSecurityAuditDeepCoversEveryCategoryWithoutLiveGateway(t *testing.T) {
	const apiSecret = "sk-live-secret"
	const botSecret = "123456:telegram-secret"

	result := AuditSecurity(SecurityAuditRequest{
		Deep: true,
		GatewayAuth: SecurityAuditGatewayAuth{
			TokenConfigured: false,
			Exposure:        "local",
			Probe: SecurityAuditProbe{
				Required:  true,
				Available: false,
				Message:   "gateway process is not running",
			},
		},
		State: SecurityAuditState{
			Files: []SecurityAuditStateFile{
				{Path: "/work/sessions.db", Exists: true, Valid: true},
				{Path: "/work/gateway_state.json", Exists: false, Valid: false},
			},
		},
		Channels: []SecurityAuditChannel{
			{Name: "telegram", Enabled: true, TokenConfigured: true, AllowedScopeConfigured: false, FirstRunDiscovery: true},
		},
		Filesystem: SecurityAuditFilesystem{
			CWD:              "/work",
			ProbeReadPath:    "/etc/passwd",
			ProbeWritePath:   "../outside/result.txt",
			ReadPaths:        nil,
			WritePaths:       nil,
			ScopeConfigured:  true,
			ExpectDenyProbes: true,
		},
		CredentialRedaction: SecurityAuditCredentialRedaction{
			Secrets: []string{apiSecret, botSecret},
			Samples: []string{
				"api_key=" + apiSecret,
				"telegram token " + botSecret,
			},
		},
		FixCandidates: []SecurityAuditFixCandidate{
			{
				Code:        SecurityAuditFixFilePermissions,
				Category:    SecurityAuditCategoryStateIntegrity,
				Path:        "/work/config.toml",
				CurrentMode: 0o644,
				DesiredMode: 0o600,
				Safe:        true,
			},
		},
	})

	if !result.OK {
		t.Fatalf("OK = false, want true because degraded warnings should not block the whole audit: %+v", result)
	}
	if result.Summary.Pass == 0 || result.Summary.Warn == 0 || result.Summary.Fail != 0 {
		t.Fatalf("summary = %+v, want pass and warn counts with no failures", result.Summary)
	}

	statuses := securityAuditStatusesByCategory(result)
	for _, category := range []string{
		SecurityAuditCategoryGatewayAuth,
		SecurityAuditCategoryStateIntegrity,
		SecurityAuditCategoryChannelSecurity,
		SecurityAuditCategoryShellBlocklist,
		SecurityAuditCategoryFilesystemScoping,
		SecurityAuditCategoryCredentialRedaction,
	} {
		if statuses[category] == "" {
			t.Fatalf("missing category %s in result: %+v", category, result.Categories)
		}
	}
	if statuses[SecurityAuditCategoryGatewayAuth] != SecurityAuditStatusWarn {
		t.Fatalf("gateway_auth status = %q, want warn", statuses[SecurityAuditCategoryGatewayAuth])
	}
	if statuses[SecurityAuditCategoryShellBlocklist] != SecurityAuditStatusPass {
		t.Fatalf("shell_blocklist status = %q, want pass", statuses[SecurityAuditCategoryShellBlocklist])
	}
	if statuses[SecurityAuditCategoryFilesystemScoping] != SecurityAuditStatusPass {
		t.Fatalf("filesystem_scoping status = %q, want pass", statuses[SecurityAuditCategoryFilesystemScoping])
	}
	if statuses[SecurityAuditCategoryCredentialRedaction] != SecurityAuditStatusPass {
		t.Fatalf("credential_redaction status = %q, want pass", statuses[SecurityAuditCategoryCredentialRedaction])
	}
	if !securityAuditFindingPresent(result.Findings, SecurityAuditFindingGatewayProbeUnavailable) {
		t.Fatalf("findings = %+v, missing gateway probe degraded finding", result.Findings)
	}
	if len(result.Fixes) != 1 || result.Fixes[0].Applied {
		t.Fatalf("fixes = %+v, want pending safe fix because --fix was not set", result.Fixes)
	}
	if !result.Redacted {
		t.Fatalf("Redacted = false, want true")
	}

	body, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if strings.Contains(string(body), apiSecret) || strings.Contains(string(body), botSecret) {
		t.Fatalf("audit JSON leaked secret:\n%s", body)
	}
}

func TestSecurityAuditDeepReportsSecretRefAvailabilityWithoutLeakingResolvedValues(t *testing.T) {
	const resolvedSecret = "sk-resolved-secret"
	result := AuditSecurity(SecurityAuditRequest{
		Deep: true,
		SecretRefs: []SecurityAuditSecretRef{
			{
				Path:         "hermes.api_key",
				Active:       true,
				Available:    true,
				Source:       "env",
				Provider:     "default",
				ID:           "CUSTOM_PROVIDER_SECRET",
				EvidenceCode: "secret_ref_resolved",
				Message:      "resolved provider secret " + resolvedSecret,
			},
			{
				Path:         "telegram.bot_token",
				Active:       true,
				Available:    false,
				Source:       "env",
				Provider:     "default",
				ID:           "MISSING_TELEGRAM_TOKEN",
				EvidenceCode: "secret_ref_missing",
				Message:      "missing env secret " + resolvedSecret,
			},
			{
				Path:         "slack.app_token",
				Active:       true,
				Available:    false,
				Source:       "exec",
				Provider:     "default",
				ID:           "secret-helper",
				EvidenceCode: "secret_ref_unsupported",
				Message:      "exec SecretRef providers are not supported",
			},
		},
		CredentialRedaction: SecurityAuditCredentialRedaction{
			Secrets: []string{resolvedSecret},
			Samples: []string{"api_key=" + resolvedSecret},
		},
	})

	statuses := securityAuditStatusesByCategory(result)
	if statuses[SecurityAuditCategorySecretRefs] != SecurityAuditStatusWarn {
		t.Fatalf("secret_refs status = %q, want warn: %+v", statuses[SecurityAuditCategorySecretRefs], result.Categories)
	}
	if !securityAuditFindingPresent(result.Findings, SecurityAuditFindingSecretRefUnavailable) {
		t.Fatalf("findings = %+v, missing unavailable SecretRef finding", result.Findings)
	}
	if !securityAuditFindingPresent(result.Findings, SecurityAuditFindingSecretRefUnsupported) {
		t.Fatalf("findings = %+v, missing unsupported SecretRef finding", result.Findings)
	}
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if strings.Contains(string(body), resolvedSecret) {
		t.Fatalf("audit JSON leaked resolved SecretRef value:\n%s", body)
	}
}

func TestSecurityAuditFixAppliesOnlySafeDeterministicRemediations(t *testing.T) {
	var applied []SecurityAuditFixCandidate
	result := AuditSecurity(SecurityAuditRequest{
		Fix: true,
		GatewayAuth: SecurityAuditGatewayAuth{
			TokenConfigured:       false,
			Exposure:              "local",
			GenerateTokenWhenFix:  true,
			GeneratedTokenPath:    "GATEWAY_PROXY_KEY",
			GeneratedTokenMessage: "write gateway bearer token to dotenv",
		},
		TokenGenerator: func() (string, error) {
			return "generated-secret-value", nil
		},
		FixCandidates: []SecurityAuditFixCandidate{
			{
				Code:        SecurityAuditFixFilePermissions,
				Category:    SecurityAuditCategoryStateIntegrity,
				Path:        "/work/config.toml",
				CurrentMode: 0o644,
				DesiredMode: 0o600,
				Safe:        true,
			},
			{
				Code:     "delete_state_file",
				Category: SecurityAuditCategoryStateIntegrity,
				Path:     "/work/sessions.db",
				Safe:     false,
				Message:  "delete state file",
			},
		},
		FixApplier: func(candidate SecurityAuditFixCandidate) error {
			applied = append(applied, candidate)
			return nil
		},
	})

	if !result.OK {
		t.Fatalf("OK = false, want true after safe fixes: %+v", result)
	}
	if len(applied) != 1 || applied[0].Code != SecurityAuditFixFilePermissions {
		t.Fatalf("applied callbacks = %+v, want only the safe file permission fix", applied)
	}
	if !securityAuditFixApplied(result.Fixes, SecurityAuditFixFilePermissions) {
		t.Fatalf("fixes = %+v, missing applied file permission fix", result.Fixes)
	}
	if !securityAuditFixApplied(result.Fixes, SecurityAuditFixGatewayAuthTokenGenerated) {
		t.Fatalf("fixes = %+v, missing applied gateway auth token generation", result.Fixes)
	}
	if securityAuditFixApplied(result.Fixes, "delete_state_file") {
		t.Fatalf("fixes = %+v, unsafe fix was applied", result.Fixes)
	}
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if strings.Contains(string(body), "generated-secret-value") {
		t.Fatalf("audit JSON leaked generated token:\n%s", body)
	}
}

func TestSecurityAuditFixReportsSafeFixFailuresWithoutStoppingAudit(t *testing.T) {
	result := AuditSecurity(SecurityAuditRequest{
		Fix: true,
		FixCandidates: []SecurityAuditFixCandidate{
			{
				Code:        SecurityAuditFixFilePermissions,
				Category:    SecurityAuditCategoryStateIntegrity,
				Path:        "/work/config.toml",
				CurrentMode: 0o644,
				DesiredMode: 0o600,
				Safe:        true,
			},
		},
		FixApplier: func(SecurityAuditFixCandidate) error {
			return errors.New("permission denied")
		},
	})

	if result.OK {
		t.Fatalf("OK = true, want false when a safe fix fails: %+v", result)
	}
	if !securityAuditFindingPresent(result.Findings, SecurityAuditFindingFixFailed) {
		t.Fatalf("findings = %+v, missing fix_failed finding", result.Findings)
	}
}

func securityAuditStatusesByCategory(result SecurityAuditResult) map[string]string {
	statuses := make(map[string]string, len(result.Categories))
	for _, category := range result.Categories {
		statuses[category.Name] = category.Status
	}
	return statuses
}

func securityAuditFindingPresent(findings []SecurityAuditFinding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func securityAuditFixApplied(fixes []SecurityAuditFix, code string) bool {
	for _, fix := range fixes {
		if fix.Code == code && fix.Applied {
			return true
		}
	}
	return false
}
