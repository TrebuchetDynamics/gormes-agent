package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	sessionpkg "github.com/TrebuchetDynamics/gormes-agent/internal/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/transcript"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

// newSessionCommand returns a fresh session command tree (parent +
// list/export/delete/prune/browse subcommands). Constructor pattern
// avoids cross-test FlagSet contamination on shared package-level vars.
func newSessionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "session",
		Aliases: []string{"sessions"},
		Short:   "Inspect and export persisted sessions",
		Args:    cobra.NoArgs,
	}
	cmd.AddCommand(
		newSessionListCommand(),
		newSessionExportCommand(),
		newSessionDeleteCommand(),
		newSessionPruneCommand(),
		newSessionBrowseCommand(),
	)
	return cmd
}

func newSessionListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			source, _ := cmd.Flags().GetString("source")
			limit, _ := cmd.Flags().GetInt("limit")
			asJSON, _ := cmd.Flags().GetBool("json")
			// On a fresh install the goncho memory.db doesn't exist
			// yet (it's created lazily on the first turn write). For
			// an inventory command the absence of state isn't an
			// error — it's the empty state. Mutating commands
			// (export/delete/continue) keep their hard error in
			// openSessionDirectoryDB.
			db, err := openSessionDirectoryDB()
			if err != nil {
				if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "memory database not found") {
					if asJSON {
						return emitSessionListJSON(cmd, nil)
					}
					fmt.Fprintln(cmd.OutOrStdout(), "No sessions found.")
					return nil
				}
				return err
			}
			defer db.Close()
			sessions, err := sessionpkg.ListDirectorySessions(context.Background(), db, sessionpkg.DirectoryFilter{Source: source, Limit: limit})
			if err != nil {
				return err
			}
			if asJSON {
				return emitSessionListJSON(cmd, sessions)
			}
			if len(sessions) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No sessions found.")
				return nil
			}
			renderSessionDirectoryList(cmd.OutOrStdout(), sessions)
			return nil
		},
	}
	cmd.Flags().String("source", "", "only list sessions from this source")
	cmd.Flags().Int("limit", 20, "max sessions to list")
	cmd.Flags().Bool("json", false, "emit a `{build, sessions: [...]}` JSON document (suitable for fleet inventory automation)")
	return cmd
}

// sessionListReportJSON is the wire shape for `gormes session list --json`.
// Build provenance leads, then the sessions array — same convention as
// the rest of the --json arc. internal/session.DirectoryEntry is
// tag-free; mirroring it here keeps presentation concerns out of the
// session package.
type sessionListReportJSON struct {
	Build    buildProvenanceJSON    `json:"build"`
	Sessions []sessionListEntryJSON `json:"sessions"`
}

type sessionListEntryJSON struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Preview      string `json:"preview"`
	Source       string `json:"source"`
	StartedAt    int64  `json:"started_at"`
	LastActiveAt int64  `json:"last_active_at"`
	MessageCount int    `json:"message_count"`
}

func emitSessionListJSON(cmd *cobra.Command, entries []sessionpkg.DirectoryEntry) error {
	out := make([]sessionListEntryJSON, len(entries))
	for i, e := range entries {
		out[i] = sessionListEntryJSON{
			ID:           e.ID,
			Title:        e.Title,
			Preview:      e.Preview,
			Source:       e.Source,
			StartedAt:    e.StartedAt,
			LastActiveAt: e.LastActiveAt,
			MessageCount: e.MessageCount,
		}
	}
	body, err := json.MarshalIndent(sessionListReportJSON{
		Build:    newBuildProvenance(),
		Sessions: out,
	}, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(body))
	return err
}

// sessionExportReportJSON is the wire shape for `gormes session export
// <id> --json`. Fleet automation aggregating exported session
// transcripts across machines parses this to ingest each export with
// stable shape and binary attribution. Build provenance leads — same
// convention as the rest of the `--json` arc. Content stays as a
// single string field so consumers can re-render the exact markdown
// the human-readable form prints.
type sessionExportReportJSON struct {
	Build     buildProvenanceJSON `json:"build"`
	SessionID string              `json:"session_id"`
	Format    string              `json:"format"`
	Content   string              `json:"content"`
}

func newSessionExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export <session-id-or-prefix>",
		Short: "Export a persisted session transcript",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")
			asJSON, _ := cmd.Flags().GetBool("json")
			if format != "markdown" {
				return fmt.Errorf("unsupported export format %q", format)
			}

			path := config.MemoryDBPath()
			if _, err := os.Stat(path); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("memory database not found at %s", path)
				}
				return err
			}

			db, err := sqlOpenGoncho(path)
			if err != nil {
				return fmt.Errorf("open transcript db: %w", err)
			}
			defer db.Close()

			// Resolve the operator's input as a session id OR a unique
			// prefix — same shape `session delete` accepts. Operators
			// shouldn't have to copy-paste a 24-char id when a 7-char
			// prefix already disambiguates.
			resolved, err := sessionpkg.ResolveSessionIDPrefix(context.Background(), db, args[0])
			if err != nil {
				if errors.Is(err, sessionpkg.ErrSessionNotFound) {
					return fmt.Errorf("session %q not found", args[0])
				}
				if errors.Is(err, sessionpkg.ErrSessionPrefixAmbiguous) {
					return fmt.Errorf("session export: prefix %q is ambiguous: %w", args[0], err)
				}
				return err
			}

			out, err := transcript.ExportMarkdown(context.Background(), db, resolved)
			if err != nil {
				if errors.Is(err, transcript.ErrSessionNotFound) {
					return fmt.Errorf("session %q not found", resolved)
				}
				return err
			}

			if asJSON {
				body, marshalErr := json.MarshalIndent(sessionExportReportJSON{
					Build:     newBuildProvenance(),
					SessionID: resolved,
					Format:    format,
					Content:   out,
				}, "", "  ")
				if marshalErr != nil {
					return marshalErr
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), string(body))
				return err
			}

			_, err = fmt.Fprint(cmd.OutOrStdout(), out)
			return err
		},
	}
	cmd.Flags().String("format", "markdown", "export format")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: {build, session_id, format, content}")
	return cmd
}

func newSessionDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <session-id-or-prefix>",
		Short: "Delete a persisted session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openSessionDirectoryDB()
			if err != nil {
				return err
			}
			defer db.Close()
			asJSON, _ := cmd.Flags().GetBool("json")
			resolved, err := sessionpkg.ResolveSessionIDPrefix(context.Background(), db, args[0])
			if err != nil {
				if errors.Is(err, sessionpkg.ErrSessionNotFound) {
					if asJSON {
						return writeSessionDeleteJSON(cmd.OutOrStdout(), sessionDeleteReportJSON{
							Build:       newBuildProvenance(),
							Action:      "not_found",
							RequestedID: args[0],
						})
					}
					fmt.Fprintf(cmd.OutOrStdout(), "Session '%s' not found.\n", args[0])
					return nil
				}
				if errors.Is(err, sessionpkg.ErrSessionPrefixAmbiguous) {
					if asJSON {
						return writeSessionDeleteJSON(cmd.OutOrStdout(), sessionDeleteReportJSON{
							Build:       newBuildProvenance(),
							Action:      "ambiguous",
							RequestedID: args[0],
							Error:       err.Error(),
						})
					}
					fmt.Fprintf(cmd.OutOrStdout(), "sessions_delete_ambiguous: %s\n", err.Error())
					return nil
				}
				return err
			}
			yes, _ := cmd.Flags().GetBool("yes")
			if !cmd.Flags().Changed("yes") {
				yes = false
			}
			if !yes && !confirmSessionAction(cmd, fmt.Sprintf("Delete session '%s' and all its messages? [y/N] ", resolved)) {
				fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
				return nil
			}
			deleted, err := sessionpkg.DeleteDirectorySession(context.Background(), db, resolved)
			if err != nil {
				return err
			}
			if !deleted {
				if asJSON {
					return writeSessionDeleteJSON(cmd.OutOrStdout(), sessionDeleteReportJSON{
						Build:       newBuildProvenance(),
						Action:      "not_found",
						RequestedID: args[0],
						ResolvedID:  resolved,
					})
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Session '%s' not found.\n", args[0])
				return nil
			}
			if asJSON {
				return writeSessionDeleteJSON(cmd.OutOrStdout(), sessionDeleteReportJSON{
					Build:       newBuildProvenance(),
					Action:      "deleted",
					RequestedID: args[0],
					ResolvedID:  resolved,
					Deleted:     true,
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted session '%s'.\n", resolved)
			return nil
		},
	}
	cmd.Flags().Bool("yes", false, "delete without prompting")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, action: 'deleted'|'not_found'|'ambiguous', requested_id, resolved_id, deleted, error?}`")
	return cmd
}

// sessionDeleteReportJSON is the wire shape for `session delete --json`.
// Operator scripts iterating across hosts parse this to learn whether
// the session actually existed (`deleted: true`), wasn't there
// (`action: "not_found"`), or matched multiple ids (`action: "ambiguous"`).
type sessionDeleteReportJSON struct {
	Build       buildProvenanceJSON `json:"build"`
	Action      string              `json:"action"`
	RequestedID string              `json:"requested_id"`
	ResolvedID  string              `json:"resolved_id,omitempty"`
	Deleted     bool                `json:"deleted"`
	Error       string              `json:"error,omitempty"`
}

func writeSessionDeleteJSON(out interface{ Write(p []byte) (int, error) }, report sessionDeleteReportJSON) error {
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(out, string(body))
	return nil
}

func newSessionPruneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Delete old persisted sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openSessionDirectoryDB()
			if err != nil {
				return err
			}
			defer db.Close()
			days, _ := cmd.Flags().GetInt("older-than")
			source, _ := cmd.Flags().GetString("source")
			yes, _ := cmd.Flags().GetBool("yes")
			if !cmd.Flags().Changed("yes") {
				yes = false
			}
			asJSON, _ := cmd.Flags().GetBool("json")
			if !yes && !confirmSessionAction(cmd, fmt.Sprintf("Delete sessions older than %d days? [y/N] ", days)) {
				fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
				return nil
			}
			cutoff := time.Now().AddDate(0, 0, -days).Unix()
			count, err := sessionpkg.PruneDirectorySessions(context.Background(), db, cutoff, source)
			if err != nil {
				return err
			}
			if asJSON {
				body, marshalErr := json.MarshalIndent(sessionPruneReportJSON{
					Build:         newBuildProvenance(),
					Action:        "pruned",
					OlderThanDays: days,
					Source:        source,
					Pruned:        count,
				}, "", "  ")
				if marshalErr != nil {
					return marshalErr
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(body))
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Pruned %d session(s).\n", count)
			return nil
		},
	}
	cmd.Flags().Int("older-than", 90, "delete sessions older than N days")
	cmd.Flags().String("source", "", "only prune sessions from this source")
	cmd.Flags().Bool("yes", false, "prune without prompting")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, action, older_than_days, source, pruned}`")
	return cmd
}

// sessionPruneReportJSON is the wire shape for `session prune --json`.
// Fleet automation running scheduled session GC parses this to audit
// how many sessions vanished per machine. Source filter is echoed so
// dashboards can correlate with the policy that drove the prune.
type sessionPruneReportJSON struct {
	Build         buildProvenanceJSON `json:"build"`
	Action        string              `json:"action"`
	OlderThanDays int                 `json:"older_than_days"`
	Source        string              `json:"source"`
	Pruned        int                 `json:"pruned"`
}

func newSessionBrowseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Browse and resume persisted sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openSessionDirectoryDB()
			if err != nil {
				return err
			}
			defer db.Close()
			source, _ := cmd.Flags().GetString("source")
			limit, _ := cmd.Flags().GetInt("limit")
			sessions, err := sessionpkg.ListDirectorySessions(context.Background(), db, sessionpkg.DirectoryFilter{Source: source, Limit: limit})
			if err != nil {
				return err
			}
			selected := sessionBrowseFallback(cmd.OutOrStdout(), cmd.InOrStdin(), sessions)
			if selected == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Resuming session: %s\n", selected)
			return nil
		},
	}
	cmd.Flags().String("source", "", "only browse sessions from this source")
	cmd.Flags().Int("limit", 500, "max sessions to load")
	cmd.Flags().Bool("no-curses", false, "use the numbered fallback picker")
	return cmd
}

func openSessionDirectoryDB() (*sql.DB, error) {
	path := config.MemoryDBPath()
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("memory database not found at %s", path)
		}
		return nil, err
	}
	db, err := sqlOpenGoncho(path)
	if err != nil {
		return nil, fmt.Errorf("open session directory db: %w", err)
	}
	return db, nil
}

func resolveContinueSessionFlag(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		value = "last"
	}
	db, err := openSessionDirectoryDB()
	if err != nil {
		if value != "last" {
			return value, nil
		}
		return "", err
	}
	defer db.Close()
	if value == "last" {
		resolved, err := sessionpkg.ResolveMostRecentSession(context.Background(), db, "cli")
		if err != nil {
			return "", err
		}
		if resolved == "" {
			return "", errors.New("no previous session found to continue")
		}
		return resolved, nil
	}
	resolved, err := sessionpkg.ResolveSessionIDPrefix(context.Background(), db, value)
	if err != nil {
		if errors.Is(err, sessionpkg.ErrSessionNotFound) {
			return value, nil
		}
		return "", err
	}
	return resolved, nil
}

func renderSessionDirectoryList(w io.Writer, sessions []sessionpkg.DirectoryEntry) {
	fmt.Fprintf(w, "%-32s %-40s %-13s %s\n", "Title", "Preview", "Last Active", "ID")
	for _, session := range sessions {
		title := truncateSessionColumn(firstNonEmpty(session.Title, "-"), 30)
		preview := truncateSessionColumn(session.Preview, 38)
		fmt.Fprintf(w, "%-32s %-40s %-13s %s\n", title, preview, relativeSessionTime(session.LastActiveAt), session.ID)
	}
}

func sessionBrowseFallback(w io.Writer, r io.Reader, sessions []sessionpkg.DirectoryEntry) string {
	if len(sessions) == 0 {
		fmt.Fprintln(w, "No sessions found.")
		return ""
	}
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  Browse sessions  (enter number to resume, q to cancel)")
	fmt.Fprintln(w, "")
	for i, session := range sessions {
		label := truncateSessionColumn(firstNonEmpty(session.Title, session.Preview, session.ID), 50)
		preview := truncateSessionColumn(session.Preview, 50)
		fmt.Fprintf(w, "  %3d. %-50s  %-50s  %s\n", i+1, label, preview, relativeSessionTime(session.LastActiveAt))
	}
	scanner := bufio.NewScanner(r)
	for {
		fmt.Fprintf(w, "\n  Select [1-%d]: ", len(sessions))
		if !scanner.Scan() {
			fmt.Fprintln(w)
			return ""
		}
		val := strings.TrimSpace(scanner.Text())
		if val == "" || strings.EqualFold(val, "q") || strings.EqualFold(val, "quit") || strings.EqualFold(val, "exit") {
			return ""
		}
		var idx int
		if _, err := fmt.Sscanf(val, "%d", &idx); err != nil || idx < 1 || idx > len(sessions) {
			fmt.Fprintln(w, "  Invalid input. Enter a number or q to cancel.")
			continue
		}
		return sessions[idx-1].ID
	}
}

func confirmSessionAction(cmd *cobra.Command, prompt string) bool {
	fmt.Fprint(cmd.OutOrStdout(), prompt)
	scanner := bufio.NewScanner(cmd.InOrStdin())
	if !scanner.Scan() {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "y" || answer == "yes"
}

func truncateSessionColumn(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}

func relativeSessionTime(ts int64) string {
	if ts <= 0 {
		return "unknown"
	}
	d := time.Since(time.Unix(ts, 0))
	if d < 0 {
		return "now"
	}
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func coalesceSessionNameArgs(argv []string) []string {
	subcommands := map[string]struct{}{
		"chat": {}, "model": {}, "gateway": {}, "setup": {}, "whatsapp": {},
		"telegram": {}, "login": {}, "logout": {}, "auth": {}, "status": {},
		"cron": {}, "doctor": {}, "config": {}, "pairing": {}, "skills": {},
		"tools": {}, "mcp": {}, "session": {}, "sessions": {}, "insights": {},
		"version": {}, "update": {}, "uninstall": {}, "profile": {}, "dashboard": {},
		"goncho": {}, "claw": {}, "plugins": {}, "acp": {}, "webhook": {},
		"memory": {}, "dump": {}, "debug": {}, "backup": {}, "import": {},
		"completion": {}, "logs": {},
	}
	sessionFlags := map[string]struct{}{"-c": {}, "--continue": {}, "-r": {}, "--resume": {}}
	out := make([]string, 0, len(argv))
	for i := 0; i < len(argv); i++ {
		token := argv[i]
		out = append(out, token)
		if _, ok := sessionFlags[token]; !ok {
			continue
		}
		var parts []string
		for i+1 < len(argv) {
			next := argv[i+1]
			if strings.HasPrefix(next, "-") {
				break
			}
			if _, ok := subcommands[next]; ok {
				break
			}
			parts = append(parts, next)
			i++
		}
		if len(parts) > 0 {
			out = append(out, strings.Join(parts, " "))
		}
	}
	return out
}

func newTUISaveExportFunc() tui.SessionExportFunc {
	return func(ctx context.Context, sessionID string) (string, error) {
		dbPath := config.MemoryDBPath()
		if _, err := os.Stat(dbPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("memory database not found at %s", dbPath)
			}
			return "", err
		}

		db, err := sqlOpenGoncho(dbPath)
		if err != nil {
			return "", fmt.Errorf("open transcript db: %w", err)
		}
		defer db.Close()

		out, err := transcript.ExportMarkdown(ctx, db, sessionID)
		if err != nil {
			return "", err
		}

		exportDir := filepath.Join(filepath.Dir(dbPath), "sessions", "exports")
		return writeTUISaveExport(exportDir, tuiSaveExportStem(sessionID), out)
	}
}

func writeTUISaveExport(dir, stem, markdown string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("prepare session export dir: %w", err)
	}

	for i := 0; i < 1000; i++ {
		name := stem + ".md"
		if i > 0 {
			name = fmt.Sprintf("%s-%d.md", stem, i)
		}
		path := filepath.Join(dir, name)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return path, fmt.Errorf("create session export: %w", err)
		}

		_, writeErr := file.WriteString(markdown)
		closeErr := file.Close()
		if writeErr != nil {
			return path, fmt.Errorf("write session export: %w", writeErr)
		}
		if closeErr != nil {
			return path, fmt.Errorf("close session export: %w", closeErr)
		}
		return path, nil
	}

	return "", fmt.Errorf("session export path collision after 1000 attempts")
}

func tuiSaveExportStem(sessionID string) string {
	stem := strings.TrimSpace(sessionID)
	stem = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', 0:
			return '_'
		default:
			return r
		}
	}, stem)
	if stem == "" {
		return "session"
	}
	return stem
}
