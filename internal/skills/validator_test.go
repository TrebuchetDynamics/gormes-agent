package skills

import (
	"context"
	"testing"
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
