package skills

import (
	"context"
	"time"
)

type ValidationResult struct {
	SkillName string
	Valid     bool
	Error     string
	Duration  time.Duration
}

type SkillValidator struct {
	executor *SkillCodeExecutor
}

func NewSkillValidator(executor *SkillCodeExecutor) *SkillValidator {
	return &SkillValidator{executor: executor}
}

func (v *SkillValidator) Validate(ctx context.Context, skillName, code, language string) ValidationResult {
	start := time.Now()

	if code == "" {
		return ValidationResult{
			SkillName: skillName,
			Valid:     false,
			Error:     "no code to validate",
			Duration:  time.Since(start),
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
		return ValidationResult{
			SkillName: skillName,
			Valid:     false,
			Error:     msg,
			Duration:  time.Since(start),
		}
	}

	return ValidationResult{
		SkillName: skillName,
		Valid:     true,
		Duration:  time.Since(start),
	}
}

func (v *SkillValidator) BatchValidate(ctx context.Context, skills []struct{ Name, Code, Language string }) []ValidationResult {
	results := make([]ValidationResult, len(skills))
	for i, s := range skills {
		results[i] = v.Validate(ctx, s.Name, s.Code, s.Language)
	}
	return results
}
