package security

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestWriteAuditResultJSONIncludesBuildProvenance(t *testing.T) {
	result := toolspkg.SecurityAuditResult{
		Code: "security_audit",
		OK:   true,
		Summary: toolspkg.SecurityAuditSummary{
			Pass: 2,
		},
	}
	var out bytes.Buffer
	WriteAuditResult(&out, result, true, BuildProvenance{Version: "v-test", GitCommit: "abc123"})

	var got struct {
		Build BuildProvenance `json:"build"`
		Code  string          `json:"code"`
		OK    bool            `json:"ok"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if got.Build.Version != "v-test" || got.Build.GitCommit != "abc123" {
		t.Fatalf("build = %+v, want v-test/abc123", got.Build)
	}
	if got.Code != result.Code || !got.OK {
		t.Fatalf("result fields = code %q ok %t, want %q true", got.Code, got.OK, result.Code)
	}
}

func TestWriteAuditResultHumanPreservesLineShape(t *testing.T) {
	result := toolspkg.SecurityAuditResult{
		Code:     "security_audit",
		OK:       false,
		Redacted: true,
		Summary:  toolspkg.SecurityAuditSummary{Pass: 1, Warn: 2, Fail: 3, Fixed: 4},
		Categories: []toolspkg.SecurityAuditCategoryResult{{
			Name:   "gateway_auth",
			Status: "fail",
		}},
		Findings: []toolspkg.SecurityAuditFinding{{
			Code:     "gateway_auth_missing",
			Category: "gateway_auth",
			Severity: "fail",
			Path:     "GATEWAY_PROXY_KEY",
			Action:   "set token",
			Redacted: true,
		}},
		Fixes: []toolspkg.SecurityAuditFix{{
			Code:     "file_permissions",
			Category: "state_integrity",
			Path:     "/tmp/config.toml",
			Applied:  true,
			Safe:     true,
			Redacted: true,
		}},
	}
	var out bytes.Buffer
	WriteAuditResult(&out, result, false, BuildProvenance{})
	text := out.String()
	for _, want := range []string{
		"security_audit ok=false pass=1 warn=2 fail=3 fixed=4 redacted=true",
		"gateway_auth status=fail",
		"gateway_auth_missing category=gateway_auth severity=fail path=GATEWAY_PROXY_KEY action=set token redacted=true",
		"file_permissions category=state_integrity applied=true safe=true path=/tmp/config.toml redacted=true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("human output missing %q:\n%s", want, text)
		}
	}
}

func TestAppendUniqueSecurityAuditSecretsTrimsAndDeduplicates(t *testing.T) {
	got := appendUniqueSecurityAuditSecrets([]string{" alpha ", "", "beta"}, "alpha", " gamma ", "beta")
	want := []string{"alpha", "beta", "gamma"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("appendUniqueSecurityAuditSecrets() = %#v, want %#v", got, want)
	}
}
