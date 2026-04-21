package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/kernel"
	slackadapter "github.com/TrebuchetDynamics/gormes-agent/gormes/internal/slack"
)

var slackCmd = &cobra.Command{
	Use:          "slack",
	Short:        "Run Gormes as a Slack Socket Mode adapter",
	SilenceUsage: true,
	RunE:         runSlack,
}

func validateSlackConfig(cfg config.Config) error {
	if cfg.Slack.BotToken == "" {
		return fmt.Errorf("no Slack bot token — set GORMES_SLACK_BOT_TOKEN env or [slack].bot_token in config.toml")
	}
	if cfg.Slack.AppToken == "" {
		return fmt.Errorf("no Slack app token — set GORMES_SLACK_APP_TOKEN env or [slack].app_token in config.toml")
	}
	if cfg.Slack.AllowedChannelID == "" {
		return fmt.Errorf("slack: allowed_channel_id is required")
	}
	if !cfg.Slack.SocketMode {
		return fmt.Errorf("slack: socket_mode must be enabled")
	}
	return nil
}

func runSlack(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if p, ok := config.LegacyHermesHome(); ok {
		slog.Info("detected upstream Hermes home — Gormes uses XDG paths and does NOT read state from it; run `gormes migrate --from-hermes` (planned Phase 5.O) to import sessions and memory", "hermes_home", p)
	}

	if err := validateSlackConfig(cfg); err != nil {
		return err
	}

	key := slackadapter.SessionKey(cfg.Slack.AllowedChannelID)
	rt, err := openGatewayRuntime(cfg, gatewayRuntimeOptions{
		ChatKey:        key,
		ResumeOverride: cfg.Resume,
		RecallEnabled:  true,
	}, slog.Default())
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), kernel.ShutdownBudget)
		defer cancelShutdown()
		rt.Close(shutdownCtx)
	}()

	client := slackadapter.NewRealClient(cfg.Slack.BotToken, cfg.Slack.AppToken)
	bot := slackadapter.New(slackadapter.Config{
		AllowedChannelID: cfg.Slack.AllowedChannelID,
		ReplyInThread:    cfg.Slack.ReplyInThread,
		CoalesceMs:       cfg.Slack.CoalesceMs,
		SessionMap:       rt.SessionMap,
	}, client, rt.Kernel, slog.Default())

	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	rt.Start(rootCtx)
	return bot.Run(rootCtx)
}
