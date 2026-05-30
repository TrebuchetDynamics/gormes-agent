package llm

import (
	"net/http"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/routing"
)

type RateLimitClass = routing.RateLimitClass

const (
	RateLimitGenuineQuota         RateLimitClass = routing.RateLimitGenuineQuota
	RateLimitUpstreamCapacity     RateLimitClass = routing.RateLimitUpstreamCapacity
	RateLimitInsufficientEvidence RateLimitClass = routing.RateLimitInsufficientEvidence
)

const (
	StatusRateGuardUnavailable = routing.StatusRateGuardUnavailable
	StatusNousRateLimited      = routing.StatusNousRateLimited
	StatusNousUpstreamCapacity = routing.StatusNousUpstreamCapacity
	StatusBudgetHeaderMissing  = routing.StatusBudgetHeaderMissing
)

type BudgetBucket = routing.BudgetBucket
type BudgetSnapshot = routing.BudgetSnapshot
type RateGuardDecision = routing.RateGuardDecision

func Classify429(headers http.Header) RateLimitClass {
	return routing.Classify429(headers)
}

func ParseBudget(headers http.Header) BudgetSnapshot {
	return routing.ParseBudget(headers)
}

func DecideRateGuard(headers http.Header, last GuardState) RateGuardDecision {
	return routing.DecideRateGuard(headers, last)
}
