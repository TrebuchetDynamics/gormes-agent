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

func newLogsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logs",
		Short: "Show recent Gormes gateway logs",
		RunE:  runLogs,
	}
}

func runLogs(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	resp, err := logsHTTPClient.Get("http://127.0.0.1:43827/api/logs")
	if err != nil {
		data, err := os.ReadFile(config.LogPath())
		if err != nil {
			return fmt.Errorf("no gateway running and no log file found: %w", err)
		}
		fmt.Fprint(out, string(data))
		return nil
	}
	defer resp.Body.Close()

	var result struct {
		Entries []struct {
			Time    string `json:"time"`
			Level   string `json:"level"`
			Message string `json:"message"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to read logs: %w", err)
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
