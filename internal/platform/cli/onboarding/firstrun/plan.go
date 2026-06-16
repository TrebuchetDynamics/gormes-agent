package firstrun

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
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

type ActionID string

const (
	ActionQuick           ActionID = "quick"
	ActionFull            ActionID = "full"
	ActionMigrateHermes   ActionID = "migrate_hermes"
	ActionMigrateOpenClaw ActionID = "migrate_openclaw"
)

type StepID string

const (
	StepProvider StepID = "provider"
	StepAuth     StepID = "auth"
	StepModel    StepID = "model"
	StepChannel  StepID = "channel"
)

type PlanInput struct {
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

type Action struct {
	ID        ActionID
	Label     string
	Available bool
	Detail    string
	Command   string
}

type Step struct {
	ID      StepID
	Label   string
	Detail  string
	Command string
}

type Plan struct {
	Ready         bool
	PromptAllowed bool
	Target        SetupTargetID
	TargetLabel   string
	DefaultTarget SetupTargetID
	Targets       []SetupTargetOption
	Actions       []Action
	DefaultAction ActionID
	MissingSteps  []Step
	NextCommand   string
	Summary       string
}

func BuildPlan(input PlanInput) Plan {
	target := normalizeSetupTarget(input.Target)
	targets := buildTargets(input, target)
	selected := findTarget(targets, target)
	coreReady := textvalue.IsNonBlank(input.Endpoint) && textvalue.IsNonBlank(input.Model) && input.APIKeyPresent
	channelConfigured := !selected.Channel || channelConfigured(input.Channels, target)

	plan := Plan{
		PromptAllowed: input.Interactive,
		Target:        target,
		TargetLabel:   selected.Label,
		DefaultTarget: SetupTargetNavivox,
		Targets:       targets,
		Actions:       buildActions(input, target),
		DefaultAction: ActionQuick,
	}

	setupCommand := selected.SetupCommand
	if setupCommand == "" {
		setupCommand = defaultSetupCommand(target)
	}
	if !textvalue.IsNonBlank(input.Endpoint) {
		plan.MissingSteps = append(plan.MissingSteps, Step{
			ID:      StepProvider,
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
		plan.MissingSteps = append(plan.MissingSteps, Step{
			ID:      StepAuth,
			Label:   "Authentication",
			Detail:  "provider credential is not configured",
			Command: command,
		})
	}
	if !textvalue.IsNonBlank(input.Model) {
		plan.MissingSteps = append(plan.MissingSteps, Step{
			ID:      StepModel,
			Label:   "Model",
			Detail:  "model is not configured",
			Command: "gormes setup model",
		})
	}
	if selected.Channel && !channelConfigured {
		plan.MissingSteps = append(plan.MissingSteps, Step{
			ID:      StepChannel,
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

func (p Plan) Step(id StepID) (Step, bool) {
	for _, step := range p.MissingSteps {
		if step.ID == id {
			return step, true
		}
	}
	return Step{}, false
}

func DefaultChannels(overrides map[SetupTargetID]ChannelState) []ChannelState {
	channels := []ChannelState{
		defaultChannel(SetupTargetTelegram),
		defaultChannel(SetupTargetWhatsApp),
		defaultChannel(SetupTargetDiscord),
		defaultChannel(SetupTargetSlack),
		defaultChannel(SetupTargetNavivox),
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

func buildTargets(input PlanInput, selected SetupTargetID) []SetupTargetOption {
	coreConfigured := textvalue.IsNonBlank(input.Endpoint) && textvalue.IsNonBlank(input.Model) && input.APIKeyPresent
	targets := []SetupTargetOption{{
		ID:             SetupTargetTerminal,
		Label:          "Terminal",
		Configured:     coreConfigured,
		Detail:         "terminal chat",
		SetupCommand:   defaultSetupCommand(SetupTargetTerminal),
		HandoffCommand: defaultHandoffCommand(SetupTargetTerminal),
	}}

	channelByTarget := map[SetupTargetID]ChannelState{}
	for _, channel := range DefaultChannels(nil) {
		channelByTarget[channel.Target] = channel
	}
	for _, channel := range input.Channels {
		normalized := normalizeSetupTarget(channel.Target)
		if !isChannelTarget(normalized) {
			continue
		}
		channel.Target = normalized
		channelByTarget[normalized] = mergeChannelState(defaultChannel(normalized), channel)
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

	if findTarget(targets, selected).ID == "" {
		return buildTargets(input, SetupTargetTerminal)
	}
	return targets
}

func buildActions(input PlanInput, target SetupTargetID) []Action {
	actions := []Action{
		{
			ID:        ActionQuick,
			Label:     "Quick setup",
			Available: true,
			Detail:    "configure the selected target with the shortest setup path",
			Command:   quickSetupCommand(target),
		},
		{
			ID:        ActionFull,
			Label:     "Full setup",
			Available: true,
			Detail:    "walk through provider, model, credentials, and channel setup",
			Command:   "gormes setup --target " + string(target),
		},
	}
	if path := strings.TrimSpace(input.HermesSourcePath); path != "" {
		actions = append(actions, Action{
			ID:        ActionMigrateHermes,
			Label:     "Migrate Hermes",
			Available: true,
			Detail:    "import settings from an existing Hermes install",
			Command:   fmt.Sprintf("gormes migrate hermes --dry-run --source %s", shellQuoteArg(path)),
		})
	}
	if path := strings.TrimSpace(input.OpenClawSourcePath); path != "" {
		actions = append(actions, Action{
			ID:        ActionMigrateOpenClaw,
			Label:     "Migrate OpenClaw",
			Available: true,
			Detail:    "import settings from an existing OpenClaw install",
			Command:   fmt.Sprintf("gormes migrate openclaw --dry-run --source %s", shellQuoteArg(path)),
		})
	}
	return actions
}

func shellQuoteArg(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func channelConfigured(channels []ChannelState, target SetupTargetID) bool {
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

func findTarget(targets []SetupTargetOption, id SetupTargetID) SetupTargetOption {
	for _, target := range targets {
		if target.ID == id {
			return target
		}
	}
	return SetupTargetOption{}
}

func defaultChannel(id SetupTargetID) ChannelState {
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
	switch SetupTargetID(textvalue.LowerTrim(string(id))) {
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
