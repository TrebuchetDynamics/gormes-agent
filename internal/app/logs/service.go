package logs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// Entry is one structured gateway log entry returned by the live gateway logs endpoint.
type Entry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

// Content is the resolved logs payload. Source is "gateway" for live gateway
// responses and "file" for on-disk fallback content.
type Content struct {
	Source  string
	Entries []Entry
	Path    string
	Content string
}

// NewHTTPClient returns the bounded gateway logs HTTP client used by the CLI.
func NewHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

// ReadContent fetches logs from the live gateway, falling back to logPath when
// the gateway is unreachable.
func ReadContent(client *http.Client, endpointURL, logPath string) (Content, error) {
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Get(endpointURL)
	if err != nil {
		data, fileErr := os.ReadFile(logPath)
		if fileErr != nil {
			return Content{}, fileErr
		}
		return Content{Source: "file", Path: logPath, Content: string(data)}, nil
	}
	defer resp.Body.Close()

	var result struct {
		Entries []Entry `json:"entries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Content{}, fmt.Errorf("failed to read logs: %w", err)
	}
	return Content{Source: "gateway", Entries: result.Entries}, nil
}

// ReadTail returns the last limit formatted lines from either gateway or file logs.
func ReadTail(content Content, limit int) string {
	var lines []string
	if content.Source == "gateway" {
		lines = FormatEntries(content.Entries)
	} else {
		lines = SplitLines(content.Content)
	}
	lines = TailLines(lines, limit)
	return strings.Join(lines, "\n")
}

// FormatEntries renders gateway entries in the CLI text format.
func FormatEntries(entries []Entry) []string {
	lines := make([]string, 0, len(entries))
	for _, e := range entries {
		lines = append(lines, fmt.Sprintf("[%s] %s: %s", e.Time, e.Level, e.Message))
	}
	return lines
}

// SplitLines splits raw log-file content while ignoring a trailing newline.
func SplitLines(content string) []string {
	content = strings.TrimRight(content, "\n")
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}

// TailLines returns the last limit lines. A non-positive limit defaults to 20.
func TailLines(lines []string, limit int) []string {
	if limit <= 0 {
		limit = 20
	}
	if len(lines) <= limit {
		return lines
	}
	return lines[len(lines)-limit:]
}
