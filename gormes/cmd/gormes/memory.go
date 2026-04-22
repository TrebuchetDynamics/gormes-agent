package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	_ "github.com/ncruces/go-sqlite3/driver"
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/memory"
)

var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Inspect persisted memory and extractor state",
}

func init() {
	memoryCmd.AddCommand(memoryStatusCmd)
}

var memoryStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show extractor queue depth and dead letters",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		path := config.MemoryDBPath()
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("memory database not found at %s", path)
			}
			return err
		}

		db, err := sql.Open("sqlite3", path)
		if err != nil {
			return fmt.Errorf("open memory db: %w", err)
		}
		defer db.Close()

		status, err := memory.ReadExtractorStatus(context.Background(), db, 0)
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(cmd.OutOrStdout(), formatExtractorStatus(status))
		return err
	},
}

func formatExtractorStatus(status memory.ExtractorStatus) string {
	var b strings.Builder
	b.WriteString("Extractor status\n")
	b.WriteString(fmt.Sprintf("worker_health: %s\n", status.WorkerHealth))
	b.WriteString(fmt.Sprintf("queue_depth: %d\n", status.QueueDepth))
	b.WriteString(fmt.Sprintf("dead_letters: %d\n", status.DeadLetterCount))
	b.WriteString(formatDeadLetterSummary(status.ErrorSummary))
	if len(status.RecentDeadLetters) == 0 {
		b.WriteString("recent_dead_letters: none\n")
		return b.String()
	}
	b.WriteString("recent_dead_letters:\n")
	for _, dl := range status.RecentDeadLetters {
		b.WriteString(fmt.Sprintf("- turn_id=%d session_id=%s chat_id=%s attempts=%d error=%q\n",
			dl.ID, dl.SessionID, dl.ChatID, dl.Attempts, dl.Error))
	}
	return b.String()
}

func formatDeadLetterSummary(items []memory.DeadLetterErrorSummary) string {
	if len(items) == 0 {
		return "dead_letter_summary: none\n"
	}

	var b strings.Builder
	b.WriteString("dead_letter_summary:\n")
	for _, item := range items {
		b.WriteString(fmt.Sprintf("- error=%q count=%d\n", item.Error, item.Count))
	}
	return b.String()
}
