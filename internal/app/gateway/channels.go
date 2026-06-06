// Package gateway provides core gateway runtime behavior. This file contains
// the channel factory types and registration logic extracted from
// cmd/gormes/gateway_channels.go.
package gateway

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
	runtimegateway "github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	channelsmodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/channels"
)

// ChannelFactory is the signature for a function that creates a runtimegateway.Channel
// from config and a logger.
type ChannelFactory func(config.Config, *slog.Logger) (runtimegateway.Channel, error)

// ChannelFactories holds one factory per supported platform. The gateway
// runtime uses these to create channel instances on startup.
type ChannelFactories struct {
	Telegram ChannelFactory
	Discord  ChannelFactory
	Slack    ChannelFactory
	Teams    ChannelFactory
	Yuanbao  ChannelFactory
	Navivox  ChannelFactory
	SimpleX  ChannelFactory
}

// DefaultChannelFactories returns the standard set of channel factories
// for the production gateway runtime.
func DefaultChannelFactories() ChannelFactories {
	return ChannelFactories{
		Telegram: func(cfg config.Config, log *slog.Logger) (runtimegateway.Channel, error) {
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
				DynamicCommands:     nil, // set by caller via RunOptions
				TokenLockDir:        config.GatewayLockDir(),
				ModelPickerResolver: runtimegateway.NewModelPickerResolver(&runtimegateway.SessionModelOverride{}),
			}, tc, log), nil
		},
		Discord: gormescli.NewDiscordGatewayChannel,
		Slack:   gormescli.NewSlackGatewayChannel,
		Teams: func(config.Config, *slog.Logger) (runtimegateway.Channel, error) {
			return nil, errors.New("teams_live_transport_unavailable: live Bot Framework binding is not implemented; Teams is fakeable only in this slice")
		},
		Yuanbao: func(config.Config, *slog.Logger) (runtimegateway.Channel, error) {
			return nil, errors.New("yuanbao_runtime_unavailable: live Yuanbao transport is not implemented; the runtime slice binds fake clients only")
		},
		Navivox: channelsmodule.NewNavivoxGatewayChannel,
		SimpleX: gormescli.NewSimpleXGatewayChannel,
	}
}

// TelegramDynamicCommands returns the dynamic Telegram commands from skills.
func TelegramDynamicCommands(ctx context.Context, cfg config.Config) []runtimegateway.PlatformCommand {
	runtime := skills.NewRuntime(cfg.SkillsRoot(), cfg.Skills.MaxDocumentBytes, cfg.Skills.SelectionCap, cfg.SkillsUsageLogPath())
	skillCommands, _, err := runtime.SkillSlashCommands(ctx, skills.RuntimeOptions{})
	if err != nil || len(skillCommands) == 0 {
		return nil
	}
	commands := make([]runtimegateway.PlatformCommand, 0, len(skillCommands))
	for _, cmd := range skillCommands {
		commands = append(commands, runtimegateway.PlatformCommand{
			Name:        strings.TrimPrefix(cmd.Command, "/"),
			Description: cmd.Description,
		})
	}
	return commands
}

// RegisterConfiguredGatewayChannels registers every configured channel
// (Telegram, Discord, Slack, Teams, Yuanbao, Navivox, SimpleX) against
// the given gateway Manager. Returns the number of channels successfully
// registered.
func RegisterConfiguredGatewayChannels(
	mgr *runtimegateway.Manager,
	cfg config.Config,
	allowedChats map[string]string,
	allowDiscovery map[string]bool,
	factories ChannelFactories,
	status runtimegateway.RuntimeStatusWriter,
	log *slog.Logger,
) (int, error) {
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
			WriteGatewayChannelDegraded(status, "telegram", fmt.Sprintf("telegram account %s: missing bot_token", accountID))
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
			WriteGatewayChannelDegraded(status, "telegram", fmt.Sprintf("telegram account %s: startup failed: %s", accountID, err.Error()))
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
			WriteGatewayChannelDegraded(status, "discord", fmt.Sprintf("discord account %s: missing token", accountID))
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
			WriteGatewayChannelDegraded(status, "discord", fmt.Sprintf("discord account %s: startup failed: %s", accountID, err.Error()))
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

		if missing := MissingSlackCredentials(cfg.Slack); len(missing) > 0 {
			errText := "slack: missing " + strings.Join(missing, ",")
			WriteGatewayChannelDegraded(status, "slack", errText)
			log.Warn("gateway: slack channel disabled by missing credentials", "missing", strings.Join(missing, ","))
		} else {
			if factories.Slack == nil {
				return registered, fmt.Errorf("register slack: missing channel factory")
			}
			ch, err := factories.Slack(cfg, log)
			if err != nil {
				errText := "slack: startup failed: " + err.Error()
				WriteGatewayChannelDegraded(status, "slack", errText)
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
			WriteGatewayChannelDegraded(status, "slack", fmt.Sprintf("slack account %s: missing bot_token or app_token", accountID))
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
			WriteGatewayChannelDegraded(status, "slack", fmt.Sprintf("slack account %s: startup failed: %s", accountID, err.Error()))
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
			WriteGatewayChannelDegraded(status, "teams", errText)
			log.Warn("gateway: teams channel disabled by missing credentials", "missing", strings.Join(missing, ","))
		} else {
			if factories.Teams == nil {
				return registered, fmt.Errorf("register teams: missing channel factory")
			}
			ch, err := factories.Teams(cfg, log)
			if err != nil {
				errText := "teams: startup failed: " + err.Error()
				WriteGatewayChannelDegraded(status, "teams", errText)
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
			WriteGatewayChannelDegraded(status, "yuanbao", errText)
			log.Warn("gateway: yuanbao channel disabled by missing credentials", "missing", strings.Join(missing, ","))
			return registered, nil
		}
		if factories.Yuanbao == nil {
			return registered, fmt.Errorf("register yuanbao: missing channel factory")
		}
		ch, err := factories.Yuanbao(cfg, log)
		if err != nil {
			errText := "yuanbao: startup failed: " + err.Error()
			WriteGatewayChannelDegraded(status, "yuanbao", errText)
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
			WriteGatewayChannelDegraded(status, channelsmodule.NavivoxPlatformName, "navivox: startup failed: "+err.Error())
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
			WriteGatewayChannelDegraded(status, simplexInfo.Platform, errText)
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

// WriteGatewayChannelDegraded records a degraded state for a channel platform
// in the gateway runtime status store.
func WriteGatewayChannelDegraded(status runtimegateway.RuntimeStatusWriter, platform, errText string) {
	if status == nil {
		return
	}
	_ = status.UpdateRuntimeStatus(context.Background(), runtimegateway.RuntimeStatusUpdate{
		Platform:      platform,
		PlatformState: runtimegateway.PlatformStateFailed,
		ErrorMessage:  errText,
	})
}

// MissingSlackCredentials returns a list of missing Slack credential field names.
func MissingSlackCredentials(cfg config.SlackCfg) []string {
	missing := []string{}
	if strings.TrimSpace(cfg.BotToken) == "" {
		missing = append(missing, "bot_token")
	}
	if strings.TrimSpace(cfg.AppToken) == "" {
		missing = append(missing, "app_token")
	}
	return missing
}
