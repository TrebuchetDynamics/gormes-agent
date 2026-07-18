package gormescli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/mcpstore"
	platformredaction "github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
	mcpconfig "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/descriptor"
	mcpprobe "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/probe"
)

const (
	mcpConfigEvidenceConfiguredUnverified = "configured_unverified"
	mcpConfigEvidenceInvalidInput         = "invalid_input"
	mcpConfigEvidenceAlreadyExists        = "already_exists"
	mcpConfigEvidenceRejected             = "config_rejected"
	mcpConfigEvidenceUnavailable          = "config_unavailable"
	mcpConfigEvidenceConfirmationRequired = "confirmation_required"
	mcpConfigEvidenceNotFound             = "not_found"
	mcpConfigEvidenceDisabled             = "disabled"
	mcpConfigEvidenceAuthRequired         = "auth_required"
	mcpConfigEvidenceUnsupportedTransport = "unsupported_transport"
	mcpConfigEvidenceTimeout              = "timeout"
	mcpConfigEvidenceConnectionFailed     = "connection_failed"
)

const (
	mcpProbeMaxTimeout       = 5 * time.Minute
	mcpProbeMaxRenderedTools = 100
	mcpProbeMaxToolNameRunes = 128
	mcpProbeMaxToolDescRunes = 240
)

type mcpAddReportJSON struct {
	Build     BuildProvenance `json:"build"`
	Action    string          `json:"action"`
	Name      string          `json:"name"`
	Evidence  string          `json:"evidence"`
	Transport string          `json:"transport,omitempty"`
	Auth      string          `json:"auth,omitempty"`
	Target    string          `json:"target,omitempty"`
	Error     string          `json:"error,omitempty"`
}

type mcpProbeToolJSON struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type mcpTestReportJSON struct {
	Build      BuildProvenance    `json:"build"`
	Action     string             `json:"action"`
	Name       string             `json:"name"`
	Evidence   string             `json:"evidence"`
	Transport  string             `json:"transport,omitempty"`
	ElapsedMS  int64              `json:"elapsed_ms"`
	ToolsCount int                `json:"tools_count"`
	Truncated  bool               `json:"truncated"`
	Tools      []mcpProbeToolJSON `json:"tools"`
	Error      string             `json:"error,omitempty"`
}

type mcpConfigureReportJSON struct {
	Build                  BuildProvenance `json:"build"`
	Action                 string          `json:"action"`
	Name                   string          `json:"name,omitempty"`
	Evidence               string          `json:"evidence"`
	SelectionMode          string          `json:"selection_mode,omitempty"`
	SelectedCount          int             `json:"selected_count"`
	RuntimeRefreshRequired bool            `json:"runtime_refresh_required"`
	Error                  string          `json:"error,omitempty"`
}

type mcpRemoveReportJSON struct {
	Build                BuildProvenance `json:"build"`
	Action               string          `json:"action"`
	Name                 string          `json:"name"`
	Evidence             string          `json:"evidence"`
	CredentialsPreserved bool            `json:"credentials_preserved"`
	ArtifactsPreserved   bool            `json:"artifacts_preserved"`
	Error                string          `json:"error,omitempty"`
}

type mcpListEntryJSON struct {
	Name          string `json:"name"`
	Transport     string `json:"transport"`
	Enabled       bool   `json:"enabled"`
	Status        string `json:"status"`
	Auth          string `json:"auth"`
	Target        string `json:"target,omitempty"`
	ToolsSelected int    `json:"tools_selected"`
}

type mcpListReportJSON struct {
	Build    BuildProvenance    `json:"build"`
	Action   string             `json:"action"`
	Evidence string             `json:"evidence"`
	Count    int                `json:"count"`
	Entries  []mcpListEntryJSON `json:"entries"`
	Error    string             `json:"error,omitempty"`
}

func newMCPConfigCommands(opts MCPCommandOptions) []*cobra.Command {
	return []*cobra.Command{newMCPAddCommand(opts), newMCPRemoveCommand(opts), newMCPListCommand(opts), newMCPTestCommand(opts), newMCPConfigureCommand(opts)}
}

type mcpToolNamesFlag struct {
	values []string
	set    bool
}

func (flag *mcpToolNamesFlag) Set(raw string) error {
	flag.set = true
	flag.values = append(flag.values, strings.Split(raw, ",")...)
	return nil
}

func (*mcpToolNamesFlag) String() string { return "" }
func (*mcpToolNamesFlag) Type() string   { return "toolNames" }

func newMCPConfigureCommand(opts MCPCommandOptions) *cobra.Command {
	var include mcpToolNamesFlag
	var none bool
	var all bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "configure <name>",
		Aliases:      []string{"config"},
		Short:        "Persist an explicit MCP tool selection without probing",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPConfigureCommand(cmd, opts, args[0], include.values, include.set, none, all, asJSON)
		},
	}
	cmd.Flags().Var(&include, "include", "exact MCP source tool names to enable (comma-separated or repeated; maximum 100)")
	cmd.Flags().BoolVar(&none, "none", false, "disable every tool from this server")
	cmd.Flags().BoolVar(&all, "all", false, "remove include/exclude filters so all current and future tools are enabled")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: {build, action, name, evidence, selection_mode, selected_count, runtime_refresh_required, error}")
	return cmd
}

func runMCPConfigureCommand(cmd *cobra.Command, opts MCPCommandOptions, name string, include []string, includeSet, none, all, asJSON bool) error {
	report := mcpConfigureReportJSON{
		Build:  opts.buildProvenance(),
		Action: "mcp_configure",
		Name:   safeMCPConfigureName(name),
	}
	modes := 0
	if includeSet {
		modes++
	}
	if none {
		modes++
	}
	if all {
		modes++
	}
	if modes != 1 {
		report.Evidence = mcpConfigEvidenceInvalidInput
		report.Error = "exactly one of --include, --none, or --all is required"
		return emitMCPConfigureFailure(cmd, opts, asJSON, report)
	}
	selection := mcpstore.ToolSelection{}
	switch {
	case includeSet:
		selection.Mode = mcpstore.ToolSelectionInclude
		selection.Include = append([]string(nil), include...)
		report.SelectionMode = string(selection.Mode)
		report.SelectedCount = len(include)
	case none:
		selection.Mode = mcpstore.ToolSelectionNone
		report.SelectionMode = string(selection.Mode)
	case all:
		selection.Mode = mcpstore.ToolSelectionAll
		report.SelectionMode = string(selection.Mode)
	}
	if err := (mcpstore.Store{Path: opts.MCPConfigPath()}).ConfigureTools(name, selection); err != nil {
		report.Evidence, report.Error = classifyMCPConfigError(err)
		return emitMCPConfigureFailure(cmd, opts, asJSON, report)
	}
	report.Evidence = "configured"
	report.RuntimeRefreshRequired = true
	if asJSON {
		return writeMCPJSON(cmd, report)
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Configured MCP %q tool selection (%s, %d selected). Reload or restart the active runtime to apply it.\n", report.Name, report.SelectionMode, report.SelectedCount)
	return err
}

func safeMCPConfigureName(name string) string {
	name = strings.TrimSpace(name)
	if len(name) > 64 {
		return ""
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && r != '_' && r != '-' {
			return ""
		}
	}
	return name
}

func emitMCPConfigureFailure(cmd *cobra.Command, opts MCPCommandOptions, asJSON bool, report mcpConfigureReportJSON) error {
	if asJSON {
		if err := writeMCPJSON(cmd, report); err != nil {
			return err
		}
	}
	return opts.exitCodeError(1, errors.New(report.Error))
}

func newMCPAddCommand(opts MCPCommandOptions) *cobra.Command {
	var urlValue string
	var auth string
	var force bool
	var asJSON bool
	var command string
	var args []string
	var env []string
	var preset string
	var connectTimeout float64
	cmd := &cobra.Command{
		Use:          "add <name>",
		Short:        "Add a custom HTTP MCP server without probing it",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, positional []string) error {
			unsupported := cmd.Flags().Changed("command") || cmd.Flags().Changed("args") || cmd.Flags().Changed("env") || cmd.Flags().Changed("preset") || cmd.Flags().Changed("connect-timeout")
			return runMCPAddCommand(cmd, opts, positional[0], urlValue, auth, force, asJSON, unsupported)
		},
	}
	cmd.Flags().StringVar(&urlValue, "url", "", "HTTP MCP endpoint URL")
	cmd.Flags().StringVar(&auth, "auth", "none", "authentication kind: none or oauth")
	cmd.Flags().BoolVar(&force, "force", false, "replace an existing server while preserving its tool selection")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: {build, action, name, evidence, transport, auth, target, error}")
	cmd.Flags().StringVar(&command, "command", "", "reserved for the future stdio lifecycle")
	cmd.Flags().StringSliceVar(&args, "args", nil, "reserved for the future stdio lifecycle")
	cmd.Flags().StringSliceVar(&env, "env", nil, "reserved for the future secret-aware stdio lifecycle")
	cmd.Flags().StringVar(&preset, "preset", "", "reserved for the future stdio preset lifecycle")
	cmd.Flags().Float64Var(&connectTimeout, "connect-timeout", 0, "reserved for the future connectivity-probe lifecycle")
	return cmd
}

func runMCPAddCommand(cmd *cobra.Command, opts MCPCommandOptions, name, rawURL, auth string, force, asJSON, unsupported bool) error {
	auth = strings.ToLower(strings.TrimSpace(auth))
	if auth == "" {
		auth = "none"
	}
	report := mcpAddReportJSON{
		Build:     opts.buildProvenance(),
		Action:    "mcp_add",
		Name:      strings.TrimSpace(name),
		Transport: "http",
		Auth:      auth,
	}
	if unsupported {
		report.Evidence = mcpConfigEvidenceInvalidInput
		report.Error = "stdio, env, preset, and probe options require the extended MCP lifecycle"
		return emitMCPAddFailure(cmd, opts, asJSON, report)
	}
	if auth != "none" && auth != "oauth" {
		report.Evidence = mcpConfigEvidenceInvalidInput
		report.Error = "auth must be none or oauth"
		return emitMCPAddFailure(cmd, opts, asJSON, report)
	}
	store := mcpstore.Store{Path: opts.MCPConfigPath()}
	err := store.AddHTTP(name, mcpstore.HTTPRecord{
		URL:     rawURL,
		OAuth:   auth == "oauth",
		Enabled: true,
	}, force)
	if err != nil {
		report.Evidence, report.Error = classifyMCPConfigError(err)
		return emitMCPAddFailure(cmd, opts, asJSON, report)
	}
	report.Evidence = mcpConfigEvidenceConfiguredUnverified
	report.Target = mcpstore.RedactedHTTPTarget(rawURL)
	if asJSON {
		return writeMCPJSON(cmd, report)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Configured unverified HTTP MCP %q (%s) in the active Gormes profile.\n", report.Name, report.Target)
	return err
}

func newMCPTestCommand(opts MCPCommandOptions) *cobra.Command {
	var timeout time.Duration
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "test <name>",
		Short:        "Test one configured HTTP MCP server and discover its tools",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPTestCommand(cmd, opts, args[0], timeout, cmd.Flags().Changed("timeout"), asJSON)
		},
	}
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "probe deadline (positive, maximum 5m; defaults to configured connect_timeout)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: {build, action, name, evidence, transport, elapsed_ms, tools_count, truncated, tools, error}")
	return cmd
}

func runMCPTestCommand(cmd *cobra.Command, opts MCPCommandOptions, name string, timeout time.Duration, timeoutSet, asJSON bool) error {
	report := mcpTestReportJSON{
		Build:  opts.buildProvenance(),
		Action: "mcp_test",
		Name:   strings.TrimSpace(name),
		Tools:  []mcpProbeToolJSON{},
	}
	if timeoutSet && (timeout <= 0 || timeout > mcpProbeMaxTimeout) {
		report.Evidence = mcpConfigEvidenceInvalidInput
		report.Error = "timeout must be positive and no greater than 5m"
		return emitMCPTestFailure(cmd, opts, asJSON, report)
	}
	record, found, err := (mcpstore.Store{Path: opts.MCPConfigPath()}).Server(name, mcpconfig.MCPConfigOptions{})
	if err != nil {
		report.Evidence, report.Error = classifyMCPConfigError(err)
		return emitMCPTestFailure(cmd, opts, asJSON, report)
	}
	if !found {
		report.Evidence = mcpConfigEvidenceNotFound
		report.Error = "MCP server was not found"
		return emitMCPTestFailure(cmd, opts, asJSON, report)
	}
	report.Transport = string(record.Status.Transport)
	if record.Status.Status == mcpconfig.MCPConfigStatusDisabled {
		report.Evidence = mcpConfigEvidenceDisabled
		report.Error = "MCP server is disabled"
		return emitMCPTestFailure(cmd, opts, asJSON, report)
	}
	if record.Auth == "oauth" {
		report.Evidence = mcpConfigEvidenceAuthRequired
		report.Error = "MCP OAuth login is required before testing this server"
		return emitMCPTestFailure(cmd, opts, asJSON, report)
	}
	if record.Definition.Transport != mcpconfig.MCPTransportHTTP {
		report.Evidence = mcpConfigEvidenceUnsupportedTransport
		report.Error = "this probe currently supports HTTP MCP servers only"
		return emitMCPTestFailure(cmd, opts, asJSON, report)
	}
	if !timeoutSet {
		timeout = record.Definition.ConnectTimeout
		if timeout <= 0 {
			timeout = time.Minute
		}
		if timeout > mcpProbeMaxTimeout {
			timeout = mcpProbeMaxTimeout
		}
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()
	started := time.Now()
	tools, err := mcpprobe.One(ctx, record.Definition, opts.MCPProbeConnector)
	report.ElapsedMS = max(time.Since(started).Milliseconds(), 0)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(ctx.Err(), context.Canceled) {
			report.Evidence = mcpConfigEvidenceTimeout
			report.Error = "MCP connection timed out"
		} else {
			report.Evidence = mcpConfigEvidenceConnectionFailed
			report.Error = "MCP connection failed"
		}
		return emitMCPTestFailure(cmd, opts, asJSON, report)
	}
	report.Evidence = "connected"
	report.ToolsCount = len(tools)
	report.Tools, report.Truncated = renderMCPProbeTools(tools)
	if asJSON {
		return writeMCPJSON(cmd, report)
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Connected to HTTP MCP %q (%dms); discovered %d tools.\n", report.Name, report.ElapsedMS, report.ToolsCount)
	for _, tool := range report.Tools {
		fmt.Fprintf(out, "  %s", tool.Name)
		if tool.Description != "" {
			fmt.Fprintf(out, " — %s", tool.Description)
		}
		fmt.Fprintln(out)
	}
	if report.Truncated {
		fmt.Fprintf(out, "  … %d additional tools omitted\n", report.ToolsCount-len(report.Tools))
	}
	return nil
}

func renderMCPProbeTools(tools []descriptor.RawTool) ([]mcpProbeToolJSON, bool) {
	rows := make([]mcpProbeToolJSON, 0, min(len(tools), mcpProbeMaxRenderedTools))
	for _, tool := range tools {
		rows = append(rows, mcpProbeToolJSON{
			Name:        sanitizeMCPProbeField(tool.Name, mcpProbeMaxToolNameRunes, "(unnamed)"),
			Description: sanitizeMCPProbeField(tool.Description, mcpProbeMaxToolDescRunes, ""),
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Name == rows[j].Name {
			return rows[i].Description < rows[j].Description
		}
		return rows[i].Name < rows[j].Name
	})
	truncated := len(rows) > mcpProbeMaxRenderedTools
	if truncated {
		rows = rows[:mcpProbeMaxRenderedTools]
	}
	return rows, truncated
}

func sanitizeMCPProbeField(value string, maxRunes int, fallback string) string {
	value = platformredaction.StripANSI(value)
	value = platformredaction.RedactSecrets(value)
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) > maxRunes {
		value = string(runes[:maxRunes-1]) + "…"
	}
	if value == "" {
		return fallback
	}
	return value
}

func emitMCPTestFailure(cmd *cobra.Command, opts MCPCommandOptions, asJSON bool, report mcpTestReportJSON) error {
	if asJSON {
		if err := writeMCPJSON(cmd, report); err != nil {
			return err
		}
	}
	return opts.exitCodeError(1, errors.New(report.Error))
}

func newMCPRemoveCommand(opts MCPCommandOptions) *cobra.Command {
	var confirmed bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "remove <name>",
		Aliases:      []string{"rm"},
		Short:        "Remove one MCP config entry while preserving credentials and artifacts",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPRemoveCommand(cmd, opts, args[0], confirmed, asJSON)
		},
	}
	cmd.Flags().BoolVarP(&confirmed, "yes", "y", false, "confirm config removal without an interactive prompt")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: {build, action, name, evidence, credentials_preserved, artifacts_preserved, error}")
	return cmd
}

func runMCPRemoveCommand(cmd *cobra.Command, opts MCPCommandOptions, name string, confirmed, asJSON bool) error {
	report := mcpRemoveReportJSON{
		Build:                opts.buildProvenance(),
		Action:               "mcp_remove",
		Name:                 strings.TrimSpace(name),
		CredentialsPreserved: true,
		ArtifactsPreserved:   true,
	}
	if !confirmed {
		report.Evidence = mcpConfigEvidenceConfirmationRequired
		report.Error = "MCP removal requires --yes"
		return emitMCPRemoveFailure(cmd, opts, asJSON, report)
	}
	removed, err := (mcpstore.Store{Path: opts.MCPConfigPath()}).Remove(name)
	if err != nil {
		report.Evidence, report.Error = classifyMCPConfigError(err)
		return emitMCPRemoveFailure(cmd, opts, asJSON, report)
	}
	if !removed {
		report.Evidence = mcpConfigEvidenceNotFound
		report.Error = "MCP server was not found"
		return emitMCPRemoveFailure(cmd, opts, asJSON, report)
	}
	report.Evidence = "removed"
	if asJSON {
		return writeMCPJSON(cmd, report)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Removed MCP %q from the active Gormes profile. Credentials and artifacts were preserved.\n", report.Name)
	return err
}

func emitMCPRemoveFailure(cmd *cobra.Command, opts MCPCommandOptions, asJSON bool, report mcpRemoveReportJSON) error {
	if asJSON {
		if err := writeMCPJSON(cmd, report); err != nil {
			return err
		}
	}
	return opts.exitCodeError(1, errors.New(report.Error))
}

func newMCPListCommand(opts MCPCommandOptions) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "list",
		Aliases:      []string{"ls"},
		Short:        "List configured MCP servers with sensitive fields redacted",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMCPListCommand(cmd, opts, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: {build, action, evidence, count, entries, error}")
	return cmd
}

func runMCPListCommand(cmd *cobra.Command, opts MCPCommandOptions, asJSON bool) error {
	rows, err := (mcpstore.Store{Path: opts.MCPConfigPath()}).List()
	if err != nil {
		evidence, message := classifyMCPConfigError(err)
		report := mcpListReportJSON{
			Build:    opts.buildProvenance(),
			Action:   "mcp_list",
			Evidence: evidence,
			Entries:  []mcpListEntryJSON{},
			Error:    message,
		}
		if asJSON {
			if writeErr := writeMCPJSON(cmd, report); writeErr != nil {
				return writeErr
			}
		}
		return opts.exitCodeError(1, errors.New(message))
	}
	report := mcpListReportJSON{
		Build:    opts.buildProvenance(),
		Action:   "mcp_list",
		Evidence: "listed",
		Count:    len(rows),
		Entries:  make([]mcpListEntryJSON, 0, len(rows)),
	}
	for _, row := range rows {
		report.Entries = append(report.Entries, mcpListEntryJSON{
			Name:          row.Name,
			Transport:     row.Transport,
			Enabled:       row.Enabled,
			Status:        row.Status,
			Auth:          row.Auth,
			Target:        row.Target,
			ToolsSelected: row.ToolsSelected,
		})
	}
	if asJSON {
		return writeMCPJSON(cmd, report)
	}
	if report.Count == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No MCP servers are configured in the active Gormes profile.")
		return err
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Configured MCP servers (%d):\n", report.Count)
	writer := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "NAME\tTRANSPORT\tSTATUS\tAUTH\tTOOLS\tTARGET")
	for _, entry := range report.Entries {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%d\t%s\n", entry.Name, entry.Transport, entry.Status, entry.Auth, entry.ToolsSelected, entry.Target)
	}
	return writer.Flush()
}

func classifyMCPConfigError(err error) (string, string) {
	var storeErr *mcpstore.Error
	if !errors.As(err, &storeErr) {
		return mcpConfigEvidenceUnavailable, "MCP configuration is unavailable"
	}
	switch storeErr.Kind {
	case mcpstore.ErrorInvalidInput:
		return mcpConfigEvidenceInvalidInput, "invalid MCP server configuration"
	case mcpstore.ErrorAlreadyExists:
		return mcpConfigEvidenceAlreadyExists, "MCP server already exists; use --force to replace it"
	case mcpstore.ErrorNotFound:
		return mcpConfigEvidenceNotFound, "MCP server was not found"
	case mcpstore.ErrorRejected:
		return mcpConfigEvidenceRejected, "MCP configuration was rejected"
	default:
		return mcpConfigEvidenceUnavailable, "MCP configuration is unavailable"
	}
}

func emitMCPAddFailure(cmd *cobra.Command, opts MCPCommandOptions, asJSON bool, report mcpAddReportJSON) error {
	if asJSON {
		if err := writeMCPJSON(cmd, report); err != nil {
			return err
		}
	}
	return opts.exitCodeError(1, errors.New(report.Error))
}

func writeMCPJSON(cmd *cobra.Command, report any) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
