package kernel

import (
	"context"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel/retry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

type RetryBudget = retry.RetryBudget
type RetryStatus = retry.RetryStatus
type RetryDelayDecision = retry.RetryDelayDecision

const (
	maxRetryAttempts = retry.MaxRetryAttempts

	RetryDecisionScheduled     = retry.RetryDecisionScheduled
	RetryDecisionProviderHint  = retry.RetryDecisionProviderHint
	RetryDecisionBudgetExhaust = retry.RetryDecisionBudgetExhaust
)

func NewRetryBudget() *RetryBudget { return retry.NewRetryBudget() }

func NewRetryStatus() RetryStatus { return retry.NewRetryStatus() }

func RetrySchedule() []time.Duration { return retry.RetrySchedule() }

func Wait(ctx context.Context, d time.Duration) error { return retry.Wait(ctx, d) }

func retryStatusWithDecision(status RetryStatus, decision RetryDelayDecision, classification llm.ProviderErrorClassification) RetryStatus {
	return retry.StatusWithDecision(status, decision, classification)
}
