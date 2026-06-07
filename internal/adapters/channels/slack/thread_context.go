package slack

import slackthread "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/slack/threadctx"

// ThreadMessage is the fakeable subset of Slack conversations.replies data
// needed to derive thread parent context without wiring a live API fetcher.
type ThreadMessage = slackthread.ThreadMessage

type ThreadContext = slackthread.ThreadContext

type ThreadContextCache = slackthread.ThreadContextCache

func newThreadContextCache(selfUserID string) *ThreadContextCache {
	return slackthread.NewCache(selfUserID)
}
