package cli

import (
	"fmt"
	"strings"
)

type SetupTargetID string

const (
	SetupTargetTerminal SetupTargetID = "terminal"
	SetupTargetTelegram SetupTargetID = "telegram"
	SetupTargetWhatsApp SetupTargetID = "whatsapp"
	SetupTargetDiscord  SetupTargetID = "discord"
	SetupTargetSlack    SetupTargetID = "slack"
	SetupTargetNavivox  SetupTargetID = "navivox"
)

type FirstRunActionID string

const (
	FirstRunActionQuick           FirstRunActionID = "quick"
	FirstRunActionFull            FirstRunActionID = "full"
	FirstRunActionMigrateHermes   FirstRunActionID = "migrate_hermes"
	FirstRunActionMigrateOpenClaw FirstRunActionID = "migrate_openclaw"
)

type FirstRunStepID string

const (
	FirstRunStepProvider FirstRunStepID = "provider"
	FirstRunStepAuth     FirstRunStepID = "auth"
	FirstRunStepModel    FirstRunStepID = "model"
	FirstRunStepChannel  FirstRunStepID = "channel"
)

type FirstRunPlanInput struct {
	Interactive        bool
	Provider           string
	Endpoint           string
	Model              string
	APIKeyPresent      bool
	Target             SetupTargetID
	Channels           []ChannelState
	HermesSourcePath   string
	OpenClawSourcePath string
}

type ChannelState struct {
	Target         SetupTargetID
	Label          string
	Configured     bool
	Detail         string
	SetupCommand   string
	HandoffCommand string
}

type SetupTargetOption struct {
	ID             SetupTargetID
	Label          string
	Channel        bool
	Configured     bool
	Detail         string
	SetupCommand   string
	HandoffCommand string
}

type FirstRunAction struct {
	ID        FirstRunActionID
	Label     string
	Available bool
	Detail    string
	Command   string
}

type FirstRunStep struct {
	ID      FirstRunStepID
	Label   string
	Detail  string
	Command string
}

type FirstRunPlan struct {
	Ready         bool
	PromptAllowed bool
	Target        SetupTargetID
	TargetLabel   string
	DefaultTarget SetupTargetID
	Targets       []SetupTargetOption
	Actions       []FirstRunAction
	DefaultAction FirstRunActionID
	MissingSteps  []FirstRunStep
	NextCommand   string
	Summary       string
}

func BuildFirstRunPlan(input FirstRunPlanInput) FirstRunPlan {
	target := normalizeSetupTarget(input.Target)
	targets := buildFirstRunTargets(input, target)
	selected := findFirstRunTarget(targets, target)
	coreReady := strings.TrimSpace(input.Endpoint) != "" && strings.TrimSpace(input.Model) != "" && input.APIKeyPresent
	channelConfigured := !selected.Channel || firstRunChannelConfigured(input.Channels, target)

	plan := FirstRunPlan{
		PromptAllowed: input.Interactive,
		Target:        target,
		TargetLabel:   selected.Label,
		DefaultTarget: SetupTargetTerminal,
		Targets:       targets,
		Actions:       buildFirstRunActions(input, target),
		DefaultAction: FirstRunActionQuick,
	}

	setupCommand := selected.SetupCommand
	if setupCommand == "" {
		setupCommand = defaultSetupCommand(target)
	}
	if strings.TrimSpace(input.Endpoint) == "" {
		plan.MissingSteps = append(plan.MissingSteps, FirstRunStep{
			ID:      FirstRunStepProvider,
			Label:   "Provider",
			Detail:  "provider endpoint is not configured",
			Command: quickSetupCommand(target),
		})
	}
	if !input.APIKeyPresent {
		provider := strings.TrimSpace(input.Provider)
		command := "gormes setup provider"
		if provider != "" {
			command = "gormes auth add " + provider
		}
		plan.MissingSteps = append(plan.MissingSteps, FirstRunStep{
			ID:      FirstRunStepAuth,
			Label:   "Authentication",
			Detail:  "provider credential is not configured",
			Command: command,
		})
	}
	if strings.TrimSpace(input.Model) == "" {
		plan.MissingSteps = append(plan.MissingSteps, FirstRunStep{
			ID:      FirstRunStepModel,
			Label:   "Model",
			Detail:  "model is not configured",
			Command: "gormes setup model",
		})
	}
	if selected.Channel && !channelConfigured {
		plan.MissingSteps = append(plan.MissingSteps, FirstRunStep{
			ID:      FirstRunStepChannel,
			Label:   selected.Label,
			Detail:  selected.Label + " channel is not configured",
			Command: setupCommand,
		})
	}

	plan.Ready = coreReady && channelConfigured
	if plan.Ready {
		plan.NextCommand = selected.HandoffCommand
		if plan.NextCommand == "" {
			plan.NextCommand = defaultHandoffCommand(target)
		}
		if selected.Channel {
			plan.Summary = selected.Label + " is ready"
		} else {
			plan.Summary = "terminal chat is ready"
		}
		return plan
	}

	if len(plan.MissingSteps) > 0 {
		plan.NextCommand = plan.MissingSteps[0].Command
		plan.Summary = "not ready: " + plan.MissingSteps[0].Detail
	} else {
		plan.NextCommand = setupCommand
		plan.Summary = "not ready"
	}
	return plan
}

func (p FirstRunPlan) Step(id FirstRunStepID) (FirstRunStep, bool) {
	for _, step := range p.MissingSteps {
		if step.ID == id {
			return step, true
		}
	}
	return FirstRunStep{}, false
}

func DefaultFirstRunChannels(overrides map[SetupTargetID]ChannelState) []ChannelState {
	channels := []ChannelState{
		defaultFirstRunChannel(SetupTargetTelegram),
		defaultFirstRunChannel(SetupTargetWhatsApp),
		defaultFirstRunChannel(SetupTargetDiscord),
		defaultFirstRunChannel(SetupTargetSlack),
		defaultFirstRunChannel(SetupTargetNavivox),
	}

	for i := range channels {
		override, ok := overrides[channels[i].Target]
		if !ok {
			continue
		}
		channels[i] = mergeChannelState(channels[i], override)
	}
	return channels
}

func buildFirstRunTargets(input FirstRunPlanInput, selected SetupTargetID) []SetupTargetOption {
	coreConfigured := strings.TrimSpace(input.Endpoint) != "" && strings.TrimSpace(input.Model) != "" && input.APIKeyPresent
	targets := []SetupTargetOption{{
		ID:             SetupTargetTerminal,
		Label:          "Terminal",
		Configured:     coreConfigured,
		Detail:         "terminal chat",
		SetupCommand:   defaultSetupCommand(SetupTargetTerminal),
		HandoffCommand: defaultHandoffCommand(SetupTargetTerminal),
	}}

	channelByTarget := map[SetupTargetID]ChannelState{}
	for _, channel := range DefaultFirstRunChannels(nil) {
		channelByTarget[channel.Target] = channel
	}
	for _, channel := range input.Channels {
		normalized := normalizeSetupTarget(channel.Target)
		if !isChannelTarget(normalized) {
			continue
		}
		channel.Target = normalized
		channelByTarget[normalized] = mergeChannelState(defaultFirstRunChannel(normalized), channel)
	}

	for _, id := range []SetupTargetID{
		SetupTargetTelegram,
		SetupTargetWhatsApp,
		SetupTargetDiscord,
		SetupTargetSlack,
		SetupTargetNavivox,
	} {
		channel := channelByTarget[id]
		targets = append(targets, SetupTargetOption{
			ID:             id,
			Label:          channel.Label,
			Channel:        true,
			Configured:     coreConfigured && channel.Configured,
			Detail:         channel.Detail,
			SetupCommand:   channel.SetupCommand,
			HandoffCommand: channel.HandoffCommand,
		})
	}

	if findFirstRunTarget(targets, selected).ID == "" {
		return buildFirstRunTargets(input, SetupTargetTerminal)
	}
	return targets
}

func buildFirstRunActions(input FirstRunPlanInput, target SetupTargetID) []FirstRunAction {
	actions := []FirstRunAction{
		{
			ID:        FirstRunActionQuick,
			Label:     "Quick setup",
			Available: true,
			Detail:    "configure the selected target with the shortest setup path",
			Command:   quickSetupCommand(target),
		},
		{
			ID:        FirstRunActionFull,
			Label:     "Full setup",
			Available: true,
			Detail:    "walk through provider, model, credentials, and channel setup",
			Command:   "gormes setup --target " + string(target),
		},
	}
	if path := strings.TrimSpace(input.HermesSourcePath); path != "" {
		actions = append(actions, FirstRunAction{
			ID:        FirstRunActionMigrateHermes,
			Label:     "Migrate Hermes",
			Available: true,
			Detail:    "import settings from an existing Hermes install",
			Command:   fmt.Sprintf("gormes migrate hermes --dry-run --source %s", shellQuoteFirstRunArg(path)),
		})
	}
	if path := strings.TrimSpace(input.OpenClawSourcePath); path != "" {
		actions = append(actions, FirstRunAction{
			ID:        FirstRunActionMigrateOpenClaw,
			Label:     "Migrate OpenClaw",
			Available: true,
			Detail:    "import settings from an existing OpenClaw install",
			Command:   fmt.Sprintf("gormes migrate openclaw --dry-run --source %s", shellQuoteFirstRunArg(path)),
		})
	}
	return actions
}

func shellQuoteFirstRunArg(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func firstRunChannelConfigured(channels []ChannelState, target SetupTargetID) bool {
	target = normalizeSetupTarget(target)
	if !isChannelTarget(target) {
		return true
	}
	for _, channel := range channels {
		if normalizeSetupTarget(channel.Target) == target {
			return channel.Configured
		}
	}
	return false
}

func findFirstRunTarget(targets []SetupTargetOption, id SetupTargetID) SetupTargetOption {
	for _, target := range targets {
		if target.ID == id {
			return target
		}
	}
	return SetupTargetOption{}
}

func defaultFirstRunChannel(id SetupTargetID) ChannelState {
	id = normalizeSetupTarget(id)
	return ChannelState{
		Target:         id,
		Label:          setupTargetLabel(id),
		Detail:         setupTargetLabel(id) + " channel",
		SetupCommand:   defaultSetupCommand(id),
		HandoffCommand: defaultHandoffCommand(id),
	}
}

func mergeChannelState(base, override ChannelState) ChannelState {
	if normalized := normalizeSetupTarget(override.Target); isChannelTarget(normalized) {
		base.Target = normalized
	}
	if override.Label != "" {
		base.Label = override.Label
	}
	base.Configured = override.Configured
	if override.Detail != "" {
		base.Detail = override.Detail
	}
	if override.SetupCommand != "" {
		base.SetupCommand = override.SetupCommand
	}
	if override.HandoffCommand != "" {
		base.HandoffCommand = override.HandoffCommand
	}
	return base
}

func normalizeSetupTarget(id SetupTargetID) SetupTargetID {
	switch SetupTargetID(strings.ToLower(strings.TrimSpace(string(id)))) {
	case "", SetupTargetTerminal, "chat", "tui":
		return SetupTargetTerminal
	case SetupTargetTelegram:
		return SetupTargetTelegram
	case SetupTargetWhatsApp, "wa":
		return SetupTargetWhatsApp
	case SetupTargetDiscord:
		return SetupTargetDiscord
	case SetupTargetSlack:
		return SetupTargetSlack
	case SetupTargetNavivox:
		return SetupTargetNavivox
	default:
		return SetupTargetTerminal
	}
}

func isChannelTarget(id SetupTargetID) bool {
	switch id {
	case SetupTargetTelegram, SetupTargetWhatsApp, SetupTargetDiscord, SetupTargetSlack, SetupTargetNavivox:
		return true
	default:
		return false
	}
}

func setupTargetLabel(id SetupTargetID) string {
	switch id {
	case SetupTargetTelegram:
		return "Telegram"
	case SetupTargetWhatsApp:
		return "WhatsApp"
	case SetupTargetDiscord:
		return "Discord"
	case SetupTargetSlack:
		return "Slack"
	case SetupTargetNavivox:
		return "Navivox"
	default:
		return "Terminal"
	}
}

func defaultSetupCommand(id SetupTargetID) string {
	id = normalizeSetupTarget(id)
	if id == SetupTargetWhatsApp {
		return "gormes whatsapp --plan"
	}
	return quickSetupCommand(id)
}

func quickSetupCommand(id SetupTargetID) string {
	return "gormes setup --quick --target " + string(normalizeSetupTarget(id))
}

func defaultHandoffCommand(id SetupTargetID) string {
	if normalizeSetupTarget(id) == SetupTargetTerminal {
		return "gormes"
	}
	return "gormes gateway"
}
