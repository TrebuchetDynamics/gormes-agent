package skills

import (
	"context"
	"testing"
	"time"
)

func TestSkillValidator_Valid(t *testing.T) {
	sandbox := &mockSandbox{
		result: SkillCodeExecutionResult{ExitCode: 0, Stdout: "ok"},
	}
	executor := NewSkillCodeExecutor(sandbox)
	validator := NewSkillValidator(executor)

	result := validator.Validate(context.Background(), "test", "echo ok", "bash")
	if !result.Valid {
		t.Fatalf("valid skill failed: %s", result.Error)
	}
}

func TestSkillValidator_InvalidCode(t *testing.T) {
	executor := NewSkillCodeExecutor(&mockSandbox{})
	validator := NewSkillValidator(executor)

	result := validator.Validate(context.Background(), "test", "", "bash")
	if result.Valid {
		t.Fatal("empty code should be invalid")
	}
}

func TestSkillValidator_ExecutionError(t *testing.T) {
	sandbox := &mockSandbox{
		result: SkillCodeExecutionResult{ExitCode: 1, Stderr: "syntax error"},
	}
	executor := NewSkillCodeExecutor(sandbox)
	validator := NewSkillValidator(executor)

	result := validator.Validate(context.Background(), "bad", "invalid", "python")
	if result.Valid {
		t.Fatal("error exit should be invalid")
	}
}

func TestSkillValidator_BatchValidate(t *testing.T) {
	sandbox := &mockSandbox{
		result: SkillCodeExecutionResult{ExitCode: 0, Stdout: "ok"},
	}
	executor := NewSkillCodeExecutor(sandbox)
	validator := NewSkillValidator(executor)

	results := validator.BatchValidate(context.Background(), []struct{ Name, Code, Language string }{
		{Name: "a", Code: "echo a", Language: "bash"},
		{Name: "b", Code: "echo b", Language: "bash"},
	})
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if !results[0].Valid || !results[1].Valid {
		t.Fatal("both should be valid")
	}
}

func TestSkillValidator_ForceLoad(t *testing.T) {
	sandbox := &mockSandbox{
		result: SkillCodeExecutionResult{ExitCode: 1, Stderr: "syntax error"},
	}
	executor := NewSkillCodeExecutor(sandbox)
	validator := NewSkillValidator(executor)
	validator.ForceLoad = true

	result := validator.Validate(context.Background(), "broken", "bad code", "python")
	if result.Valid {
		t.Fatal("broken skill should not be Valid even with ForceLoad")
	}
	if !result.ForceLoaded {
		t.Fatal("ForceLoaded should be true when ForceLoad is set")
	}
	if result.Error == "" {
		t.Fatal("Error should be preserved for force-loaded broken skills")
	}
}

func TestSkillValidator_ForceLoadOff(t *testing.T) {
	sandbox := &mockSandbox{
		result: SkillCodeExecutionResult{ExitCode: 1, Stderr: "syntax error"},
	}
	executor := NewSkillCodeExecutor(sandbox)
	validator := NewSkillValidator(executor)
	validator.ForceLoad = false

	result := validator.Validate(context.Background(), "broken", "bad code", "python")
	if result.ForceLoaded {
		t.Fatal("ForceLoaded should be false when ForceLoad is not set")
	}
}

func TestSkillValidator_ValidateAsync(t *testing.T) {
	sandbox := &mockSandbox{
		result: SkillCodeExecutionResult{ExitCode: 0, Stdout: "ok"},
	}
	executor := NewSkillCodeExecutor(sandbox)
	validator := NewSkillValidator(executor)

	ch := validator.ValidateAsync(context.Background(), "test", "echo ok", "bash")

	select {
	case result := <-ch:
		if !result.Valid {
			t.Fatalf("async validation failed: %s", result.Error)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for async validation")
	}
}

func TestSkillValidator_ValidateAsyncBroken(t *testing.T) {
	executor := NewSkillCodeExecutor(&mockSandbox{})
	validator := NewSkillValidator(executor)
	validator.ForceLoad = true

	ch := validator.ValidateAsync(context.Background(), "broken", "", "bash")

	select {
	case result := <-ch:
		if result.Valid {
			t.Fatal("empty code should not be valid")
		}
		if !result.ForceLoaded {
			t.Fatal("ForceLoaded should be true with ForceLoad set")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for async validation")
	}
}

func TestSkillValidator_ValidateCanary_Success(t *testing.T) {
	sandbox := &mockSandbox{
		result: SkillCodeExecutionResult{ExitCode: 0, Stdout: "ok"},
	}
	executor := NewSkillCodeExecutor(sandbox)
	validator := NewSkillValidator(executor)

	result := validator.ValidateCanary(context.Background(), "test", "print('hello')", "python")
	if !result.Valid {
		t.Fatalf("canary validation failed: %s", result.Error)
	}
}

func TestSkillValidator_ValidateCanary_Failure(t *testing.T) {
	sandbox := &mockSandbox{
		result: SkillCodeExecutionResult{ExitCode: 1, Stderr: "ImportError"},
	}
	executor := NewSkillCodeExecutor(sandbox)
	validator := NewSkillValidator(executor)

	result := validator.ValidateCanary(context.Background(), "bad-skill", "import nonexistent", "python")
	if result.Valid {
		t.Fatal("canary should fail on import error")
	}
	if result.Error == "" {
		t.Fatal("canary failure should have error message")
	}
}

func TestSkillValidator_ValidateCanary_NoCanaryForLanguage(t *testing.T) {
	sandbox := &mockSandbox{
		result: SkillCodeExecutionResult{ExitCode: 0, Stdout: "ok"},
	}
	executor := NewSkillCodeExecutor(sandbox)
	validator := NewSkillValidator(executor)

	result := validator.ValidateCanary(context.Background(), "test", "puts ok", "ruby")
	if !result.Valid {
		t.Fatalf("language without specific canary should fall back to full validate: %s", result.Error)
	}
}

func TestSkillValidator_ValidateCanaryAsync(t *testing.T) {
	sandbox := &mockSandbox{
		result: SkillCodeExecutionResult{ExitCode: 0, Stdout: "ok"},
	}
	executor := NewSkillCodeExecutor(sandbox)
	validator := NewSkillValidator(executor)

	ch := validator.ValidateCanaryAsync(context.Background(), "test", "echo ok", "bash")

	select {
	case result := <-ch:
		if !result.Valid {
			t.Fatalf("async canary validation failed: %s", result.Error)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for async canary validation")
	}
}

func TestSkillValidator_Duration(t *testing.T) {
	sandbox := &mockSandbox{
		result: SkillCodeExecutionResult{ExitCode: 0, Stdout: "fast"},
	}
	executor := NewSkillCodeExecutor(sandbox)
	validator := NewSkillValidator(executor)

	result := validator.Validate(context.Background(), "fast", "echo ok", "bash")
	if result.Duration <= 0 {
		t.Fatal("duration should be positive")
	}
	if result.Duration > 500*time.Millisecond {
		t.Fatalf("mock validation should complete quickly, got %v", result.Duration)
	}
}

func TestForceLoadMsg(t *testing.T) {
	msg := ForceLoadMsg("my-skill", "syntax error on line 1")
	if msg == "" {
		t.Fatal("ForceLoadMsg should return a non-empty string")
	}
	if !strContains(msg, "my-skill") || !strContains(msg, "force-loaded") || !strContains(msg, "syntax error on line 1") {
		t.Fatalf("ForceLoadMsg missing expected content: %s", msg)
	}
}

func strContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
