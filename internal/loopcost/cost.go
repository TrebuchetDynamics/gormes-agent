// Package loopcost provides cost-tracking for the autonomous builder loop.
// It parses opencode JSONL output files and computes daily/weekly/monthly spend rollups.
package loopcost

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// UnknownCost is the sentinel value returned when cost data is unavailable.
// A cost of -1 means "unknown", not "zero".
const UnknownCost = -1.0

// RunCost represents the cost of a single builder-loop iteration.
type RunCost struct {
	RunID        string    `json:"run_id"`
	Timestamp    time.Time `json:"timestamp"`
	CostUSD      float64   `json:"cost_usd"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	Backend      string    `json:"backend"`
}

// IsUnknownCost reports whether this run's cost is unknown (sentinel value).
func (rc RunCost) IsUnknownCost() bool {
	return rc.CostUSD <= -1.0
}

// CostSummary aggregates cost and token counts across runs.
type CostSummary struct {
	TotalCost        float64 `json:"total_cost"`
	TotalInputTokens int     `json:"total_input_tokens"`
	TotalOutputTokens int    `json:"total_output_tokens"`
	RunCount         int     `json:"run_count"`
	UnknownRuns      int     `json:"unknown_runs"`
}

// Rollup represents a cost aggregation over a time window.
type Rollup struct {
	Period  string      `json:"period"`
	Start   time.Time   `json:"start"`
	End     time.Time   `json:"end"`
	Summary CostSummary `json:"summary"`
}

// opencodeLine is the subset of opencode JSONL we parse.
type opencodeLine struct {
	Usage     *opencodeUsage `json:"usage"`
	Timestamp string         `json:"timestamp"`
}

type opencodeUsage struct {
	Cost         interface{} `json:"cost"` // number or string
	InputTokens  int         `json:"input_tokens"`
	OutputTokens int         `json:"output_tokens"`
}

// ParseRunCost parses a single opencode JSONL run file and returns a RunCost.
func ParseRunCost(r io.Reader, runID string) (RunCost, error) {
	scanner := bufio.NewScanner(r)
	var lastCost *RunCost

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ol opencodeLine
		if err := json.Unmarshal([]byte(line), &ol); err != nil {
			continue // skip non-JSON lines
		}
		rc := RunCost{RunID: runID, Backend: "opencode"}
		if ol.Usage != nil {
			rc.InputTokens = ol.Usage.InputTokens
			rc.OutputTokens = ol.Usage.OutputTokens
			c, hasCost := parseCost(ol.Usage.Cost)
			if hasCost {
				rc.CostUSD = c
			} else {
				rc.CostUSD = UnknownCost
			}
		} else {
			rc.CostUSD = UnknownCost
		}
		if ol.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339, ol.Timestamp); err == nil {
				rc.Timestamp = t
			}
		}
		lastCost = &rc
	}

	if err := scanner.Err(); err != nil {
		return RunCost{}, fmt.Errorf("reading run %q: %w", runID, err)
	}
	if lastCost == nil {
		return RunCost{}, fmt.Errorf("run %q: no valid JSONL lines found", runID)
	}
	if lastCost.CostUSD == UnknownCost {
		return RunCost{}, fmt.Errorf("run %q: no cost data found in JSONL", runID)
	}
	return *lastCost, nil
}

// ParseJSONLCosts parses a multi-line JSONL stream and returns all RunCost entries.
func ParseJSONLCosts(r io.Reader) ([]RunCost, error) {
	var costs []RunCost
	scanner := bufio.NewScanner(r)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ol opencodeLine
		if err := json.Unmarshal([]byte(line), &ol); err != nil {
			continue // skip non-JSON lines
		}
		rc := RunCost{
			RunID:   fmt.Sprintf("line-%d", lineNum),
			Backend: "opencode",
		}
		if ol.Usage != nil {
			rc.InputTokens = ol.Usage.InputTokens
			rc.OutputTokens = ol.Usage.OutputTokens
			c, hasCost := parseCost(ol.Usage.Cost)
			if hasCost {
				rc.CostUSD = c
			} else {
				rc.CostUSD = UnknownCost
			}
		} else {
			rc.CostUSD = UnknownCost
		}
		if ol.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339, ol.Timestamp); err == nil {
				rc.Timestamp = t
			}
		}
		costs = append(costs, rc)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading JSONL: %w", err)
	}
	if len(costs) == 0 {
		return nil, errors.New("no valid JSONL lines found")
	}
	return costs, nil
}

// DailyRollup aggregates costs over the last 24 hours.
func DailyRollup(costs []RunCost, now time.Time) Rollup {
	return WindowRollup(costs, 24*time.Hour, now)
}

// WindowRollup aggregates costs over a given time window ending at now.
func WindowRollup(costs []RunCost, window time.Duration, now time.Time) Rollup {
	start := now.Add(-window)
	var summary CostSummary

	for _, rc := range costs {
		if rc.Timestamp.Before(start) || rc.Timestamp.After(now) {
			continue
		}
		summary.RunCount++
		if rc.IsUnknownCost() {
			summary.UnknownRuns++
			continue
		}
		summary.TotalCost += rc.CostUSD
		summary.TotalInputTokens += rc.InputTokens
		summary.TotalOutputTokens += rc.OutputTokens
	}

	period := "window"
	if window == 24*time.Hour {
		period = "daily"
	} else if window == 7*24*time.Hour {
		period = "weekly"
	} else if window == 30*24*time.Hour {
		period = "monthly"
	}

	return Rollup{
		Period:  period,
		Start:   start,
		End:     now,
		Summary: summary,
	}
}

// parseCost extracts a float64 cost from either a JSON number or string.
func parseCost(v interface{}) (float64, bool) {
	if v == nil {
		return 0, false
	}
	switch val := v.(type) {
	case float64:
		return val, true
	case string:
		if val == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}
