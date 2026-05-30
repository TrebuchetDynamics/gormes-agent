package gateway

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/stalecode"
)

const DefaultStaleCodeCacheFreshness = stalecode.DefaultStaleCodeCacheFreshness

// RuntimeStaleCodeStatus classifies whether the live gateway process was
// booted from the same git revision as the managed source checkout.
type RuntimeStaleCodeStatus = stalecode.RuntimeStaleCodeStatus

const (
	RuntimeStaleCodeFresh          = stalecode.RuntimeStaleCodeFresh
	RuntimeStaleCodeStale          = stalecode.RuntimeStaleCodeStale
	RuntimeStaleCodeGitUnavailable = stalecode.RuntimeStaleCodeGitUnavailable
)

// RuntimeStaleCodeEvidence is dynamic read-only status evidence. It is
// generated during status reads and is not used for PID identity.
type RuntimeStaleCodeEvidence = stalecode.RuntimeStaleCodeEvidence

type StaleCodeChecker = stalecode.StaleCodeChecker

func NewStaleCodeChecker(sourceRoot string) *StaleCodeChecker {
	return stalecode.NewStaleCodeChecker(sourceRoot)
}

func RuntimeBootGitSHA() string {
	return stalecode.RuntimeBootGitSHA()
}

func DefaultStaleCodeSourceRoot() string {
	return stalecode.DefaultStaleCodeSourceRoot()
}

var _ time.Duration = DefaultStaleCodeCacheFreshness
