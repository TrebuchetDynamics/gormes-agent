package gateway

import (
	"log/slog"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	runtimegateway "github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/gateway/startupsecurity"
)

// StartupSecurityReport is the sanitized startup admission result used before
// opening live gateway channels. Config may have weak placeholder credentials
// blanked so the foreground gateway cannot accidentally enable those channels.
type StartupSecurityReport = startupsecurity.Report

// EvaluateStartupSecurity preserves the gateway startup guards for missing
// allowlists and placeholder credentials. It returns a copy of cfg with weak
// credential platforms disabled plus redacted admission evidence for logs.
func EvaluateStartupSecurity(cfg config.Config, lookupEnv func(string) string) StartupSecurityReport {
	return startupsecurity.Evaluate(cfg, lookupEnv)
}

// StartupAllowlistConfigured reports whether startup has a scoped gateway
// allowlist configured through config or supported environment variables.
func StartupAllowlistConfigured(cfg config.Config, lookupEnv func(string) string) bool {
	return startupsecurity.AllowlistConfigured(cfg, lookupEnv)
}

// StartupAllowAllConfigured reports whether startup explicitly opted into
// allowing all gateway users for a supported channel.
func StartupAllowAllConfigured(lookupEnv func(string) string) bool {
	return startupsecurity.AllowAllConfigured(lookupEnv)
}

// LogStartupSecurityEvidence writes redacted startup admission findings to the
// gateway logger while skipping empty evidence entries.
func LogStartupSecurityEvidence(evidence []runtimegateway.AdmissionEvidence, log *slog.Logger) {
	startupsecurity.LogEvidence(evidence, log)
}
