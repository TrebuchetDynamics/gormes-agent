package gormescli

import (
	"encoding/json"
	"fmt"
	"io"

	appsetup "github.com/TrebuchetDynamics/gormes-agent/internal/app/setup"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type SetupResetReportJSON struct {
	Build          VersionBuildProvenance `json:"build"`
	Action         string                 `json:"action"`
	ConfigPath     string                 `json:"config_path"`
	BreadcrumbPath string                 `json:"breadcrumb_path"`
}

func ResetSetupDefaultConfig() (string, error) {
	return appsetup.ResetDefaultConfig(config.ConfigPath())
}

func EmitSetupResetResult(out io.Writer, build VersionBuildProvenance, configPath, breadcrumb string, asJSON bool) (bool, error) {
	if out == nil {
		out = io.Discard
	}
	if asJSON {
		body, err := json.MarshalIndent(SetupResetReportJSON{
			Build:          build,
			Action:         "reset",
			ConfigPath:     configPath,
			BreadcrumbPath: breadcrumb,
		}, "", "  ")
		if err != nil {
			return false, err
		}
		fmt.Fprintln(out, string(body))
		return true, nil
	}
	fmt.Fprintln(out, "Configuration reset to defaults.")
	if breadcrumb != "" {
		fmt.Fprintf(out, "Prior config preserved at %s — restore with `cp %s %s`\n", breadcrumb, breadcrumb, configPath)
	}
	return false, nil
}
