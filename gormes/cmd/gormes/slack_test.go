package main

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
)

func TestValidateSlackConfig_RejectsMissingBotToken(t *testing.T) {
	cfg := config.Config{}
	cfg.Slack.AppToken = "xapp-token"
	cfg.Slack.AllowedChannelID = "C123"

	err := validateSlackConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "no Slack bot token") {
		t.Fatalf("err = %v, want missing bot token error", err)
	}
}

func TestValidateSlackConfig_RejectsMissingAppToken(t *testing.T) {
	cfg := config.Config{}
	cfg.Slack.BotToken = "xoxb-token"
	cfg.Slack.AllowedChannelID = "C123"

	err := validateSlackConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "no Slack app token") {
		t.Fatalf("err = %v, want missing app token error", err)
	}
}

func TestValidateSlackConfig_RejectsMissingAllowedChannel(t *testing.T) {
	cfg := config.Config{}
	cfg.Slack.BotToken = "xoxb-token"
	cfg.Slack.AppToken = "xapp-token"
	cfg.Slack.SocketMode = true

	err := validateSlackConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "allowed_channel_id") {
		t.Fatalf("err = %v, want allowed_channel_id validation error", err)
	}
}

func TestValidateSlackConfig_RejectsSocketModeDisabled(t *testing.T) {
	cfg := config.Config{}
	cfg.Slack.BotToken = "xoxb-token"
	cfg.Slack.AppToken = "xapp-token"
	cfg.Slack.AllowedChannelID = "C123"

	err := validateSlackConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "socket_mode") {
		t.Fatalf("err = %v, want socket_mode validation error", err)
	}
}

func TestNewRootCmd_RegistersSlack(t *testing.T) {
	root := newRootCmd()
	names := make(map[string]bool)
	for _, cmd := range root.Commands() {
		names[cmd.Name()] = true
	}
	if !names["slack"] {
		t.Fatal("root command missing slack subcommand")
	}
}

func TestNewRootCmd_MakesResumePersistentForSlack(t *testing.T) {
	root := newRootCmd()
	if root.PersistentFlags().Lookup("resume") == nil {
		t.Fatal("root command missing persistent resume flag")
	}

	slack, _, err := root.Find([]string{"slack"})
	if err != nil {
		t.Fatalf("find slack command: %v", err)
	}
	if slack == nil {
		t.Fatal("slack command not found")
	}
	if slack.InheritedFlags().Lookup("resume") == nil {
		t.Fatal("slack command does not inherit resume flag")
	}
}

func TestSlackCommand_AcceptsResumeFlagPath(t *testing.T) {
	oldRunE := slackCmd.RunE
	t.Cleanup(func() {
		slackCmd.RunE = oldRunE
	})

	var gotResume string
	slackCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		gotResume, _ = cmd.Flags().GetString("resume")
		return nil
	}

	for _, args := range [][]string{
		{"slack", "--resume", "test-sid"},
		{"--resume", "test-sid", "slack"},
	} {
		root := newRootCmd()
		root.SetArgs(args)
		gotResume = ""
		if err := root.Execute(); err != nil {
			t.Fatalf("args %v: execute failed: %v", args, err)
		}
		if gotResume != "test-sid" {
			t.Fatalf("args %v: got resume %q, want %q", args, gotResume, "test-sid")
		}
	}
}
