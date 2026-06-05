package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	telegram "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/telegram"
	"github.com/TrebuchetDynamics/gormes-agent/internal/app/audiotools"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	channelsmodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/channels"
)

type gatewayChannelFactory func(config.Config, *slog.Logger) (gateway.Channel, error)

type gatewayChannelFactories struct {
	Telegram gatewayChannelFactory
	Discord  gatewayChannelFactory
	Slack    gatewayChannelFactory
	Teams    gatewayChannelFactory
	Yuanbao  gatewayChannelFactory
	Navivox  gatewayChannelFactory
	SimpleX  gatewayChannelFactory
}

func defaultGatewayChannelFactories() gatewayChannelFactories {
	return gatewayChannelFactories{
		Telegram: func(cfg config.Config, log *slog.Logger) (gateway.Channel, error) {
			tc, err := telegram.NewRealClient(cfg.Telegram.BotToken)
			if err != nil {
				return nil, err
			}
			return telegram.New(telegram.Config{
				AllowedChatID:       cfg.Telegram.AllowedChatID,
				AllowedChatIDs:      cfg.Telegram.AllowedChatIDs(),
				AllowedUserIDs:      cfg.Telegram.AllowedUserIDs,
				FirstRunDiscovery:   cfg.Telegram.FirstRunDiscovery,
				RequireMention:      cfg.Telegram.RequireMention,
				GuestMode:           cfg.Telegram.GuestMode,
				BotUsername:         cfg.Telegram.BotUsername,
				Notifications:       cfg.Telegram.Notifications,
				AudioTranscriber:    audiotools.ResolveTelegramAudioTranscriber(),
				DynamicCommands:     gatewayTelegramDynamicCommands(context.Background(), cfg),
				TokenLockDir:        config.GatewayLockDir(),
				ModelPickerResolver: gateway.NewModelPickerResolver(&gateway.SessionModelOverride{}),
			}, tc, log), nil
		},
		Discord: gormescli.NewDiscordGatewayChannel,
		Slack:   gormescli.NewSlackGatewayChannel,
		Teams: func(config.Config, *slog.Logger) (gateway.Channel, error) {
			return nil, errors.New("teams_live_transport_unavailable: live Bot Framework binding is not implemented; Teams is fakeable only in this slice")
		},
		Yuanbao: func(config.Config, *slog.Logger) (gateway.Channel, error) {
			return nil, errors.New("yuanbao_runtime_unavailable: live Yuanbao transport is not implemented; the runtime slice binds fake clients only")
		},
		Navivox: channelsmodule.NewNavivoxGatewayChannel,
		SimpleX: gormescli.NewSimpleXGatewayChannel,
	}
}

func gatewayTelegramDynamicCommands(ctx context.Context, cfg config.Config) []gateway.PlatformCommand {
	runtime := skills.NewRuntime(cfg.SkillsRoot(), cfg.Skills.MaxDocumentBytes, cfg.Skills.SelectionCap, cfg.SkillsUsageLogPath())
	skillCommands, _, err := runtime.SkillSlashCommands(ctx, skills.RuntimeOptions{})
	if err != nil || len(skillCommands) == 0 {
		return nil
	}
	commands := make([]gateway.PlatformCommand, 0, len(skillCommands))
	for _, cmd := range skillCommands {
		commands = append(commands, gateway.PlatformCommand{
			Name:        strings.TrimPrefix(cmd.Command, "/"),
			Description: cmd.Description,
		})
	}
	return commands
}

func registerConfiguredGatewayChannels(mgr *gateway.Manager, cfg config.Config, allowedChats map[string]string, allowDiscovery map[string]bool, factories gatewayChannelFactories, status gateway.RuntimeStatusWriter, log *slog.Logger) (int, error) {
	if log == nil {
		log = slog.Default()
	}
	registered := 0

	tgAccounts := cfg.Telegram.Accounts
	if len(tgAccounts) == 0 && cfg.Telegram.BotToken != "" {
		if factories.Telegram == nil {
			return registered, fmt.Errorf("register telegram: missing channel factory")
		}
		ch, err := factories.Telegram(cfg, log)
		if err != nil {
			return registered, err
		}
		if err := mgr.Register(ch); err != nil {
			return registered, fmt.Errorf("register telegram: %w", err)
		}
		if cfg.Telegram.AllowedChatID != 0 {
			allowedChats["telegram"] = strconv.FormatInt(cfg.Telegram.AllowedChatID, 10)
		}
		allowDiscovery["telegram"] = cfg.Telegram.FirstRunDiscovery
		registered++
		log.Info("gateway: telegram channel enabled", "allowed_chat_id", cfg.Telegram.AllowedChatID, "allowed_user_count", len(cfg.Telegram.AllowedUserIDs))
	}
	for accountID, acct := range tgAccounts {
		if acct.BotToken == "" {
			writeGatewayChannelDegraded(status, "telegram", fmt.Sprintf("telegram account %s: missing bot_token", accountID))
			log.Warn("gateway: telegram account disabled by missing token", "account", accountID)
			continue
		}
		if factories.Telegram == nil {
			return registered, fmt.Errorf("register telegram: missing channel factory")
		}
		acctCfg := cfg
		acctCfg.Telegram.BotToken = acct.BotToken
		acctCfg.Telegram.AllowedChatID = acct.AllowedChatID
		acctCfg.Telegram.AllowedUserIDs = acct.AllowedUserIDs
		acctCfg.Telegram.AccountID = accountID
		ch, err := factories.Telegram(acctCfg, log)
		if err != nil {
			writeGatewayChannelDegraded(status, "telegram", fmt.Sprintf("telegram account %s: startup failed: %s", accountID, err.Error()))
			log.Warn("gateway: telegram account startup failed", "account", accountID, "err", err)
			continue
		}
		if err := mgr.Register(ch); err != nil {
			return registered, fmt.Errorf("register telegram account %s: %w", accountID, err)
		}
		if acct.AllowedChatID != 0 {
			allowedChats["telegram:"+accountID] = strconv.FormatInt(acct.AllowedChatID, 10)
		}
		allowDiscovery["telegram:"+accountID] = cfg.Telegram.FirstRunDiscovery
		registered++
		log.Info("gateway: telegram account enabled", "account", accountID, "allowed_chat_id", acct.AllowedChatID, "allowed_user_count", len(acct.AllowedUserIDs))
	}

	discordAccounts := cfg.Discord.Accounts
	if len(discordAccounts) == 0 && cfg.Discord.Enabled() {
		if factories.Discord == nil {
			return registered, fmt.Errorf("register discord: missing channel factory")
		}
		ch, err := factories.Discord(cfg, log)
		if err != nil {
			return registered, err
		}
		if err := mgr.Register(ch); err != nil {
			return registered, fmt.Errorf("register discord: %w", err)
		}
		if cfg.Discord.AllowedChannelID != "" {
			allowedChats["discord"] = cfg.Discord.AllowedChannelID
		}
		allowDiscovery["discord"] = cfg.Discord.FirstRunDiscovery
		registered++
		log.Info("gateway: discord channel enabled", "allowed_channel_id", cfg.Discord.AllowedChannelID)
	}
	for accountID, acct := range discordAccounts {
		if acct.Token == "" {
			writeGatewayChannelDegraded(status, "discord", fmt.Sprintf("discord account %s: missing token", accountID))
			log.Warn("gateway: discord account disabled by missing token", "account", accountID)
			continue
		}
		if factories.Discord == nil {
			return registered, fmt.Errorf("register discord: missing channel factory")
		}
		acctCfg := cfg
		acctCfg.Discord.Token = acct.Token
		acctCfg.Discord.AllowedChannelID = acct.AllowedChannelID
		acctCfg.Discord.AllowedChannels = acct.AllowedChannels
		acctCfg.Discord.AccountID = accountID
		ch, err := factories.Discord(acctCfg, log)
		if err != nil {
			writeGatewayChannelDegraded(status, "discord", fmt.Sprintf("discord account %s: startup failed: %s", accountID, err.Error()))
			log.Warn("gateway: discord account startup failed", "account", accountID, "err", err)
			continue
		}
		if err := mgr.Register(ch); err != nil {
			return registered, fmt.Errorf("register discord account %s: %w", accountID, err)
		}
		if acct.AllowedChannelID != "" {
			allowedChats["discord:"+accountID] = acct.AllowedChannelID
		}
		allowDiscovery["discord:"+accountID] = cfg.Discord.FirstRunDiscovery
		registered++
		log.Info("gateway: discord account enabled", "account", accountID, "allowed_channel_id", acct.AllowedChannelID)
	}

	slackAccounts := cfg.Slack.Accounts
	if len(slackAccounts) == 0 && cfg.Slack.Enabled {
		if cfg.Slack.AllowedChannelID != "" {
			allowedChats["slack"] = cfg.Slack.AllowedChannelID
		}
		allowDiscovery["slack"] = cfg.Slack.FirstRunDiscovery

		if missing := missingSlackCredentials(cfg.Slack); len(missing) > 0 {
			errText := "slack: missing " + strings.Join(missing, ",")
			writeGatewayChannelDegraded(status, "slack", errText)
			log.Warn("gateway: slack channel disabled by missing credentials", "missing", strings.Join(missing, ","))
		} else {
			if factories.Slack == nil {
				return registered, fmt.Errorf("register slack: missing channel factory")
			}
			ch, err := factories.Slack(cfg, log)
			if err != nil {
				errText := "slack: startup failed: " + err.Error()
				writeGatewayChannelDegraded(status, "slack", errText)
				log.Warn("gateway: slack channel startup failed", "err", err)
			} else {
				if err := mgr.Register(ch); err != nil {
					return registered, fmt.Errorf("register slack: %w", err)
				}
				registered++
				log.Info("gateway: slack channel enabled", "allowed_channel_id", cfg.Slack.AllowedChannelID)
			}
		}
	}
	for accountID, acct := range slackAccounts {
		if acct.BotToken == "" || acct.AppToken == "" {
			writeGatewayChannelDegraded(status, "slack", fmt.Sprintf("slack account %s: missing bot_token or app_token", accountID))
			log.Warn("gateway: slack account disabled by missing token", "account", accountID)
			continue
		}
		if factories.Slack == nil {
			return registered, fmt.Errorf("register slack: missing channel factory")
		}
		acctCfg := cfg
		acctCfg.Slack.BotToken = acct.BotToken
		acctCfg.Slack.AppToken = acct.AppToken
		acctCfg.Slack.AllowedChannelID = acct.AllowedChannelID
		acctCfg.Slack.AccountID = accountID
		ch, err := factories.Slack(acctCfg, log)
		if err != nil {
			writeGatewayChannelDegraded(status, "slack", fmt.Sprintf("slack account %s: startup failed: %s", accountID, err.Error()))
			log.Warn("gateway: slack account startup failed", "account", accountID, "err", err)
			continue
		}
		if err := mgr.Register(ch); err != nil {
			return registered, fmt.Errorf("register slack account %s: %w", accountID, err)
		}
		if acct.AllowedChannelID != "" {
			allowedChats["slack:"+accountID] = acct.AllowedChannelID
		}
		allowDiscovery["slack:"+accountID] = cfg.Slack.FirstRunDiscovery
		registered++
		log.Info("gateway: slack account enabled", "account", accountID, "allowed_channel_id", acct.AllowedChannelID)
	}

	if cfg.Teams.Enabled {
		if missing := cfg.Teams.MissingCredentials(); len(missing) > 0 {
			errText := "teams: missing " + strings.Join(missing, ",")
			writeGatewayChannelDegraded(status, "teams", errText)
			log.Warn("gateway: teams channel disabled by missing credentials", "missing", strings.Join(missing, ","))
		} else {
			if factories.Teams == nil {
				return registered, fmt.Errorf("register teams: missing channel factory")
			}
			ch, err := factories.Teams(cfg, log)
			if err != nil {
				errText := "teams: startup failed: " + err.Error()
				writeGatewayChannelDegraded(status, "teams", errText)
				log.Warn("gateway: teams channel startup failed", "err", err)
			} else {
				if err := mgr.Register(ch); err != nil {
					return registered, fmt.Errorf("register teams: %w", err)
				}
				registered++
				log.Info("gateway: teams channel enabled", "port", cfg.Teams.EffectivePort(), "allowed_user_count", len(cfg.Teams.AllowedUserIDs()))
			}
		}
	}

	if cfg.Yuanbao.Enabled {
		if cfg.Yuanbao.AllowedConversationID != "" {
			allowedChats["yuanbao"] = cfg.Yuanbao.AllowedConversationID
		}
		allowDiscovery["yuanbao"] = cfg.Yuanbao.FirstRunDiscovery

		if missing := cfg.Yuanbao.MissingCredentials(); len(missing) > 0 {
			errText := "yuanbao: missing " + strings.Join(missing, ",")
			writeGatewayChannelDegraded(status, "yuanbao", errText)
			log.Warn("gateway: yuanbao channel disabled by missing credentials", "missing", strings.Join(missing, ","))
			return registered, nil
		}
		if factories.Yuanbao == nil {
			return registered, fmt.Errorf("register yuanbao: missing channel factory")
		}
		ch, err := factories.Yuanbao(cfg, log)
		if err != nil {
			errText := "yuanbao: startup failed: " + err.Error()
			writeGatewayChannelDegraded(status, "yuanbao", errText)
			log.Warn("gateway: yuanbao channel startup failed", "err", err)
			return registered, nil
		}
		if err := mgr.Register(ch); err != nil {
			return registered, fmt.Errorf("register yuanbao: %w", err)
		}
		registered++
		log.Info("gateway: yuanbao channel enabled", "allowed_conversation_id", cfg.Yuanbao.AllowedConversationID)
	}

	if cfg.Navivox.Enabled {
		if factories.Navivox == nil {
			return registered, fmt.Errorf("register navivox: missing channel factory")
		}
		ch, err := factories.Navivox(cfg, log)
		if err != nil {
			writeGatewayChannelDegraded(status, channelsmodule.NavivoxPlatformName, "navivox: startup failed: "+err.Error())
			return registered, fmt.Errorf("register navivox: %w", err)
		}
		if err := mgr.Register(ch); err != nil {
			return registered, fmt.Errorf("register navivox: %w", err)
		}
		registered++
		log.Info("gateway: navivox channel enabled",
			"bind_host", cfg.Navivox.BindHost,
			"port", cfg.Navivox.Port,
			"exposure_mode", cfg.Navivox.ExposureMode,
			"auth_mode", cfg.Navivox.AuthMode)
	}

	if simplexInfo := gormescli.SimpleXEnv(os.LookupEnv); simplexInfo.Enabled {
		if simplexInfo.HomeChannel != "" {
			allowedChats[simplexInfo.Platform] = simplexInfo.HomeChannel
		}
		allowDiscovery[simplexInfo.Platform] = false
		if factories.SimpleX == nil {
			return registered, fmt.Errorf("register simplex: missing channel factory")
		}
		ch, err := factories.SimpleX(cfg, log)
		if err != nil {
			errText := "simplex: startup failed: " + err.Error()
			writeGatewayChannelDegraded(status, simplexInfo.Platform, errText)
			log.Warn("gateway: simplex channel startup failed", "err", err)
		} else {
			if err := mgr.Register(ch); err != nil {
				return registered, fmt.Errorf("register simplex: %w", err)
			}
			registered++
			log.Info("gateway: simplex channel enabled", "home_channel", simplexInfo.HomeChannel, "allowed_user_count", len(simplexInfo.AllowedUsers), "allow_all_users", simplexInfo.AllowAllUsers)
		}
	}

	return registered, nil
}

func writeGatewayChannelDegraded(status gateway.RuntimeStatusWriter, platform, errText string) {
	if status == nil {
		return
	}
	_ = status.UpdateRuntimeStatus(context.Background(), gateway.RuntimeStatusUpdate{
		Platform:      platform,
		PlatformState: gateway.PlatformStateFailed,
		ErrorMessage:  errText,
	})
}

func missingSlackCredentials(cfg config.SlackCfg) []string {
	missing := []string{}
	if strings.TrimSpace(cfg.BotToken) == "" {
		missing = append(missing, "bot_token")
	}
	if strings.TrimSpace(cfg.AppToken) == "" {
		missing = append(missing, "app_token")
	}
	return missing
}
