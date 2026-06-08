package session

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

	gonchoservice "github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	sessionpkg "github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/transcript"
	sessionsmodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/sessions"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

// newSessionCommand returns a fresh session command tree (parent +
// list/export/delete/prune/browse subcommands). Constructor pattern
// avoids cross-test FlagSet contamination on shared package-level vars.
type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

type CommandOptions struct {
	Build              func() BuildProvenance
	UnavailableCommand func(sessionsmodule.UnavailableCommandSpec) *cobra.Command
}

var buildProvenance = func() BuildProvenance { return BuildProvenance{} }

func currentBuildProvenance() BuildProvenance {
	if buildProvenance == nil {
		return BuildProvenance{}
	}
	return buildProvenance()
}

func NewCommand(options CommandOptions) *cobra.Command {
	if options.Build != nil {
		buildProvenance = options.Build
	}
	unavailable := options.UnavailableCommand
	if unavailable == nil {
		unavailable = func(spec sessionsmodule.UnavailableCommandSpec) *cobra.Command {
			return &cobra.Command{Use: spec.Use, Short: spec.Short}
		}
	}
	return sessionsmodule.NewSessionCommandWithSeams(sessionsmodule.SessionCommandSeams{
		RunList:            runSessionListCommand,
		RunExport:          runSessionExportCommand,
		RunDelete:          runSessionDeleteCommand,
		RunPrune:           runSessionPruneCommand,
		RunBrowse:          runSessionBrowseCommand,
		RunRecap:           runSessionRecapCommand,
		UnavailableCommand: unavailable,
	})
}

func runSessionListCommand(cmd *cobra.Command, _ []string) error {
	source, _ := cmd.Flags().GetString("source")
	limit, _ := cmd.Flags().GetInt("limit")
	asJSON, _ := cmd.Flags().GetBool("json")
	// On a fresh install the goncho memory.db doesn't exist
	// yet (it's created lazily on the first turn write). For
	// an inventory command the absence of state isn't an
	// error — it's the empty state. Mutating commands
	// (export/delete/continue) keep their hard error in
	// openSessionDirectoryDB.
	db, err := OpenSessionDirectoryDB()
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
	sessions, err := sessionpkg.ListDirectorySessions(context.Background(), db, sessionpkg.DirectoryFilter{})
	if err != nil {
		return err
	}
	sessions = ApplySessionMirrorSources(sessions, effectiveSessionIndexMirrorPath())
	sessions = filterSessionDirectoryBySource(sessions, source, limit)
	if asJSON {
		return emitSessionListJSON(cmd, sessions)
	}
	if len(sessions) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No sessions found.")
		return nil
	}
	renderSessionDirectoryList(cmd.OutOrStdout(), sessions)
	return nil
}

// sessionListReportJSON is the wire shape for `gormes session list --json`.
// Build provenance leads, then the sessions array — same convention as
// the rest of the --json arc. internal/persistence/session.DirectoryEntry is
// tag-free; mirroring it here keeps presentation concerns out of the
// session package.
type sessionListReportJSON struct {
	Build    BuildProvenance        `json:"build"`
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
		Build:    currentBuildProvenance(),
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
	Build     BuildProvenance `json:"build"`
	SessionID string          `json:"session_id"`
	Format    string          `json:"format"`
	Content   string          `json:"content"`
}

func runSessionExportCommand(cmd *cobra.Command, args []string) error {
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
	out = applySessionExportMirrorSource(out, resolved, effectiveSessionIndexMirrorPath())

	if asJSON {
		body, marshalErr := json.MarshalIndent(sessionExportReportJSON{
			Build:     currentBuildProvenance(),
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
}

func runSessionDeleteCommand(cmd *cobra.Command, args []string) error {
	db, err := OpenSessionDirectoryDB()
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
					Build:       currentBuildProvenance(),
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
					Build:       currentBuildProvenance(),
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
				Build:       currentBuildProvenance(),
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
			Build:       currentBuildProvenance(),
			Action:      "deleted",
			RequestedID: args[0],
			ResolvedID:  resolved,
			Deleted:     true,
		})
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Deleted session '%s'.\n", resolved)
	return nil
}

// sessionDeleteReportJSON is the wire shape for `session delete --json`.
// Operator scripts iterating across hosts parse this to learn whether
// the session actually existed (`deleted: true`), wasn't there
// (`action: "not_found"`), or matched multiple ids (`action: "ambiguous"`).
type sessionDeleteReportJSON struct {
	Build       BuildProvenance `json:"build"`
	Action      string          `json:"action"`
	RequestedID string          `json:"requested_id"`
	ResolvedID  string          `json:"resolved_id,omitempty"`
	Deleted     bool            `json:"deleted"`
	Error       string          `json:"error,omitempty"`
}

func writeSessionDeleteJSON(out interface{ Write(p []byte) (int, error) }, report sessionDeleteReportJSON) error {
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(out, string(body))
	return nil
}

func runSessionPruneCommand(cmd *cobra.Command, _ []string) error {
	db, err := OpenSessionDirectoryDB()
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
			Build:         currentBuildProvenance(),
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
}

// sessionPruneReportJSON is the wire shape for `session prune --json`.
// Fleet automation running scheduled session GC parses this to audit
// how many sessions vanished per machine. Source filter is echoed so
// dashboards can correlate with the policy that drove the prune.
type sessionPruneReportJSON struct {
	Build         BuildProvenance `json:"build"`
	Action        string          `json:"action"`
	OlderThanDays int             `json:"older_than_days"`
	Source        string          `json:"source"`
	Pruned        int             `json:"pruned"`
}

func runSessionBrowseCommand(cmd *cobra.Command, _ []string) error {
	db, err := OpenSessionDirectoryDB()
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
}

func OpenSessionDirectoryDB() (*sql.DB, error) {
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

func ResolveContinueSessionFlag(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		value = "last"
	}
	db, err := OpenSessionDirectoryDB()
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

func effectiveSessionIndexMirrorPath() string {
	path := config.SessionIndexMirrorPath()
	if _, err := os.Stat(path); err == nil {
		return path
	}
	contract := config.CurrentProfileStorageContract()
	if contract.Scope == config.ProfileStorageScopeProfileRoot && contract.ProfileID == config.DefaultProfileID {
		baseMirror := filepath.Join(contract.BaseHome, "sessions", "index.yaml")
		if _, err := os.Stat(baseMirror); err == nil {
			return baseMirror
		}
	}
	return path
}

func ApplySessionMirrorSources(entries []sessionpkg.DirectoryEntry, mirrorPath string) []sessionpkg.DirectoryEntry {
	sources := readSessionMirrorSources(mirrorPath)
	if len(sources) == 0 {
		return entries
	}
	out := append([]sessionpkg.DirectoryEntry(nil), entries...)
	for i := range out {
		if source := strings.TrimSpace(sources[out[i].ID]); source != "" {
			out[i].Source = source
		}
	}
	return out
}

func filterSessionDirectoryBySource(entries []sessionpkg.DirectoryEntry, source string, limit int) []sessionpkg.DirectoryEntry {
	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" && limit <= 0 {
		return entries
	}
	out := make([]sessionpkg.DirectoryEntry, 0, len(entries))
	for _, entry := range entries {
		if source != "" && strings.ToLower(strings.TrimSpace(entry.Source)) != source {
			continue
		}
		out = append(out, entry)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func applySessionExportMirrorSource(markdown, sessionID, mirrorPath string) string {
	source := strings.TrimSpace(readSessionMirrorSources(mirrorPath)[sessionID])
	if source == "" {
		return markdown
	}
	lines := strings.Split(markdown, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "**Platform:** ") {
			lines[i] = "**Platform:** " + source + "  "
			return strings.Join(lines, "\n")
		}
	}
	return markdown
}

func readSessionMirrorSources(path string) map[string]string {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	out := make(map[string]string)
	section := ""
	currentID := ""
	for _, raw := range strings.Split(string(body), "\n") {
		line := strings.TrimRight(raw, " \t")
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case "sessions:":
			section = "sessions"
			currentID = ""
			continue
		case "lineage:":
			section = "lineage"
			currentID = ""
			continue
		}
		if section == "sessions" && strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") {
			key, sessionID, ok := strings.Cut(trimmed, ": ")
			if !ok || strings.TrimSpace(sessionID) == "" {
				continue
			}
			source, _, _ := strings.Cut(key, ":")
			if source = strings.ToLower(strings.TrimSpace(source)); source != "" {
				out[strings.TrimSpace(sessionID)] = source
			}
			continue
		}
		if section != "lineage" {
			continue
		}
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && strings.HasSuffix(trimmed, ":") {
			currentID = strings.TrimSuffix(trimmed, ":")
			continue
		}
		if currentID != "" && strings.HasPrefix(line, "    source: ") {
			source := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "source: ")))
			if source != "" {
				out[currentID] = source
			}
		}
	}
	return out
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

func CoalesceSessionNameArgs(argv []string) []string {
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

func NewTUISaveExportFunc() tui.SessionExportFunc {
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

func runSessionRecapCommand(cmd *cobra.Command, args []string) error {
	limit, _ := cmd.Flags().GetInt("limit")
	asJSON, _ := cmd.Flags().GetBool("json")

	smap, err := sessionpkg.OpenBolt(config.SessionDBPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if asJSON {
				return emitSessionListJSON(cmd, nil)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "No sessions found.")
			return nil
		}
		return err
	}
	defer smap.Close()

	ctx := context.Background()

	if len(args) > 0 {
		sessionID := strings.TrimSpace(args[0])
		result, err := sessionpkg.GenerateSessionRecap(ctx, smap, sessionID, sessionpkg.RecapConfig{MaxEntries: limit})
		if err != nil {
			return err
		}
		if asJSON {
			body, err := json.MarshalIndent(map[string]any{
				"build":      currentBuildProvenance(),
				"session_id": result.SessionID,
				"title":      result.Title,
				"source":     result.Source,
				"user_id":    result.UserID,
				"created_at": result.CreatedAt,
				"updated_at": result.UpdatedAt,
				"tokens_in":  result.TokensIn,
				"tokens_out": result.TokensOut,
				"not_found":  result.NotFound,
			}, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(body))
			return nil
		}
		fmt.Fprint(cmd.OutOrStdout(), result.HumanOutput())
		return nil
	}

	envelope, err := sessionpkg.GenerateRecap(ctx, smap, sessionpkg.RecapConfig{MaxEntries: limit})
	if err != nil {
		return err
	}
	if asJSON {
		entries := make([]map[string]any, len(envelope.Entries))
		for i, e := range envelope.Entries {
			entries[i] = map[string]any{
				"session_id": e.SessionID,
				"title":      e.Title,
				"source":     e.Source,
				"user_id":    e.UserID,
				"created_at": e.CreatedAt,
				"updated_at": e.UpdatedAt,
				"tokens_in":  e.TokensIn,
				"tokens_out": e.TokensOut,
			}
		}
		body, err := json.MarshalIndent(map[string]any{
			"build":          currentBuildProvenance(),
			"total_sessions": envelope.TotalSessions,
			"entries":        entries,
			"truncated":      envelope.Truncated,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(body))
		return nil
	}
	fmt.Fprint(cmd.OutOrStdout(), envelope.HumanOutput())
	return nil
}

func sqlOpenGoncho(path string) (*sql.DB, error) {
	db, err := sqlOpenGonchoRaw(path)
	if err == nil {
		return db, nil
	}
	if !memory.IsSQLiteCorruptionError(err) {
		return nil, err
	}
	if _, healErr := memory.SelfHealCorruptGonchoSQLite(path); healErr != nil {
		return nil, fmt.Errorf("%w; self-heal failed: %v", err, healErr)
	}
	return sqlOpenGonchoRaw(path)
}

func sqlOpenGonchoUnmigrated(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func sqlOpenGonchoRaw(path string) (*sql.DB, error) {
	db, err := sqlOpenGonchoUnmigrated(path)
	if err != nil {
		return nil, err
	}
	if err := memory.EnsureSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := gonchoservice.RunMigrations(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	for _, stmt := range []string{"PRAGMA journal_mode = WAL", "PRAGMA busy_timeout = 5000"} {
		if _, err := db.Exec(stmt); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return db, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
