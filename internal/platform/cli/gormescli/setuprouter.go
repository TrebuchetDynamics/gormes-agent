package gormescli

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/app/setuprouter"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func RunSetupRouterSection(cmd *cobra.Command) error {
	return setuprouter.Run(cmd.OutOrStdout(), setuprouter.Options{})
}

func SetupRouterFallbackRules(routes []config.RouterRouteCfg) []config.RouterFallbackCfg {
	return setuprouter.FallbackRules(routes)
}

func SetupRouterRouteLabels(route config.RouterRouteCfg) []string {
	return setuprouter.RouteLabels(route)
}

func SetupRouterOpenAIBaseURL(listen, defaultListen string) string {
	return setuprouter.OpenAIBaseURL(listen, defaultListen)
}

func SetupRouterSlug(value string) string {
	return setuprouter.Slug(value)
}
