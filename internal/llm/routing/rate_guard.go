package routing

import (
	"net/http"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/routing/rateguard"
)

type RateLimitClass = rateguard.RateLimitClass

const (
	RateLimitGenuineQuota         RateLimitClass = rateguard.RateLimitGenuineQuota
	RateLimitUpstreamCapacity     RateLimitClass = rateguard.RateLimitUpstreamCapacity
	RateLimitInsufficientEvidence RateLimitClass = rateguard.RateLimitInsufficientEvidence
)

const (
	StatusRateGuardUnavailable = rateguard.StatusRateGuardUnavailable
	StatusNousRateLimited      = rateguard.StatusNousRateLimited
	StatusNousUpstreamCapacity = rateguard.StatusNousUpstreamCapacity
	StatusBudgetHeaderMissing  = rateguard.StatusBudgetHeaderMissing
)

type BudgetBucket = rateguard.BudgetBucket
type BudgetSnapshot = rateguard.BudgetSnapshot
type RateGuardDecision = rateguard.RateGuardDecision
type GuardState = rateguard.GuardState

func Classify429(headers http.Header) RateLimitClass {
	return rateguard.Classify429(headers)
}

func ParseBudget(headers http.Header) BudgetSnapshot {
	return rateguard.ParseBudget(headers)
}

func DecideRateGuard(headers http.Header, last GuardState) RateGuardDecision {
	return rateguard.DecideRateGuard(headers, last)
}

func ApplyClassification(state GuardState, now time.Time, class RateLimitClass) GuardState {
	return rateguard.ApplyClassification(state, now, class)
}
