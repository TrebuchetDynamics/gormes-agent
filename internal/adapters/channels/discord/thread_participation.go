package discord

import "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/discord/threads"

const defaultDiscordThreadParticipationLimit = threads.DefaultDiscordThreadParticipationLimit

type ThreadParticipationOptions = threads.ThreadParticipationOptions
type ThreadParticipationEvidence = threads.ThreadParticipationEvidence
type ThreadParticipationTracker = threads.ThreadParticipationTracker

func NewThreadParticipationTracker(opts ThreadParticipationOptions) *ThreadParticipationTracker {
	return threads.NewThreadParticipationTracker(opts)
}
