package mcplogin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/mcpstore"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

type ReportJSON struct {
	Build     BuildProvenance `json:"build"`
	Server    string          `json:"server"`
	Evidence  string          `json:"evidence"`
	Message   string          `json:"message,omitempty"`
	Available []string        `json:"available,omitempty"`
}

type Runtime struct {
	LoadConfig func() (tools.MCPConfigResolution, error)
	Store      *tools.MCPOAuthStore
	Flow       tools.MCPLoginFlow
}

type Options struct {
	ServerName string
	JSON       bool
	Build      BuildProvenance
	Stdout     io.Writer
}

func Run(ctx context.Context, runtime Runtime, opts Options) error {
	loadConfig := runtime.LoadConfig
	if loadConfig == nil {
		loadConfig = LoadDefaultMCPConfig
	}
	resolution, err := loadConfig()
	if err != nil {
		return fmt.Errorf("mcp_config_unavailable: %w", err)
	}
	store := runtime.Store
	if store == nil {
		store = tools.NewMCPOAuthStore()
	}
	flow := runtime.Flow
	if flow == nil {
		flow = tools.NoninteractiveLoginFlow()
	}
	result, err := tools.RunMCPLogin(ctx, resolution, store, flow, opts.ServerName)
	if err != nil {
		return err
	}
	if opts.JSON {
		body, marshalErr := json.MarshalIndent(ReportJSON{
			Build:     opts.Build,
			Server:    result.Server,
			Evidence:  string(result.Evidence),
			Message:   result.Message,
			Available: result.Available,
		}, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Fprintln(opts.Stdout, string(body))
	} else if result.Evidence == tools.MCPLoginEvidenceSaved {
		// Success path emits to stdout. On the error path cobra renders
		// the returned result on stderr; printing here would duplicate it.
		fmt.Fprintln(opts.Stdout, result.Error())
	}
	if result.Evidence == tools.MCPLoginEvidenceSaved {
		return nil
	}
	return result
}

func ExitCodeForError(err error) int {
	if err == nil {
		return 0
	}
	return 2
}

func LoadDefaultMCPConfig() (tools.MCPConfigResolution, error) {
	return (mcpstore.Store{}).Load(tools.MCPConfigOptions{})
}
