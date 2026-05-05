package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
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
	for _, phaseKey := range sortedStatusKeys(prog.Phases) {
		phase := prog.Phases[phaseKey]
		for _, subphaseKey := range sortedStatusKeys(phase.Subphases) {
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

func sortedStatusKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
