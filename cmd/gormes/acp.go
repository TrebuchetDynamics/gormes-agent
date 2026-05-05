package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/acp"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

func newACPCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "acp",
		Short: "Run ACP bridge tools",
	}
	cmd.AddCommand(newACPClientCommand())
	return cmd
}

func newACPClientCommand() *cobra.Command {
	var (
		opts          acp.ClientOptions
		provenanceRaw string
		jsonOut       bool
	)
	opts.ServerCommand = "gormes"

	cmd := &cobra.Command{
		Use:   "client",
		Short: "Connect a debug ACP client to the Go-native ACP server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			mode, err := acp.ParseProvenanceMode(provenanceRaw)
			if err != nil {
				return newExitCodeError(2, err)
			}
			opts.ProvenanceMode = mode
			return runACPClientCommand(cmd, opts, jsonOut)
		},
	}
	cmd.Flags().StringVar(&opts.SessionKey, "session", "", "session key to bridge, for example agent:main:main")
	cmd.Flags().StringVar(&opts.SessionLabel, "session-label", "", "session title or id to resolve before bridging")
	cmd.Flags().BoolVar(&opts.RequireExisting, "require-existing", false, "fail when the resolved session key does not already exist")
	cmd.Flags().BoolVar(&opts.ResetSession, "reset-session", false, "clear and reinitialize the resolved session key before connecting")
	cmd.Flags().BoolVar(&opts.NoPrefixCWD, "no-prefix-cwd", false, "do not prepend working-directory provenance to bridged prompts")
	cmd.Flags().StringVar(&provenanceRaw, "provenance", string(acp.ProvenanceOff), "provenance mode: off, meta, or meta+receipt")
	cmd.Flags().StringVar(&opts.CWD, "cwd", "", "working directory to expose to the ACP bridge")
	cmd.Flags().StringVar(&opts.ServerCommand, "server", opts.ServerCommand, "ACP server command label")
	cmd.Flags().StringArrayVar(&opts.ServerArgs, "server-args", nil, "additional ACP server argument, repeatable")
	cmd.Flags().BoolVar(&opts.ServerVerbose, "server-verbose", false, "request verbose ACP server logging")
	cmd.Flags().BoolVar(&opts.Verbose, "verbose", false, "print verbose ACP client diagnostics")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "write machine-readable ACP client result")
	return cmd
}

func runACPClientCommand(cmd *cobra.Command, opts acp.ClientOptions, jsonOut bool) error {
	smap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		return newExitCodeError(2, fmt.Errorf("acp client session store unavailable: %w", err))
	}
	defer smap.Close()

	result, err := (acp.ClientBridge{
		Resolver:  acp.NewSessionMapResolver(smap),
		Connector: acp.LocalClientConnector{},
	}).Run(cmd.Context(), opts)
	if err != nil {
		return newExitCodeError(2, err)
	}

	if jsonOut {
		if err := json.NewEncoder(cmd.OutOrStdout()).Encode(result); err != nil {
			return err
		}
	} else {
		writeACPClientText(cmd.OutOrStdout(), result)
	}
	if !result.OK {
		return newExitCodeError(1, errors.New(result.Message))
	}
	return nil
}

func writeACPClientText(w io.Writer, result acp.ClientResult) {
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
