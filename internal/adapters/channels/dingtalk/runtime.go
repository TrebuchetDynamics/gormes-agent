package dingtalk

import (
	"context"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/dingtalk/runtimeplan"
)

type IngressMode = runtimeplan.IngressMode

const IngressModeStream IngressMode = runtimeplan.IngressModeStream

type ReplyMode = runtimeplan.ReplyMode

const ReplyModeSessionWebhook ReplyMode = runtimeplan.ReplyModeSessionWebhook

type RuntimeConfig = runtimeplan.RuntimeConfig

type RuntimePlan = runtimeplan.RuntimePlan

type IngressPlan = runtimeplan.IngressPlan

type ReplyPlan = runtimeplan.ReplyPlan

type ReplyRetryPolicy = runtimeplan.ReplyRetryPolicy

func DecideRuntime(cfg RuntimeConfig) (RuntimePlan, error) {
	return runtimeplan.DecideRuntime(cfg)
}

func DefaultReplyRetryPolicy() ReplyRetryPolicy {
	return runtimeplan.DefaultReplyRetryPolicy()
}

type SessionWebhooks = runtimeplan.SessionWebhooks

func NewSessionWebhooks() *SessionWebhooks {
	return runtimeplan.NewSessionWebhooks()
}

type ReplyClient = runtimeplan.ReplyClient

type ReplySender = runtimeplan.ReplySender

type ReplySenderOption = runtimeplan.ReplySenderOption

func WithReplySleep(fn func(context.Context, time.Duration) error) ReplySenderOption {
	return runtimeplan.WithReplySleep(fn)
}

func NewReplySender(client ReplyClient, retry ReplyRetryPolicy, opts ...ReplySenderOption) *ReplySender {
	return runtimeplan.NewReplySender(client, retry, opts...)
}
