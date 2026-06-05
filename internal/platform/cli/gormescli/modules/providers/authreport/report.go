package authreport

import (
	"encoding/json"
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

// LifecycleReportJSON is the shared lifecycle wire shape for provider auth
// add/remove/reset/logout reports.
type LifecycleReportJSON struct {
	Build    gormescli.BuildProvenance `json:"build"`
	Action   string                    `json:"action"`
	Provider string                    `json:"provider"`
	Count    int                       `json:"count,omitempty"`
	Removed  *RemovedJSON              `json:"removed,omitempty"`
	Redacted bool                      `json:"redacted"`
}

type RemovedJSON struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func WriteLifecycleJSON(out interface{ Write(p []byte) (int, error) }, report LifecycleReportJSON) error {
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(out, string(body))
	return nil
}
