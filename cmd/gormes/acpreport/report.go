package acpreport

import (
	"fmt"
	"io"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/protocols/acp"
)

// WriteBrowserBootstrapText writes the human-readable ACP browser bootstrap report.
func WriteBrowserBootstrapText(w io.Writer, report acp.BrowserBootstrapReport) {
	if report.DryRun {
		fmt.Fprintln(w, "ACP browser bootstrap dry-run")
	} else if report.Executed {
		fmt.Fprintln(w, "ACP browser bootstrap executed")
	} else {
		fmt.Fprintln(w, "ACP browser bootstrap")
	}
	if report.Evidence.Code != "" {
		fmt.Fprintf(w, "evidence: %s\n", report.Evidence.Code)
	}
	if report.Platform != "" {
		fmt.Fprintf(w, "platform: %s\n", report.Platform)
	}
	if report.NodePrefix != "" {
		fmt.Fprintf(w, "node_prefix: %s\n", report.NodePrefix)
	}
	for _, step := range report.Steps {
		fmt.Fprintf(w, "- %s: %s", step.Name, step.Status)
		if len(step.Command) > 0 {
			fmt.Fprintf(w, " `%s`", strings.Join(step.Command, " "))
		}
		if step.Message != "" {
			fmt.Fprintf(w, " — %s", step.Message)
		}
		fmt.Fprintln(w)
	}
	if report.Message != "" {
		fmt.Fprintf(w, "message: %s\n", report.Message)
	}
}

// WriteClientText writes the human-readable ACP debug client connection report.
func WriteClientText(w io.Writer, result acp.ClientResult) {
	if result.OK {
		fmt.Fprintln(w, "ACP client connected")
	} else {
		fmt.Fprintln(w, "ACP client degraded")
	}
	if result.SessionKey != "" {
		fmt.Fprintf(w, "session_key: %s\n", result.SessionKey)
	}
	if result.SessionID != "" {
		fmt.Fprintf(w, "session_id: %s\n", result.SessionID)
	}
	if result.SessionLabel != "" {
		fmt.Fprintf(w, "session_label: %s\n", result.SessionLabel)
	}
	if result.ProvenanceMode != "" {
		fmt.Fprintf(w, "provenance: %s\n", result.ProvenanceMode)
	}
	if result.Reset {
		fmt.Fprintln(w, "reset_session: true")
	}
	if result.Evidence.Code != "" {
		fmt.Fprintf(w, "evidence: %s\n", result.Evidence.Code)
	}
	if result.Evidence.Reason != "" {
		fmt.Fprintf(w, "reason: %s\n", result.Evidence.Reason)
	}
	if result.Message != "" {
		fmt.Fprintf(w, "message: %s\n", result.Message)
	}
}
