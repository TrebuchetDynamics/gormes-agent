package gormescli

import (
	"net/http"
	"time"

	applogs "github.com/TrebuchetDynamics/gormes-agent/internal/app/logs"
)

type LogsEntry = applogs.Entry
type LogsContent = applogs.Content

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
