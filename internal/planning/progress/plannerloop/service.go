package plannerloop

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/plannerloop/serviceunit"
)

const PlannerInterval = serviceunit.PlannerInterval

type PlannerServiceUnitOptions = serviceunit.PlannerServiceUnitOptions
type PlannerTimerUnitOptions = serviceunit.PlannerTimerUnitOptions
type PlannerPathUnitOptions = serviceunit.PlannerPathUnitOptions
type PlannerImplPathUnitOptions = serviceunit.PlannerImplPathUnitOptions
type PlannerServiceInstallOptions = serviceunit.PlannerServiceInstallOptions

func RenderPlannerServiceUnit(opts PlannerServiceUnitOptions) string {
	return serviceunit.RenderPlannerServiceUnit(opts)
}

func RenderPlannerTimerUnit(opts PlannerTimerUnitOptions) string {
	return serviceunit.RenderPlannerTimerUnit(opts)
}

func RenderPlannerPathUnit(opts PlannerPathUnitOptions) string {
	return serviceunit.RenderPlannerPathUnit(opts)
}

func RenderPlannerImplPathUnit(opts PlannerImplPathUnitOptions) string {
	return serviceunit.RenderPlannerImplPathUnit(opts)
}

func InstallPlannerService(ctx context.Context, opts PlannerServiceInstallOptions) error {
	return serviceunit.InstallPlannerService(ctx, opts)
}
