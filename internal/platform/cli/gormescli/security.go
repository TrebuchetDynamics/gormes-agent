package gormescli

import (
	"context"
	"io"

	appsecurity "github.com/TrebuchetDynamics/gormes-agent/internal/app/security"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type SecurityAuditOptions = appsecurity.AuditOptions
type SecurityBuildProvenance = appsecurity.BuildProvenance

func RunSecurityAudit(ctx context.Context, out io.Writer, opts SecurityAuditOptions, build SecurityBuildProvenance) (toolspkg.SecurityAuditResult, error) {
	return appsecurity.RunAudit(ctx, out, opts, build)
}
