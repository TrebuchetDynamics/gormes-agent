package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

func newLogsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logs",
		Short: "Show recent Gormes gateway logs",
		RunE:  runLogs,
	}
}

func runLogs(_ *cobra.Command, _ []string) error {
	resp, err := http.Get("http://127.0.0.1:43827/api/logs")
	if err != nil {
		data, err := os.ReadFile(logFilePath())
		if err != nil {
			return fmt.Errorf("no gateway running and no log file found: %w", err)
		}
		fmt.Print(string(data))
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
		fmt.Println("No log entries.")
		return nil
	}
	for _, e := range result.Entries {
		fmt.Printf("[%s] %s: %s\n", e.Time, e.Level, e.Message)
	}
	return nil
}

func logFilePath() string {
	home := os.Getenv("GORMES_HOME")
	if home == "" {
		home = os.Getenv("HOME") + "/.gormes"
	}
	return home + "/gormes.log"
}
