package legacy

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/discord/legacy/formatting"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func formatStream(f kernel.RenderFrame) string { return formatting.FormatStream(f) }

func formatFinal(f kernel.RenderFrame) string { return formatting.FormatFinal(f) }

func formatError(f kernel.RenderFrame) string { return formatting.FormatError(f) }

func truncateDiscord(s string) string { return formatting.TruncateDiscord(s) }

func stripSelfMention(text, selfID string) string { return formatting.StripSelfMention(text, selfID) }

func hasSelfMention(text, selfID string) bool { return formatting.HasSelfMention(text, selfID) }
