package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/goncho/service"
)

func newGonchoContinueCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "continue [session-key]",
		Short: "List recent sessions or resume a specific session",
		Long: `Without arguments, lists recent sessions with their summaries.
With a session-key argument, loads that session's full context for resumption.`,
		RunE: runGonchoContinue,
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON")
	cmd.Flags().Int("limit", 10, "max sessions to list")
	return cmd
}

func runGonchoContinue(cmd *cobra.Command, args []string) error {
	emitJSON, _ := cmd.Flags().GetBool("json")
	limit, _ := cmd.Flags().GetInt("limit")

	db, svc, err := openGonchoService()
	if err != nil {
		return err
	}
	defer db.Close()

	if len(args) > 0 {
		return runGonchoContinueSession(cmd, svc, args[0], emitJSON)
	}

	return runGonchoContinueList(cmd, db, svc, limit, emitJSON)
}

func runGonchoContinueList(cmd *cobra.Command, db interface{}, _ interface{}, limit int, emitJSON bool) error {
	ctx := context.Background()

	// Query recent sessions with summaries
	rows, err := db.(*sql.DB).QueryContext(ctx, `
		SELECT DISTINCT session_key, summary_type, content, created_at
		FROM goncho_session_summaries
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	type sessionEntry struct {
		SessionKey  string `json:"session_key"`
		SummaryType string `json:"summary_type"`
		Content     string `json:"content"`
		CreatedAt   int64  `json:"created_at"`
	}

	var sessions []sessionEntry
	for rows.Next() {
		var s sessionEntry
		if err := rows.Scan(&s.SessionKey, &s.SummaryType, &s.Content, &s.CreatedAt); err != nil {
			return fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, s)
	}

	if emitJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		return enc.Encode(sessions)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Recent sessions (use 'gormes goncho continue <session-key>' to resume):\n\n")
	for _, s := range sessions {
		fmt.Fprintf(cmd.OutOrStdout(), "session: %s\n", s.SessionKey)
		fmt.Fprintf(cmd.OutOrStdout(), "  type: %s\n", s.SummaryType)
		fmt.Fprintf(cmd.OutOrStdout(), "  summary: %s\n\n", truncateString(s.Content, 120))
	}
	return nil
}

func runGonchoContinueSession(cmd *cobra.Command, svc interface{}, sessionKey string, emitJSON bool) error {
	ctx := context.Background()
	s := svc.(*goncho.Service)

	result, err := s.Context(ctx, goncho.ContextParams{
		Peer:       "operator",
		SessionKey: sessionKey,
		MaxTokens:  4000,
	})
	if err != nil {
		return fmt.Errorf("load session context: %w", err)
	}

	if emitJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		return enc.Encode(result)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Session context: %s\n\n", sessionKey)
	fmt.Fprintf(cmd.OutOrStdout(), "peer_card: %s\n", strings.Join(result.PeerCard, ", "))
	fmt.Fprintf(cmd.OutOrStdout(), "representation: %s\n", result.Representation)
	if len(result.Conclusions) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "conclusions:\n")
		for _, c := range result.Conclusions {
			fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", c)
		}
	}
	if len(result.RecentMessages) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\nrecent_messages:\n")
		for _, m := range result.RecentMessages {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s: %s\n", m.Role, truncateString(m.Content, 100))
		}
	}
	return nil
}
