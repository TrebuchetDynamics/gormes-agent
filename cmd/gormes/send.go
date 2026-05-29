package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

const sendBackendUnavailableEvidence = "send_backend_unavailable"

type sendCommandDeliveryFunc func(context.Context, string, string) (sendCommandResult, error)

type sendCommandResult struct {
	Success   bool   `json:"success"`
	Skipped   bool   `json:"skipped,omitempty"`
	DryRun    bool   `json:"dry_run,omitempty"`
	Target    string `json:"target,omitempty"`
	Message   string `json:"message,omitempty"`
	MessageID string `json:"message_id,omitempty"`
	Note      string `json:"note,omitempty"`
	Error     string `json:"error,omitempty"`
	Evidence  string `json:"evidence,omitempty"`
	Source    string `json:"source,omitempty"`
}

func newSendCommand(runtime rootRuntime) *cobra.Command {
	var target string
	var filePath string
	var subject string
	var listTargets bool
	var quiet bool
	var jsonMode bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:          "send [message]",
		Short:        "Send a message to a configured platform",
		SilenceUsage: true,
		Args:         cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if listTargets {
				filter := ""
				if len(args) > 0 {
					filter = strings.TrimSpace(strings.Join(args, " "))
				}
				return runSendListCommand(cmd, filter, jsonMode)
			}

			var targetMeta cli.TerminalResponseSanitizerMeta
			target, targetMeta = cli.StripLeakedTerminalResponsesWithMeta(strings.TrimSpace(target))
			target = strings.TrimSpace(target)
			if target == "" {
				return newExitCodeError(2, errors.New("gormes send: --to PLATFORM[:channel[:thread]] is required"))
			}

			body, err := cli.ResolveSendMessageBody(cli.SendMessageBodyOptions{
				Positional: strings.TrimSpace(strings.Join(args, " ")),
				FilePath:   strings.TrimSpace(filePath),
				Stdin:      cmd.InOrStdin(),
				StdinIsTTY: sendCommandStdinIsTTY(cmd, runtime),
			})
			if err != nil {
				return newExitCodeError(2, fmt.Errorf("gormes send: %w", err))
			}
			message := body.Text
			subjectText, subjectMeta := cli.StripLeakedTerminalResponsesWithMeta(strings.TrimSpace(subject))
			subjectText = strings.TrimSpace(subjectText)
			if subjectText != "" {
				message = subjectText + "\n\n" + strings.TrimLeftFunc(message, unicode.IsSpace)
			}

			if dryRun {
				result := sendCommandResult{
					Success:  true,
					Skipped:  true,
					DryRun:   true,
					Target:   target,
					Message:  message,
					Source:   body.Source,
					Evidence: sendCommandSanitizerEvidence(body.SanitizerMeta, subjectMeta, targetMeta),
					Note:     fmt.Sprintf("dry run: would send %d byte(s) to %s", len([]byte(message)), target),
				}
				return emitSendCommandResult(cmd.OutOrStdout(), cmd.ErrOrStderr(), result, jsonMode, quiet)
			}

			deliver := runtime.sendMessage
			if deliver == nil {
				deliver = defaultSendCommandBackend
			}
			result, err := deliver(cmd.Context(), target, message)
			if err != nil {
				cleaned := cli.StripLeakedTerminalResponses(err.Error())
				result = sendCommandResult{
					Success:  false,
					Target:   target,
					Error:    cleaned,
					Evidence: "send_backend_failed",
				}
			}
			if result.Target == "" {
				result.Target = target
			}
			if result.Error != "" {
				result.Error = cli.StripLeakedTerminalResponses(result.Error)
			}
			return emitSendCommandResult(cmd.OutOrStdout(), cmd.ErrOrStderr(), result, jsonMode, quiet)
		},
	}
	cmd.Flags().StringVarP(&target, "to", "t", "", "delivery target, e.g. telegram, telegram:123456, discord:#ops")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "read message body from PATH; use - for stdin")
	cmd.Flags().StringVarP(&subject, "subject", "s", "", "prepend a subject/header line before the message body")
	cmd.Flags().BoolVarP(&listTargets, "list", "l", false, "list available targets; optional positional filter")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress stdout on success")
	cmd.Flags().BoolVar(&jsonMode, "json", false, "emit JSON result")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "resolve and print the send payload without delivering")
	return cmd
}

func sendCommandStdinIsTTY(cmd *cobra.Command, runtime rootRuntime) bool {
	if cmd != nil {
		if in := cmd.InOrStdin(); in != nil && in != os.Stdin {
			return false
		}
	}
	if runtime.isTTY != nil {
		return runtime.isTTY()
	}
	return isStdinTTY()
}

func defaultSendCommandBackend(_ context.Context, target, _ string) (sendCommandResult, error) {
	return sendCommandResult{
		Success:  false,
		Target:   target,
		Error:    "send_backend_unavailable: standalone delivery is not configured for this platform yet; use `gormes gateway` or `gormes send --dry-run`",
		Evidence: sendBackendUnavailableEvidence,
	}, nil
}

func emitSendCommandResult(stdout, stderr io.Writer, result sendCommandResult, jsonMode, quiet bool) error {
	result = sanitizeSendCommandResult(result)
	if result.Error != "" {
		result.Success = false
	}
	if jsonMode {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(result)
	} else if result.Error != "" {
		fmt.Fprintf(stderr, "gormes send: %s\n", result.Error)
	} else if !quiet {
		if result.Note != "" {
			fmt.Fprintln(stdout, result.Note)
		} else {
			fmt.Fprintln(stdout, "sent")
		}
	}
	if result.Error != "" {
		return newExitCodeError(1, errors.New(result.Error))
	}
	return nil
}

func sendCommandSanitizerEvidence(metas ...cli.TerminalResponseSanitizerMeta) string {
	for _, meta := range metas {
		if meta.Evidence != "" {
			return meta.Evidence
		}
	}
	return ""
}

func sanitizeSendCommandResult(result sendCommandResult) sendCommandResult {
	result.Target = cli.StripLeakedTerminalResponses(result.Target)
	result.Message = cli.StripLeakedTerminalResponses(result.Message)
	result.MessageID = cli.StripLeakedTerminalResponses(result.MessageID)
	result.Note = cli.StripLeakedTerminalResponses(result.Note)
	result.Error = cli.StripLeakedTerminalResponses(result.Error)
	result.Evidence = cli.StripLeakedTerminalResponses(result.Evidence)
	result.Source = cli.StripLeakedTerminalResponses(result.Source)
	return result
}

func runSendListCommand(cmd *cobra.Command, platformFilter string, jsonMode bool) error {
	store := gateway.NewChannelDirectoryStore(config.GormesHome())
	dir, evidence := store.Load()
	if evidence.Code != "" && evidence.Code != "channel_directory_missing" {
		return newExitCodeError(1, fmt.Errorf("gormes send: %s", evidence.Code))
	}
	platforms := dir.Platforms
	if platforms == nil {
		platforms = map[string][]gateway.ChannelDirectoryEntry{}
	}
	if platformFilter != "" {
		key := strings.ToLower(strings.TrimSpace(platformFilter))
		entries := platforms[key]
		if len(entries) == 0 {
			configured := make([]string, 0, len(platforms))
			for platform := range platforms {
				configured = append(configured, platform)
			}
			sort.Strings(configured)
			if len(configured) == 0 {
				configured = append(configured, "(none)")
			}
			return newExitCodeError(1, fmt.Errorf("gormes send: no targets found for platform %q. Configured: %s", platformFilter, strings.Join(configured, ", ")))
		}
		platforms = map[string][]gateway.ChannelDirectoryEntry{key: entries}
		dir.Platforms = platforms
	}
	if jsonMode {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(map[string]any{"platforms": platforms})
		return nil
	}
	fmt.Fprintln(cmd.OutOrStdout(), dir.FormatForDisplay())
	return nil
}
