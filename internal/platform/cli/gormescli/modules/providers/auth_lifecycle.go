package providers

import (
	"encoding/json"
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

// AuthLifecycleReportJSON is the shared lifecycle wire shape for provider auth
// add/remove/reset/logout reports.
type AuthLifecycleReportJSON struct {
	Build    gormescli.BuildProvenance `json:"build"`
	Action   string                    `json:"action"`
	Provider string                    `json:"provider"`
	Count    int                       `json:"count,omitempty"`
	Removed  *AuthRemovedJSON          `json:"removed,omitempty"`
	Redacted bool                      `json:"redacted"`
}

type AuthRemovedJSON struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func WriteAuthLifecycleJSON(out interface{ Write(p []byte) (int, error) }, report AuthLifecycleReportJSON) error {
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(out, string(body))
	return nil
}
