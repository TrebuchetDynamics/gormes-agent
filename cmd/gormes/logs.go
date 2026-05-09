package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// logsHTTPClient bounds the gateway-logs fetch so a hung gateway can't
// hang the operator's terminal indefinitely. http.DefaultClient has no
// timeout — an accept-but-don't-respond gateway would block forever.
// 5s is well above any healthy gateway's response time and well below
// what an operator will tolerate before Ctrl-C.
var logsHTTPClient = &http.Client{Timeout: 5 * time.Second}

// logsEndpointURL is the gateway logs endpoint. Test seam: tests point
// this at httptest servers (or dead URLs) to drive the live-gateway
// path or the file-fallback path deterministically.
var logsEndpointURL = "http://127.0.0.1:43827/api/logs"

func newLogsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Show recent Gormes gateway logs",
		RunE:  runLogs,
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: {build, source: 'gateway'|'file', entries|content, path}")
	return cmd
}

// logsReportJSON is the wire shape for `logs --json`. Fleet log
// aggregation pipelines parse this to ingest entries directly.
// `source: "gateway"` means a live gateway responded; `source: "file"`
// means we fell back to the on-disk log file (raw content). `entries`
// is populated only on the gateway path; `content`/`path` only on file.
type logsReportJSON struct {
	Build   buildProvenanceJSON `json:"build"`
	Source  string              `json:"source"`
	Entries []logsEntryJSON     `json:"entries,omitempty"`
	Path    string              `json:"path,omitempty"`
	Content string              `json:"content,omitempty"`
}

type logsEntryJSON struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

func runLogs(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	asJSON, _ := cmd.Flags().GetBool("json")
	resp, err := logsHTTPClient.Get(logsEndpointURL)
	if err != nil {
		path := config.LogPath()
		data, err := os.ReadFile(path)
		if err != nil {
			msg := fmt.Sprintf("no gateway running and no log file found: %v", err)
			if asJSON {
				return emitJSONInputError(cmd, "no_logs", msg)
			}
			return fmt.Errorf("%s", msg)
		}
		if asJSON {
			body, marshalErr := json.MarshalIndent(logsReportJSON{
				Build:   newBuildProvenance(),
				Source:  "file",
				Path:    path,
				Content: string(data),
			}, "", "  ")
			if marshalErr != nil {
				return marshalErr
			}
			fmt.Fprintln(out, string(body))
			return nil
		}
		fmt.Fprint(out, string(data))
		return nil
	}
	defer resp.Body.Close()

	var result struct {
		Entries []logsEntryJSON `json:"entries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to read logs: %w", err)
	}
	if asJSON {
		body, marshalErr := json.MarshalIndent(logsReportJSON{
			Build:   newBuildProvenance(),
			Source:  "gateway",
			Entries: result.Entries,
		}, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Fprintln(out, string(body))
		return nil
	}
	if len(result.Entries) == 0 {
		fmt.Fprintln(out, "No log entries.")
		return nil
	}
	for _, e := range result.Entries {
		fmt.Fprintf(out, "[%s] %s: %s\n", e.Time, e.Level, e.Message)
	}
	return nil
}
