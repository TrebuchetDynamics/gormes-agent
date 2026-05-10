package hermes

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	backgroundReviewMemoryToolset = "memory"
	backgroundReviewSkillsToolset = "skills"

	// BackgroundReviewSkillWriteOrigin is the provenance marker used by
	// skill_manage writes that originate in the self-improvement review fork.
	BackgroundReviewSkillWriteOrigin = "background_review"
)

// BackgroundReviewRuntime is the parent turn runtime snapshot copied into the
// review fork. It deliberately stores already-resolved values so the review
// worker does not re-read env vars, live config, or credential files.
type BackgroundReviewRuntime struct {
	Model              string
	Provider           string
	APIMode            string
	BaseURL            string
	APIKey             string
	CredentialPool     any
	Platform           string
	ParentSessionID    string
	MemoryStore        any
	MemoryEnabled      bool
	UserProfileEnabled bool
}

// BackgroundReviewMessage is the compact transcript shape needed by the review
// fork and summary extraction. It mirrors the Hermes tool-message fields that
// define whether an action is new or inherited stale history.
type BackgroundReviewMessage struct {
	Role       string
	Content    string
	ToolCallID string
}

// BackgroundReviewFork is the fakeable worker construction packet passed to a
// runner. Production wiring can map this into a real worker later; tests keep
// it hermetic.
type BackgroundReviewFork struct {
	Runtime                 BackgroundReviewRuntime
	Prompt                  string
	ConversationHistory     []BackgroundReviewMessage
	EnabledToolsets         []string
	ToolsetPolicy           BackgroundReviewToolsetPolicy
	SkillManagerWriteOrigin string
	SuppressStatusOutput    bool
}

// BackgroundReviewRequest describes one review pass.
type BackgroundReviewRequest struct {
	Runtime      BackgroundReviewRuntime
	Messages     []BackgroundReviewMessage
	ReviewMemory bool
	ReviewSkills bool
	Runner       BackgroundReviewRunner

	SummaryCallback        func(string)
	ApprovalSlot           BackgroundReviewApprovalSlot
	ShutdownMemoryProvider func()
	Close                  func()
}

// BackgroundReviewResult is the observable outcome of one review pass.
type BackgroundReviewResult struct {
	Actions []string
	Summary string
}

// BackgroundReviewRunner is the only execution dependency for the review
// worker. Implementations must be fakeable and must not mutate the parent
// transcript supplied in the fork packet.
type BackgroundReviewRunner interface {
	RunBackgroundReview(context.Context, BackgroundReviewFork) ([]BackgroundReviewMessage, error)
}

// BackgroundReviewRunnerFunc adapts a function into BackgroundReviewRunner.
type BackgroundReviewRunnerFunc func(context.Context, BackgroundReviewFork) ([]BackgroundReviewMessage, error)

func (f BackgroundReviewRunnerFunc) RunBackgroundReview(ctx context.Context, fork BackgroundReviewFork) ([]BackgroundReviewMessage, error) {
	return f(ctx, fork)
}

// BackgroundReviewApprovalCallback is the noninteractive dangerous-action
// callback installed for the duration of the review fork.
type BackgroundReviewApprovalCallback func(command, description string) string

// BackgroundReviewApprovalSlot is a small adapter over the caller's approval
// callback storage. The fork installs an auto-deny callback and clears it when
// cleanup runs.
type BackgroundReviewApprovalSlot struct {
	Set   func(BackgroundReviewApprovalCallback)
	Clear func()
}

// BackgroundReviewToolsetStatus is the status of a review toolset decision.
type BackgroundReviewToolsetStatus string

const (
	BackgroundReviewToolsetAllowed     BackgroundReviewToolsetStatus = "background_review_toolset_allowed"
	BackgroundReviewToolsetRestricted  BackgroundReviewToolsetStatus = "background_review_toolset_restricted"
	BackgroundReviewToolsetUnavailable BackgroundReviewToolsetStatus = "background_review_toolset_unavailable"
)

// BackgroundReviewToolsetEvidence is prompt-free telemetry for one toolset
// request.
type BackgroundReviewToolsetEvidence struct {
	Status          BackgroundReviewToolsetStatus
	AllowedToolsets []string
	DeniedToolset   string
	Reason          string
}

// BackgroundReviewToolsetPolicy is the local review-worker allowlist. It
// intentionally mirrors internal/subagent's background-review policy without
// importing that package, because subagent already imports hermes for runner
// contracts.
type BackgroundReviewToolsetPolicy struct {
	allowed map[string]struct{}
}

// DefaultBackgroundReviewToolsetPolicy returns the Hermes-compatible allowlist:
// memory + skills, and nothing with shell/browser/provider side effects.
func DefaultBackgroundReviewToolsetPolicy() BackgroundReviewToolsetPolicy {
	return BackgroundReviewToolsetPolicy{allowed: map[string]struct{}{
		backgroundReviewMemoryToolset: {},
		backgroundReviewSkillsToolset: {},
	}}
}

func (p BackgroundReviewToolsetPolicy) AllowedToolsets() []string {
	out := make([]string, 0, len(p.allowed))
	for name := range p.allowed {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func (p BackgroundReviewToolsetPolicy) CheckToolset(name string) (BackgroundReviewToolsetEvidence, bool) {
	allowed := p.AllowedToolsets()
	normalised := strings.ToLower(strings.TrimSpace(name))
	if normalised == "" {
		return BackgroundReviewToolsetEvidence{
			Status:          BackgroundReviewToolsetUnavailable,
			AllowedToolsets: allowed,
			Reason:          "background review toolset request carried no resolvable name",
		}, false
	}
	if _, ok := p.allowed[normalised]; ok {
		return BackgroundReviewToolsetEvidence{
			Status:          BackgroundReviewToolsetAllowed,
			AllowedToolsets: allowed,
		}, true
	}
	return BackgroundReviewToolsetEvidence{
		Status:          BackgroundReviewToolsetRestricted,
		AllowedToolsets: allowed,
		DeniedToolset:   normalised,
		Reason:          fmt.Sprintf("background review workers only allow toolsets %s", strings.Join(allowed, ",")),
	}, false
}

// RunBackgroundReview executes one synchronous, fakeable review pass. Async
// scheduling is intentionally outside this function; this slice proves the fork
// construction, inherited runtime, summary, and cleanup contract.
func RunBackgroundReview(ctx context.Context, req BackgroundReviewRequest) (BackgroundReviewResult, error) {
	fork := BackgroundReviewFork{
		Runtime:                 req.Runtime,
		Prompt:                  backgroundReviewPrompt(req.ReviewMemory, req.ReviewSkills),
		ConversationHistory:     cloneBackgroundReviewMessages(req.Messages),
		EnabledToolsets:         []string{backgroundReviewMemoryToolset, backgroundReviewSkillsToolset},
		ToolsetPolicy:           DefaultBackgroundReviewToolsetPolicy(),
		SkillManagerWriteOrigin: BackgroundReviewSkillWriteOrigin,
		SuppressStatusOutput:    true,
	}

	if req.ApprovalSlot.Set != nil {
		req.ApprovalSlot.Set(func(string, string) string { return "deny" })
		if req.ApprovalSlot.Clear != nil {
			defer req.ApprovalSlot.Clear()
		}
	}
	if req.Close != nil {
		defer req.Close()
	}
	if req.ShutdownMemoryProvider != nil {
		defer req.ShutdownMemoryProvider()
	}

	if req.Runner == nil {
		return BackgroundReviewResult{}, nil
	}
	reviewMessages, err := req.Runner.RunBackgroundReview(ctx, fork)
	if err != nil {
		return BackgroundReviewResult{}, err
	}
	actions := SummarizeBackgroundReviewActions(reviewMessages, req.Messages)
	result := BackgroundReviewResult{Actions: actions}
	if len(actions) > 0 {
		summary := "Self-improvement review: " + strings.Join(dedupeStrings(actions), "; ")
		result.Summary = summary
		if req.SummaryCallback != nil {
			req.SummaryCallback(summary)
		}
	}
	return result, nil
}

// SummarizeBackgroundReviewActions extracts newly-successful memory/skill tool
// actions from a review transcript, skipping tool messages already present in
// the inherited parent history.
func SummarizeBackgroundReviewActions(reviewMessages, priorSnapshot []BackgroundReviewMessage) []string {
	existingToolCallIDs := map[string]struct{}{}
	existingToolContents := map[string]struct{}{}
	for _, prior := range priorSnapshot {
		if prior.Role != "tool" {
			continue
		}
		if strings.TrimSpace(prior.ToolCallID) != "" {
			existingToolCallIDs[prior.ToolCallID] = struct{}{}
			continue
		}
		if prior.Content != "" {
			existingToolContents[prior.Content] = struct{}{}
		}
	}

	var actions []string
	for _, msg := range reviewMessages {
		if msg.Role != "tool" {
			continue
		}
		if msg.ToolCallID != "" {
			if _, stale := existingToolCallIDs[msg.ToolCallID]; stale {
				continue
			}
		} else if msg.Content != "" {
			if _, stale := existingToolContents[msg.Content]; stale {
				continue
			}
		}
		var data struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
			Target  string `json:"target"`
		}
		if err := json.Unmarshal([]byte(msg.Content), &data); err != nil || !data.Success {
			continue
		}
		messageLower := strings.ToLower(data.Message)
		switch {
		case strings.Contains(messageLower, "created"):
			actions = append(actions, data.Message)
		case strings.Contains(messageLower, "updated"):
			actions = append(actions, data.Message)
		case strings.Contains(messageLower, "added") || (data.Target != "" && strings.Contains(messageLower, "add")):
			actions = append(actions, backgroundReviewTargetLabel(data.Target)+" updated")
		case strings.Contains(data.Message, "Entry added"):
			actions = append(actions, backgroundReviewTargetLabel(data.Target)+" updated")
		case strings.Contains(messageLower, "removed") || strings.Contains(messageLower, "replaced"):
			actions = append(actions, backgroundReviewTargetLabel(data.Target)+" updated")
		}
	}
	return actions
}

func backgroundReviewPrompt(reviewMemory, reviewSkills bool) string {
	switch {
	case reviewMemory && reviewSkills:
		return "Review the conversation above and update two things:\n\n" +
			"Memory: save durable user facts, preferences, and expectations with the memory tool when they are worth remembering.\n\n" +
			backgroundReviewSkillPrompt()
	case reviewMemory:
		return "Review the conversation for durable memory updates. If nothing stands out, say 'Nothing to save.'"
	default:
		return backgroundReviewSkillPrompt()
	}
}

func backgroundReviewSkillPrompt() string {
	return "Skills: review the conversation for reusable skill improvements. " +
		"Use skills_list and skill_view to survey currently relevant skills before deciding. " +
		"Prefer updating or generalizing an existing class-level skill over creating a narrow one-off skill. " +
		"Create a new class-level skill only when no existing skill covers the task class.\n\n" +
		"Do NOT capture as skills:\n" +
		"- Environment-dependent failures such as missing binaries, fresh-install errors, post-migration path mismatches, command not found, unconfigured credentials, or uninstalled packages.\n" +
		"- Negative claims about tools or features such as browser tools do not work, a tool is broken, or cannot use a feature. These become durable self-imposed constraints after the setup state changes.\n" +
		"- Session-specific transient errors that resolved before the conversation ended. If retrying worked, capture the retry pattern rather than the original failure.\n" +
		"- One-off task narratives such as summarize today's market or analyze this PR.\n\n" +
		"If a tool failed because of setup state, capture the fix: the install command, config step, or environment variable to set under an existing setup or troubleshooting skill. " +
		"Never save this tool does not work as a standalone constraint.\n\n" +
		"If nothing stands out, say 'Nothing to save.'"
}

func backgroundReviewTargetLabel(target string) string {
	switch strings.ToLower(strings.TrimSpace(target)) {
	case "memory":
		return "Memory"
	case "user":
		return "User profile"
	default:
		return strings.TrimSpace(target)
	}
}

func cloneBackgroundReviewMessages(in []BackgroundReviewMessage) []BackgroundReviewMessage {
	if len(in) == 0 {
		return nil
	}
	out := make([]BackgroundReviewMessage, len(in))
	copy(out, in)
	return out
}

func dedupeStrings(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, value := range in {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
