package skills

import (
	"context"
	"fmt"
	"time"
)

type SkillCodeRequest struct {
	SkillName   string
	Code        string
	Language    string
	InputParams map[string]string
	Timeout     time.Duration
}

type SkillCodeResult struct {
	Success  bool
	Output   string
	Error    string
	Duration time.Duration
}

type SkillCodeExecutionRequest struct {
	Language string
	Code     string
	Timeout  time.Duration
}

type SkillCodeExecutionResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

type SkillCodeSandbox interface {
	Execute(ctx context.Context, req SkillCodeExecutionRequest) (SkillCodeExecutionResult, error)
}

type SkillCodeExecutor struct {
	sandbox SkillCodeSandbox
}

func NewSkillCodeExecutor(sandbox SkillCodeSandbox) *SkillCodeExecutor {
	return &SkillCodeExecutor{sandbox: sandbox}
}

func (e *SkillCodeExecutor) Execute(ctx context.Context, req SkillCodeRequest) (SkillCodeResult, error) {
	if req.Code == "" {
		return SkillCodeResult{Success: false, Error: "no code to execute"}, nil
	}
	if req.Timeout <= 0 {
		req.Timeout = 30 * time.Second
	}

	start := time.Now()
	result, err := e.sandbox.Execute(ctx, SkillCodeExecutionRequest{
		Language: req.Language,
		Code:     req.Code,
		Timeout:  req.Timeout,
	})
	duration := time.Since(start)

	if err != nil {
		return SkillCodeResult{Success: false, Error: err.Error(), Duration: duration}, nil
	}

	if result.ExitCode != 0 {
		return SkillCodeResult{
			Success:  false,
			Output:   result.Stdout,
			Error:    fmt.Sprintf("exit code %d: %s", result.ExitCode, result.Stderr),
			Duration: duration,
		}, nil
	}

	return SkillCodeResult{
		Success:  true,
		Output:   result.Stdout,
		Duration: duration,
	}, nil
}
