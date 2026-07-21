package gormescli

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	mcpcatalog "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/catalog"
)

type mcpCatalogEntryReport struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	Source           string `json:"source,omitempty"`
	Transport        string `json:"transport"`
	Auth             string `json:"auth"`
	InstallAvailable bool   `json:"install_available"`
}

type mcpCatalogReport struct {
	Build       BuildProvenance         `json:"build"`
	Action      string                  `json:"action"`
	Count       int                     `json:"count"`
	Entries     []mcpCatalogEntryReport `json:"entries"`
	Diagnostics []mcpcatalog.Diagnostic `json:"diagnostics"`
}

func newMCPCatalogCommand(opts MCPCommandOptions) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "catalog",
		Short:        "List approved MCP catalog entries",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			catalog := opts.Catalog()
			report := newMCPCatalogReport(opts.buildProvenance(), catalog)
			if asJSON {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(report)
			}
			writeMCPCatalogText(cmd, report)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable approved MCP catalog JSON")
	return cmd
}

func newMCPCatalogReport(build BuildProvenance, catalog mcpcatalog.Catalog) mcpCatalogReport {
	entries := catalog.List()
	report := mcpCatalogReport{
		Build:       build,
		Action:      "mcp_catalog",
		Count:       len(entries),
		Entries:     make([]mcpCatalogEntryReport, 0, len(entries)),
		Diagnostics: catalog.Diagnostics(),
	}
	for _, entry := range entries {
		report.Entries = append(report.Entries, mcpCatalogEntryReport{
			Name:             entry.Name,
			Description:      entry.Description,
			Source:           entry.Source,
			Transport:        entry.Transport.Type.String(),
			Auth:             entry.Auth.Type.String(),
			InstallAvailable: entry.Install != nil,
		})
	}
	return report
}

func writeMCPCatalogText(cmd *cobra.Command, report mcpCatalogReport) {
	out := cmd.OutOrStdout()
	if report.Count == 0 {
		fmt.Fprintln(out, "No approved MCP catalog entries are available.")
	} else {
		fmt.Fprintf(out, "Approved MCP catalog (%d):\n", report.Count)
		writer := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
		fmt.Fprintln(writer, "NAME\tTRANSPORT\tAUTH\tDESCRIPTION")
		for _, entry := range report.Entries {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", entry.Name, entry.Transport, entry.Auth, entry.Description)
		}
		_ = writer.Flush()
		fmt.Fprintln(out, "Install a config-safe catalog server with: gormes mcp install <name>")
	}
	for _, diagnostic := range report.Diagnostics {
		switch diagnostic.Kind {
		case mcpcatalog.DiagnosticFutureManifest:
			fmt.Fprintf(out, "Catalog entry %q requires a newer Gormes version.\n", diagnostic.Entry)
		default:
			fmt.Fprintf(out, "Catalog entry %q was omitted (%s).\n", diagnostic.Entry, diagnostic.Kind)
		}
	}
}
