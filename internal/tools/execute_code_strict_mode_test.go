package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestStrictMode_Contract(t *testing.T) {
	t.Run("isolated_temp_dir", func(t *testing.T) {
		s := NewStrictModeSandbox()
		ctx := context.Background()
		result, err := s.Execute(ctx, CodeExecutionRequest{
			Language: "sh",
			Code:     "echo $PWD",
			Timeout:  5 * time.Second,
		})
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if result.Status != "success" {
			t.Fatalf("Status = %q, want success; error=%q", result.Status, result.Error)
		}
		if !strings.Contains(result.Stdout, "gormes-execute-code-") {
			t.Errorf("stdout missing temp dir marker: %q", result.Stdout)
		}
		cwd, _ := os.Getwd()
		if strings.TrimSpace(result.Stdout) == cwd {
			t.Errorf("execution ran in cwd %q instead of isolated temp dir", cwd)
		}
	})

	t.Run("canonical_interpreter", func(t *testing.T) {
		var calledWith string
		fakeLookPath := func(file string) (string, error) {
			calledWith = file
			return "/usr/bin/sh", nil
		}
		s := newStrictModeSandboxWithLookPath(fakeLookPath)
		ctx := context.Background()
		result, err := s.Execute(ctx, CodeExecutionRequest{
			Language: "sh",
			Code:     "echo ok",
			Timeout:  5 * time.Second,
		})
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if result.Status != "success" {
			t.Fatalf("Status = %q, want success; error=%q", result.Status, result.Error)
		}
		if calledWith != "sh" {
			t.Errorf("lookPath called with %q, want sh (canonical system PATH, no venv/conda)", calledWith)
		}
	})

	t.Run("blocked_result_envelope_unchanged", func(t *testing.T) {
		s := NewStrictModeSandbox()
		ctx := context.Background()
		result, err := s.Execute(ctx, CodeExecutionRequest{
			Language: "sh",
			Code:     "curl https://evil.com",
			Timeout:  5 * time.Second,
		})
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if result.Status != "blocked" {
			t.Fatalf("Status = %q, want blocked", result.Status)
		}
		if result.FilesystemAccess {
			t.Errorf("FilesystemAccess = true, want false for blocked result")
		}
		if result.NetworkAccess {
			t.Errorf("NetworkAccess = true, want false for blocked result")
		}

		raw, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		var envelope map[string]interface{}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
		for _, field := range []string{"status", "error", "filesystem_access", "network_access"} {
			if _, ok := envelope[field]; !ok {
				t.Errorf("blocked result JSON missing field %q; envelope=%v", field, envelope)
			}
		}
	})

	t.Run("relative_imports_fail_outside_tempdir", func(t *testing.T) {
		s := NewStrictModeSandbox()
		ctx := context.Background()
		result, err := s.Execute(ctx, CodeExecutionRequest{
			Language: "sh",
			Code:     "cat /etc/hostname 2>/dev/null || echo blocked",
			Timeout:  5 * time.Second,
		})
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if result.Status != "blocked" {
			t.Fatalf("Status = %q, want blocked (cat blocked by sandbox guard); error=%q", result.Status, result.Error)
		}
		if result.FilesystemAccess {
			t.Errorf("FilesystemAccess = true, want false (cat outside tempdir blocked)")
		}
	})

	t.Run("no_host_filesystem_access", func(t *testing.T) {
		s := NewStrictModeSandbox()
		ctx := context.Background()
		result, err := s.Execute(ctx, CodeExecutionRequest{
			Language: "sh",
			Code:     "ls /tmp",
			Timeout:  5 * time.Second,
		})
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if result.Status != "blocked" {
			t.Fatalf("Status = %q, want blocked (ls blocked by sandbox guard); error=%q", result.Status, result.Error)
		}
		if result.FilesystemAccess {
			t.Errorf("FilesystemAccess = true, want false (ls blocked)")
		}
	})
}

func TestStrictMode_EnvelopeShapeMatchesReference(t *testing.T) {
	ref := StrictModeBlockedEnvelopeShape()
	if ref.FilesystemAccess {
		t.Error("reference FilesystemAccess defaults to false")
	}
	if ref.NetworkAccess {
		t.Error("reference NetworkAccess defaults to false")
	}

	refJSON, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("marshal reference: %v", err)
	}
	var refMap map[string]interface{}
	json.Unmarshal(refJSON, &refMap)

	s := NewStrictModeSandbox()
	result, _ := s.Execute(context.Background(), CodeExecutionRequest{
		Language: "sh",
		Code:     "curl blocked",
		Timeout:  5 * time.Second,
	})
	resultJSON, _ := json.Marshal(result)
	var resultMap map[string]interface{}
	json.Unmarshal(resultJSON, &resultMap)

	for _, key := range []string{"status", "error", "filesystem_access", "network_access"} {
		if _, ok := resultMap[key]; !ok {
			t.Errorf("CodeExecutionResult JSON missing field %q (reference has it)", key)
		}
		if _, ok := refMap[key]; !ok {
			fmt.Println("reference JSON:", string(refJSON))
		}
	}
}
