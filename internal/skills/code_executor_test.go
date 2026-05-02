package skills

import (
	"context"
	"testing"
	"time"
)

type mockSandbox struct {
	result SkillCodeExecutionResult
	err    error
}

func (m *mockSandbox) Execute(ctx context.Context, req SkillCodeExecutionRequest) (SkillCodeExecutionResult, error) {
	return m.result, m.err
}

func TestSkillCodeExecutor_Success(t *testing.T) {
	sandbox := &mockSandbox{
		result: SkillCodeExecutionResult{
			ExitCode: 0, Stdout: "hello world",
		},
	}
	executor := NewSkillCodeExecutor(sandbox)

	result, err := executor.Execute(context.Background(), SkillCodeRequest{
		SkillName: "test-skill",
		Code:      "print('hello')",
		Language:  "python",
		Timeout:   5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("execution failed: %s", result.Error)
	}
	if result.Output != "hello world" {
		t.Fatalf("output = %q, want 'hello world'", result.Output)
	}
}

func TestSkillCodeExecutor_ErrorExit(t *testing.T) {
	sandbox := &mockSandbox{
		result: SkillCodeExecutionResult{
			ExitCode: 1, Stderr: "syntax error",
		},
	}
	executor := NewSkillCodeExecutor(sandbox)

	result, err := executor.Execute(context.Background(), SkillCodeRequest{
		Code:     "invalid code",
		Language: "python",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Fatal("should fail on non-zero exit code")
	}
}

func TestSkillCodeExecutor_EmptyCode(t *testing.T) {
	executor := NewSkillCodeExecutor(&mockSandbox{})
	result, _ := executor.Execute(context.Background(), SkillCodeRequest{})
	if result.Success {
		t.Fatal("empty code should fail")
	}
}

func TestSkillCodeExecutor_DefaultTimeout(t *testing.T) {
	sandbox := &mockSandbox{
		result: SkillCodeExecutionResult{ExitCode: 0},
	}
	executor := NewSkillCodeExecutor(sandbox)
	result, err := executor.Execute(context.Background(), SkillCodeRequest{
		Code: "echo ok", Language: "bash", Timeout: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatal("should succeed with default timeout")
	}
}
