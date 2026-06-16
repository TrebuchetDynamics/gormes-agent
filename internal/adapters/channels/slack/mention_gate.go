package slack

import slackmention "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/slack/mention"

const SlackMentionPolicyUnavailable = slackmention.SlackMentionPolicyUnavailable

type MentionPolicyConfig = slackmention.MentionPolicyConfig

type MentionPolicy = slackmention.MentionPolicy

type MentionPolicyEvidence = slackmention.MentionPolicyEvidence

type MentionGateInput = slackmention.MentionGateInput

type MentionGateDecision = slackmention.MentionGateDecision

func ResolveMentionPolicy(cfg MentionPolicyConfig) MentionPolicy {
	return slackmention.ResolveMentionPolicy(cfg)
}

func EvaluateMentionGate(policy MentionPolicy, input MentionGateInput) MentionGateDecision {
	return slackmention.EvaluateMentionGate(policy, input)
}

func slackMentionGateIsDM(channelID, chatType string) bool {
	return slackmention.IsDM(channelID, chatType)
}
