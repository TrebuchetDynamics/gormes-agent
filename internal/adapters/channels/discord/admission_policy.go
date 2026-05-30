package discord

import "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/discord/admission"

const (
	DiscordAdmissionAllowed        = admission.DiscordAdmissionAllowed
	DiscordAdmissionOwnMessage     = admission.DiscordAdmissionOwnMessage
	DiscordAdmissionBotDenied      = admission.DiscordAdmissionBotDenied
	DiscordAdmissionAllowedChannel = admission.DiscordAdmissionAllowedChannel
	DiscordAdmissionIgnoredChannel = admission.DiscordAdmissionIgnoredChannel
	DiscordAdmissionMentionMissing = admission.DiscordAdmissionMentionMissing
)

type AdmissionPolicy = admission.AdmissionPolicy
type AdmissionContext = admission.AdmissionContext
type AdmissionResult = admission.AdmissionResult

func EvaluateAdmission(policy AdmissionPolicy, ctx AdmissionContext) AdmissionResult {
	return admission.EvaluateAdmission(policy, ctx)
}
