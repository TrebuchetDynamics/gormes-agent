package gormescli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/mcpstore"
	mcpcatalog "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/catalog"
)

const (
	mcpInstallEvidenceInstalled               = "installed"
	mcpInstallEvidenceNotFound                = "not_found"
	mcpInstallEvidenceExtendedInstallRequired = "extended_install_required"
	mcpInstallEvidenceConfigUnavailable       = "config_unavailable"
	mcpInstallEvidenceConfigRejected          = "config_rejected"
)

type mcpInstallReportJSON struct {
	Build     BuildProvenance `json:"build"`
	Action    string          `json:"action"`
	Name      string          `json:"name"`
	Evidence  string          `json:"evidence"`
	Transport string          `json:"transport,omitempty"`
	Auth      string          `json:"auth,omitempty"`
	Error     string          `json:"error,omitempty"`
}

func newMCPInstallCommand(opts MCPCommandOptions) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "install <name>",
		Short:        "Install a config-safe HTTP server from the approved MCP catalog",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPInstallCommand(cmd, opts, args[0], asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: {build, action, name, evidence, transport, auth, error}")
	return cmd
}

func runMCPInstallCommand(cmd *cobra.Command, opts MCPCommandOptions, identifier string, asJSON bool) error {
	entry, ok := opts.Catalog().Get(identifier)
	if !ok {
		return emitMCPInstallFailure(cmd, opts, asJSON, mcpInstallReportJSON{
			Build:    opts.buildProvenance(),
			Action:   "mcp_install",
			Name:     strings.TrimPrefix(identifier, "official/"),
			Evidence: mcpInstallEvidenceNotFound,
			Error:    "approved MCP catalog entry not found",
		})
	}
	report := mcpInstallReportJSON{
		Build:     opts.buildProvenance(),
		Action:    "mcp_install",
		Name:      entry.Name,
		Transport: entry.Transport.Type.String(),
		Auth:      entry.Auth.Type.String(),
	}
	if !configSafeHTTPEntry(entry) {
		report.Evidence = mcpInstallEvidenceExtendedInstallRequired
		report.Error = "catalog entry requires the extended install lifecycle"
		return emitMCPInstallFailure(cmd, opts, asJSON, report)
	}
	store := mcpstore.Store{Path: opts.MCPConfigPath()}
	if err := store.UpsertHTTP(entry.Name, mcpstore.HTTPRecord{
		URL:          entry.Transport.URL,
		OAuth:        entry.Auth.Type == mcpcatalog.AuthOAuth,
		Enabled:      true,
		ToolsInclude: entry.Tools.DefaultEnabled,
	}); err != nil {
		report.Evidence = mcpInstallEvidenceConfigUnavailable
		var storeErr *mcpstore.Error
		if errors.As(err, &storeErr) && (storeErr.Kind == mcpstore.ErrorRejected || storeErr.Kind == mcpstore.ErrorInvalidInput) {
			report.Evidence = mcpInstallEvidenceConfigRejected
		}
		report.Error = "MCP configuration could not be updated"
		return emitMCPInstallFailure(cmd, opts, asJSON, report)
	}
	report.Evidence = mcpInstallEvidenceInstalled
	if asJSON {
		return writeMCPInstallJSON(cmd, report)
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Installed approved MCP %q in the active Gormes profile. Start a new Gormes session to load its tools.\n", entry.Name)
	return err
}

func configSafeHTTPEntry(entry mcpcatalog.Entry) bool {
	return entry.Transport.Type == mcpcatalog.TransportHTTP &&
		entry.Install == nil &&
		len(entry.Auth.Env) == 0 &&
		(entry.Auth.Type == mcpcatalog.AuthOAuth || entry.Auth.Type == mcpcatalog.AuthNone)
}

func emitMCPInstallFailure(cmd *cobra.Command, opts MCPCommandOptions, asJSON bool, report mcpInstallReportJSON) error {
	if asJSON {
		if err := writeMCPInstallJSON(cmd, report); err != nil {
			return err
		}
	}
	return opts.exitCodeError(1, errors.New(report.Error))
}

func writeMCPInstallJSON(cmd *cobra.Command, report mcpInstallReportJSON) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
