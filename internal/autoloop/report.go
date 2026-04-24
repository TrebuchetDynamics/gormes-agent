package autoloop

import (
	"fmt"
	"regexp"
	"strings"
)

type FinalReport struct {
	Commit     string
	Acceptance []string
}

var commitLinePattern = regexp.MustCompile(`^Commit:\s*([0-9a-fA-F]+)\s*$`)

func ParseFinalReport(text string) (FinalReport, error) {
	var report FinalReport
	var inAcceptance bool

	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if match := commitLinePattern.FindStringSubmatch(trimmed); match != nil {
			report.Commit = match[1]
		}

		if strings.EqualFold(trimmed, "Acceptance:") {
			inAcceptance = true
			continue
		}
		if !inAcceptance {
			continue
		}

		if strings.HasPrefix(trimmed, "-") {
			item := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			if item != "" {
				report.Acceptance = append(report.Acceptance, item)
			}
			continue
		}
		if trimmed != "" {
			inAcceptance = false
		}
	}

	if report.Commit == "" {
		return FinalReport{}, fmt.Errorf("final report missing commit")
	}
	if len(report.Acceptance) == 0 {
		return FinalReport{}, fmt.Errorf("final report missing acceptance")
	}

	hasRed, hasGreen := acceptanceEvidence(report.Acceptance)
	if !hasRed {
		return FinalReport{}, fmt.Errorf("final report missing RED evidence with exit 1")
	}
	if !hasGreen {
		return FinalReport{}, fmt.Errorf("final report missing GREEN evidence with exit 0")
	}

	return report, nil
}

func acceptanceEvidence(acceptance []string) (bool, bool) {
	var hasRed bool
	var hasGreen bool

	for _, item := range acceptance {
		lower := strings.ToLower(strings.TrimSpace(item))
		if strings.HasPrefix(lower, "red:") && strings.Contains(lower, "exit 1") {
			hasRed = true
		}
		if strings.HasPrefix(lower, "green:") && strings.Contains(lower, "exit 0") {
			hasGreen = true
		}
	}

	return hasRed, hasGreen
}
