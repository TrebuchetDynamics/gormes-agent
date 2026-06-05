package gormescli

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

var logsHTTPClient = NewLogsHTTPClient(5 * time.Second)
var logsEndpointURL = "http://127.0.0.1:43827/api/logs"

func newLogsCommand() *cobra.Command {
	return NewLogsCommand(func() BuildProvenance {
		return BuildProvenance{Version: Version, GitCommit: "test-git"}
	}, logsCommandOptions())
}

func logsCommandOptions() LogsCommandOptions {
	return LogsCommandOptions{
		Client:      logsHTTPClient,
		EndpointURL: logsEndpointURL,
		LogPath:     config.LogPath(),
	}
}
