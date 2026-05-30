package statusreport

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

const DefaultStatusProgressPath = "webpages/docs/content/building-gormes/architecture_plan/progress.json"

type StatusReportOptions struct {
	ProgressPath string
}

type statusBlocker struct {
	Phase    string
	Subphase string
	Row      string
	Record   tools.BlockerRecord
}

// StatusBlocker is the exported, JSON-tagged shape of one active
// blocker. Returned by CollectStatusBlockers for `gormes status --json`.
// Embeds tools.BlockerRecord so all of its existing JSON fields (title,
// type, status, evidence, owner, …) flatten into the same object.
type StatusBlocker struct {
	Phase    string `json:"phase"`
	Subphase string `json:"subphase"`
	Row      string `json:"row"`
	tools.BlockerRecord
}

// CollectStatusBlockers loads progress from opts.ProgressPath and
// returns the active-blocker list — same data RenderStatusReport
// renders as text, but in machine-readable form for the JSON surface.
//
// A missing or unreadable progress file is reported as an empty list
// with the Load error returned to the caller; callers can distinguish
// "no blockers" (nil err, empty slice) from "couldn't read" (non-nil
// err) at the boundary.
func CollectStatusBlockers(opts StatusReportOptions) ([]StatusBlocker, error) {
	progressPath := strings.TrimSpace(opts.ProgressPath)
	if progressPath == "" {
		progressPath = DefaultStatusProgressPath
	}
	prog, err := progress.Load(progressPath)
	if err != nil {
		return nil, err
	}
	internal := collectStatusBlockers(prog)
	out := make([]StatusBlocker, len(internal))
	for i, b := range internal {
		out[i] = StatusBlocker{
			Phase:         b.Phase,
			Subphase:      b.Subphase,
			Row:           b.Row,
			BlockerRecord: tools.NormalizeBlockerRecord(b.Record),
		}
	}
	return out, nil
}

func RenderStatusReport(opts StatusReportOptions) (string, error) {
	progressPath := strings.TrimSpace(opts.ProgressPath)
	if progressPath == "" {
		progressPath = DefaultStatusProgressPath
	}

	var b strings.Builder
	b.WriteString("Gormes Status\n")

	prog, err := progress.Load(progressPath)
	if err != nil {
		fmt.Fprintf(&b, "blockers: unavailable status=progress_unavailable reason=%q\n", err)
		return b.String(), nil
	}

	blockers := collectStatusBlockers(prog)
	if len(blockers) == 0 {
		b.WriteString("blockers: none\n")
		return b.String(), nil
	}

	fmt.Fprintf(&b, "blockers: %d active\n", len(blockers))
	for _, blocker := range blockers {
		record := tools.NormalizeBlockerRecord(blocker.Record)
		fmt.Fprintf(&b, "- %s/%s %s type=%s owner=%s status=%s\n", blocker.Phase, blocker.Subphase, blocker.Row, record.Type, record.Owner, record.Status)
		if record.Blocker != "" {
			fmt.Fprintf(&b, "  blocker: %s\n", record.Blocker)
		}
		if record.Evidence != "" {
			fmt.Fprintf(&b, "  evidence: %s\n", record.Evidence)
		}
		if record.UnblocksWhen != "" {
			fmt.Fprintf(&b, "  unblocks when: %s\n", record.UnblocksWhen)
		}
		if record.Pivot != "" {
			fmt.Fprintf(&b, "  workaround/pivot: %s\n", record.Pivot)
		}
		if record.NextCheck != "" {
			fmt.Fprintf(&b, "  next check: %s\n", record.NextCheck)
		}
		if len(record.MissingFields) > 0 {
			fmt.Fprintf(&b, "  missing fields: %s\n", strings.Join(record.MissingFields, ","))
		}
	}
	return b.String(), nil
}

func collectStatusBlockers(prog *progress.Progress) []statusBlocker {
	var blockers []statusBlocker
	for _, phaseKey := range textvalue.SortedKeys(prog.Phases) {
		phase := prog.Phases[phaseKey]
		for _, subphaseKey := range textvalue.SortedKeys(phase.Subphases) {
			subphase := phase.Subphases[subphaseKey]
			for _, item := range subphase.Items {
				if item.Blocker == nil {
					continue
				}
				blockers = append(blockers, statusBlocker{
					Phase:    phaseKey,
					Subphase: subphaseKey,
					Row:      item.Name,
					Record:   statusBlockerRecord(item),
				})
			}
		}
	}
	return blockers
}

func statusBlockerRecord(item progress.Item) tools.BlockerRecord {
	blocker := item.Blocker
	title := strings.TrimSpace(blocker.Title)
	if title == "" {
		title = item.Name
	}
	return tools.BlockerRecord{
		Title:         title,
		Type:          tools.BlockerType(blocker.Type),
		Status:        tools.BlockerStatus(blocker.Status),
		RecordedAt:    blocker.RecordedAt,
		Blocker:       blocker.Blocker,
		Evidence:      blocker.Evidence,
		UnblocksWhen:  blocker.UnblocksWhen,
		Owner:         blocker.Owner,
		Pivot:         blocker.Pivot,
		NextCheck:     blocker.NextCheck,
		Degraded:      blocker.Degraded,
		MissingFields: append([]string(nil), blocker.MissingFields...),
	}
}

