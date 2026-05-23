package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
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

type logsContent struct {
	Source  string
	Entries []logsEntryJSON
	Path    string
	Content string
}

func runLogs(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	asJSON, _ := cmd.Flags().GetBool("json")
	content, err := readLogsContent()
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
			Source:  content.Source,
			Entries: content.Entries,
			Path:    content.Path,
			Content: content.Content,
		}, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Fprintln(out, string(body))
		return nil
	}
	if content.Source == "file" {
		fmt.Fprint(out, content.Content)
		return nil
	}
	if len(content.Entries) == 0 {
		fmt.Fprintln(out, "No log entries.")
		return nil
	}
	for _, line := range formatLogsEntries(content.Entries) {
		fmt.Fprintln(out, line)
	}
	return nil
}

func readLogsContent() (logsContent, error) {
	resp, err := logsHTTPClient.Get(logsEndpointURL)
	if err != nil {
		path := config.LogPath()
		data, fileErr := os.ReadFile(path)
		if fileErr != nil {
			return logsContent{}, fileErr
		}
		return logsContent{Source: "file", Path: path, Content: string(data)}, nil
	}
	defer resp.Body.Close()

	var result struct {
		Entries []logsEntryJSON `json:"entries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return logsContent{}, fmt.Errorf("failed to read logs: %w", err)
	}
	return logsContent{Source: "gateway", Entries: result.Entries}, nil
}

func readLogsTail(limit int) (string, error) {
	content, err := readLogsContent()
	if err != nil {
		return "", err
	}
	var lines []string
	if content.Source == "gateway" {
		lines = formatLogsEntries(content.Entries)
	} else {
		lines = splitLogLines(content.Content)
	}
	lines = tailLogLines(lines, limit)
	return strings.Join(lines, "\n"), nil
}

func formatLogsEntries(entries []logsEntryJSON) []string {
	lines := make([]string, 0, len(entries))
	for _, e := range entries {
		lines = append(lines, fmt.Sprintf("[%s] %s: %s", e.Time, e.Level, e.Message))
	}
	return lines
}

func splitLogLines(content string) []string {
	content = strings.TrimRight(content, "\n")
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}

func tailLogLines(lines []string, limit int) []string {
	if limit <= 0 {
		limit = 20
	}
	if len(lines) <= limit {
		return lines
	}
	return lines[len(lines)-limit:]
}
