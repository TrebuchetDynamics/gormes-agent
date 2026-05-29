package skills

import (
	"context"
	"fmt"
	"time"
)

// ValidationResult captures the outcome of a skill validation run.
type ValidationResult struct {
	SkillName string
	Valid     bool
	Error     string
	Duration  time.Duration
	// ForceLoaded is true when the skill was loaded despite validation failure
	// because the operator explicitly requested force-load.
	ForceLoaded bool
}

// SkillValidator runs lightweight sandbox execution to verify skills at load time.
// Broken skills are rejected unless the operator sets ForceLoad, which marks
// the result as ForceLoaded=true while preserving the original error.
type SkillValidator struct {
	executor  *SkillCodeExecutor
	ForceLoad bool
}

// NewSkillValidator returns a SkillValidator that uses the given sandbox-backed executor.
func NewSkillValidator(executor *SkillCodeExecutor) *SkillValidator {
	return &SkillValidator{executor: executor}
}

// Validate runs the given skill code in the sandbox and returns a result.
// An empty code body is treated as invalid. Execution errors and non-zero
// exit codes produce a Valid=false result with the error message preserved.
//
// When ForceLoad is true, a failed validation still returns Valid=false
// but sets ForceLoaded=true to signal that the operator chose to load the
// skill anyway.
func (v *SkillValidator) Validate(ctx context.Context, skillName, code, language string) ValidationResult {
	start := time.Now()

	if code == "" {
		return ValidationResult{
			SkillName:   skillName,
			Valid:       false,
			Error:       "no code to validate",
			Duration:    time.Since(start),
			ForceLoaded: v.ForceLoad,
		}
	}

	result, err := v.executor.Execute(ctx, SkillCodeRequest{
		SkillName: skillName,
		Code:      code,
		Language:  language,
		Timeout:   5 * time.Second,
	})

	if err != nil || !result.Success {
		msg := "execution failed"
		if result.Error != "" {
			msg = result.Error
		}
		if err != nil {
			msg = err.Error()
		}
		vr := ValidationResult{
			SkillName:   skillName,
			Valid:       false,
			Error:       msg,
			Duration:    time.Since(start),
			ForceLoaded: v.ForceLoad,
		}
		return vr
	}

	return ValidationResult{
		SkillName: skillName,
		Valid:     true,
		Duration:  time.Since(start),
	}
}

// ValidateAsync runs Validate in a background goroutine and returns a
// channel that receives the result. The caller must read the channel;
// the goroutine will block until the result is consumed.
func (v *SkillValidator) ValidateAsync(ctx context.Context, skillName, code, language string) <-chan ValidationResult {
	ch := make(chan ValidationResult, 1)
	go func() {
		ch <- v.Validate(ctx, skillName, code, language)
	}()
	return ch
}

// ValidateCanary runs a minimal canary input through the skill to verify
// basic functionality. A canary execution uses a small, safe input (e.g.
// an empty JSON object or a single-line echo) appropriate for the language.
// This is cheaper than a full execution and catches gross syntax/import
// errors quickly.
func (v *SkillValidator) ValidateCanary(ctx context.Context, skillName, code, language string) ValidationResult {
	canaryInput := canaryInputForLanguage(language)
	if canaryInput == nil {
		return v.Validate(ctx, skillName, code, language)
	}

	start := time.Now()
	result, err := v.executor.Execute(ctx, SkillCodeRequest{
		SkillName:   skillName,
		Code:        code,
		Language:    language,
		InputParams: canaryInput,
		Timeout:     2 * time.Second,
	})

	if err != nil || !result.Success {
		msg := "canary execution failed"
		if result.Error != "" {
			msg = result.Error
		}
		if err != nil {
			msg = err.Error()
		}
		vr := ValidationResult{
			SkillName:   skillName,
			Valid:       false,
			Error:       msg,
			Duration:    time.Since(start),
			ForceLoaded: v.ForceLoad,
		}
		return vr
	}

	return ValidationResult{
		SkillName: skillName,
		Valid:     true,
		Duration:  time.Since(start),
	}
}

// ValidateCanaryAsync runs ValidateCanary in a background goroutine.
func (v *SkillValidator) ValidateCanaryAsync(ctx context.Context, skillName, code, language string) <-chan ValidationResult {
	ch := make(chan ValidationResult, 1)
	go func() {
		ch <- v.ValidateCanary(ctx, skillName, code, language)
	}()
	return ch
}

// BatchValidate runs Validate on each skill sequentially and returns all results.
func (v *SkillValidator) BatchValidate(ctx context.Context, skills []struct{ Name, Code, Language string }) []ValidationResult {
	results := make([]ValidationResult, len(skills))
	for i, s := range skills {
		results[i] = v.Validate(ctx, s.Name, s.Code, s.Language)
	}
	return results
}

// canaryInputForLanguage returns a minimal safe input map for canary execution.
// Returns nil when no language-specific canary is defined; callers should
// fall back to a full Validate in that case.
func canaryInputForLanguage(language string) map[string]string {
	switch language {
	case "python":
		return map[string]string{"input": "{}"}
	case "bash":
		return map[string]string{"input": "canary"}
	case "javascript", "js":
		return map[string]string{"input": "{}"}
	default:
		return nil
	}
}

// ForceLoadMsg formats a human-readable message explaining that a skill was
// force-loaded despite validation failure.
func ForceLoadMsg(skillName string, validationErr string) string {
	return fmt.Sprintf("skill %q force-loaded despite validation failure: %s", skillName, validationErr)
}
