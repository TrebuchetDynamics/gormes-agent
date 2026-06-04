package gormescli

import (
	"net/http"
	"time"

	"github.com/spf13/cobra"

	applogs "github.com/TrebuchetDynamics/gormes-agent/internal/app/logs"
)

type LogsBuildProvenance = applogs.BuildProvenance
type LogsCommandOptions = applogs.CommandOptions
type LogsEntry = applogs.Entry
type LogsContent = applogs.Content

func NewLogsCommand(build func() BuildProvenance, opts LogsCommandOptions) *cobra.Command {
	applogs.BuildProvenanceFunc = func() applogs.BuildProvenance {
		if build == nil {
			return applogs.BuildProvenance{}
		}
		provenance := build()
		return applogs.BuildProvenance{Version: provenance.Version, GitCommit: provenance.GitCommit}
	}
	return applogs.NewCommand(opts)
}

func RunLogs(cmd *cobra.Command, opts LogsCommandOptions) error { return applogs.Run(cmd, opts) }

func NewLogsHTTPClient(timeout time.Duration) *http.Client {
	return applogs.NewHTTPClient(timeout)
}

func ReadLogsContent(client *http.Client, endpointURL, logPath string) (LogsContent, error) {
	return applogs.ReadContent(client, endpointURL, logPath)
}

func ReadLogsTail(content LogsContent, limit int) string {
	return applogs.ReadTail(content, limit)
}

func FormatLogsEntries(entries []LogsEntry) []string {
	return applogs.FormatEntries(entries)
}
