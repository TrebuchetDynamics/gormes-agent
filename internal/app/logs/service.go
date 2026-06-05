package logs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

var BuildProvenanceFunc = func() BuildProvenance { return BuildProvenance{} }

type exitCodeError struct {
	code int
	err  error
}

func (e exitCodeError) Error() string { return e.err.Error() }
func (e exitCodeError) Unwrap() error { return e.err }
func (e exitCodeError) ExitCode() int { return e.code }

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

type CommandOptions struct {
	Client      *http.Client
	EndpointURL string
	LogPath     string
}

// logsReportJSON is the wire shape for `logs --json`. Fleet log
// aggregation pipelines parse this to ingest entries directly.
// `source: "gateway"` means a live gateway responded; `source: "file"`
// means we fell back to the on-disk log file (raw content). `entries`
// is populated only on the gateway path; `content`/`path` only on file.
type reportJSON struct {
	Build   BuildProvenance `json:"build"`
	Source  string          `json:"source"`
	Entries []Entry         `json:"entries,omitempty"`
	Path    string          `json:"path,omitempty"`
	Content string          `json:"content,omitempty"`
}

type inputErrorReportJSON struct {
	Build  BuildProvenance `json:"build"`
	Action string          `json:"action"`
	Error  string          `json:"error"`
}

func NewCommand(opts CommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Show recent Gormes gateway logs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return Run(cmd, opts)
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: {build, source: 'gateway'|'file', entries|content, path}")
	return cmd
}

func Run(cmd *cobra.Command, opts CommandOptions) error {
	out := cmd.OutOrStdout()
	asJSON, _ := cmd.Flags().GetBool("json")
	content, err := ReadContent(opts.Client, opts.EndpointURL, opts.LogPath)
	if err != nil {
		msg := fmt.Sprintf("no gateway running and no log file found: %v", err)
		if asJSON {
			return emitInputError(cmd, "no_logs", msg)
		}
		return fmt.Errorf("%s", msg)
	}
	if asJSON {
		body, marshalErr := json.MarshalIndent(reportJSON{
			Build:   BuildProvenanceFunc(),
			Source:  content.Source,
			Entries: content.Entries,
			Path:    content.Path,
			Content: content.Content,
		}, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Fprintln(out, string(body))
		return nil
	}
	if content.Source == "file" {
		fmt.Fprint(out, content.Content)
		return nil
	}
	if len(content.Entries) == 0 {
		fmt.Fprintln(out, "No log entries.")
		return nil
	}
	for _, line := range FormatEntries(content.Entries) {
		fmt.Fprintln(out, line)
	}
	return nil
}

func emitInputError(cmd *cobra.Command, action, errMsg string) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(inputErrorReportJSON{
		Build:  BuildProvenanceFunc(),
		Action: action,
		Error:  errMsg,
	})
	return exitCodeError{code: 1, err: fmt.Errorf("%s", errMsg)}
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
