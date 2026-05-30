package interactions

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/bwmarrin/discordgo"
)

// NormalizeCommandName returns a Discord slash-command-compatible name.
func NormalizeCommandName(name string) string {
	name = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(name)), "/")
	name = strings.ReplaceAll(name, "_", "-")
	var b strings.Builder
	for _, r := range name {
		if unicode.IsLower(r) || unicode.IsDigit(r) || r == '-' {
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 32 {
		out = strings.Trim(out[:32], "-")
	}
	return out
}

// BoundedDescription returns a non-empty Discord command description <= 100 bytes.
func BoundedDescription(desc, fallback string) string {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		desc = fallback
	}
	desc = strings.Join(strings.Fields(desc), " ")
	if len(desc) > 100 {
		desc = strings.TrimSpace(desc[:100])
	}
	if desc == "" {
		return "Run command"
	}
	return desc
}

// OptionString returns a trimmed Discord slash-command option string.
func OptionString(data discordgo.ApplicationCommandInteractionData, name string) string {
	if opt := data.GetOption(name); opt != nil {
		return OptionValueString(opt)
	}
	return ""
}

// OptionValueString returns a trimmed string representation for supported option values.
func OptionValueString(opt *discordgo.ApplicationCommandInteractionDataOption) string {
	if opt == nil || opt.Value == nil {
		return ""
	}
	switch opt.Type {
	case discordgo.ApplicationCommandOptionString:
		if value, ok := opt.Value.(string); ok {
			return strings.TrimSpace(value)
		}
	case discordgo.ApplicationCommandOptionInteger:
		switch value := opt.Value.(type) {
		case float64:
			return fmt.Sprintf("%.0f", value)
		case int:
			return fmt.Sprintf("%d", value)
		case int64:
			return fmt.Sprintf("%d", value)
		}
	}
	return strings.TrimSpace(fmt.Sprint(opt.Value))
}

// OptionInt returns an integer Discord slash-command option or fallback.
func OptionInt(data discordgo.ApplicationCommandInteractionData, name string, fallback int) int {
	if opt := data.GetOption(name); opt != nil && opt.Value != nil {
		switch value := opt.Value.(type) {
		case float64:
			return int(value)
		case int:
			return value
		case int64:
			return int(value)
		}
	}
	return fallback
}

// CommandPayloadBytes marshals commands for soft-limit sizing and tests.
func CommandPayloadBytes(commands []*discordgo.ApplicationCommand) []byte {
	raw, _ := json.Marshal(commands)
	return raw
}
