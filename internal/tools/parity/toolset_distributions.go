package parity

import (
	"errors"
	"fmt"
	"math/rand"
)

// ErrUnknownToolsetDistribution is returned when a sampler caller asks for a
// Hermes distribution name that is not present in the frozen manifest below.
var ErrUnknownToolsetDistribution = errors.New("tools: unknown toolset distribution")

// ToolsetDistributionIssueKind identifies sampler degraded-mode evidence.
type ToolsetDistributionIssueKind string

const (
	ToolsetDistributionIssueUnknownDistribution   ToolsetDistributionIssueKind = "unknown_distribution"
	ToolsetDistributionIssueInvalidToolsetSkipped ToolsetDistributionIssueKind = "invalid_toolset_skipped"
	ToolsetDistributionIssueFallbackSelected      ToolsetDistributionIssueKind = "fallback_selected"
)

// ToolsetDistributionEntry is one Hermes toolset probability entry.
type ToolsetDistributionEntry struct {
	Toolset     string
	Probability int
}

// ToolsetDistribution captures one named Hermes distribution from
// toolset_distributions.py. Entries remain ordered to preserve the donor's
// deterministic sampling order.
type ToolsetDistribution struct {
	Name        string
	Description string
	Toolsets    []ToolsetDistributionEntry
}

// ToolsetDistributionIssue records validation/fallback evidence from sampling.
type ToolsetDistributionIssue struct {
	Kind         ToolsetDistributionIssueKind
	Distribution string
	Toolset      string
	Detail       string
}

// ToolsetDistributionSampleOptions controls hermetic sampling.
type ToolsetDistributionSampleOptions struct {
	// Random returns a random float in [0,1). Tests can inject a deterministic
	// sequence. When nil, math/rand.Float64 is used to match Hermes' random.random.
	Random func() float64
	// ValidateToolset reports whether a referenced toolset is usable. When nil,
	// validation uses the embedded upstream tool parity manifest.
	ValidateToolset func(string) bool
}

// ToolsetDistributionSample is the deterministic sampler result plus evidence.
type ToolsetDistributionSample struct {
	Distribution string
	Toolsets     []string
	Issues       []ToolsetDistributionIssue
}

var hermesToolsetDistributions = []ToolsetDistribution{
	{
		Name:        "default",
		Description: "All available tools, all the time",
		Toolsets: []ToolsetDistributionEntry{
			{Toolset: "web", Probability: 100},
			{Toolset: "vision", Probability: 100},
			{Toolset: "image_gen", Probability: 100},
			{Toolset: "terminal", Probability: 100},
			{Toolset: "file", Probability: 100},
			{Toolset: "moa", Probability: 100},
			{Toolset: "browser", Probability: 100},
		},
	},
	{
		Name:        "image_gen",
		Description: "Heavy focus on image generation with vision and web support",
		Toolsets: []ToolsetDistributionEntry{
			{Toolset: "image_gen", Probability: 90},
			{Toolset: "vision", Probability: 90},
			{Toolset: "web", Probability: 55},
			{Toolset: "terminal", Probability: 45},
			{Toolset: "moa", Probability: 10},
		},
	},
	{
		Name:        "research",
		Description: "Web research with vision analysis and reasoning",
		Toolsets: []ToolsetDistributionEntry{
			{Toolset: "web", Probability: 90},
			{Toolset: "browser", Probability: 70},
			{Toolset: "vision", Probability: 50},
			{Toolset: "moa", Probability: 40},
			{Toolset: "terminal", Probability: 10},
		},
	},
	{
		Name:        "science",
		Description: "Scientific research with web, terminal, file, and browser capabilities",
		Toolsets: []ToolsetDistributionEntry{
			{Toolset: "web", Probability: 94},
			{Toolset: "terminal", Probability: 94},
			{Toolset: "file", Probability: 94},
			{Toolset: "vision", Probability: 65},
			{Toolset: "browser", Probability: 50},
			{Toolset: "image_gen", Probability: 15},
			{Toolset: "moa", Probability: 10},
		},
	},
	{
		Name:        "development",
		Description: "Terminal, file tools, and reasoning with occasional web lookup",
		Toolsets: []ToolsetDistributionEntry{
			{Toolset: "terminal", Probability: 80},
			{Toolset: "file", Probability: 80},
			{Toolset: "moa", Probability: 60},
			{Toolset: "web", Probability: 30},
			{Toolset: "vision", Probability: 10},
		},
	},
	{
		Name:        "safe",
		Description: "All tools except terminal for safety",
		Toolsets: []ToolsetDistributionEntry{
			{Toolset: "web", Probability: 80},
			{Toolset: "browser", Probability: 70},
			{Toolset: "vision", Probability: 60},
			{Toolset: "image_gen", Probability: 60},
			{Toolset: "moa", Probability: 50},
		},
	},
	{
		Name:        "balanced",
		Description: "Equal probability of all toolsets",
		Toolsets: []ToolsetDistributionEntry{
			{Toolset: "web", Probability: 50},
			{Toolset: "vision", Probability: 50},
			{Toolset: "image_gen", Probability: 50},
			{Toolset: "terminal", Probability: 50},
			{Toolset: "file", Probability: 50},
			{Toolset: "moa", Probability: 50},
			{Toolset: "browser", Probability: 50},
		},
	},
	{
		Name:        "minimal",
		Description: "Only web tools for basic research",
		Toolsets:    []ToolsetDistributionEntry{{Toolset: "web", Probability: 100}},
	},
	{
		Name:        "terminal_only",
		Description: "Terminal and file tools for code execution tasks",
		Toolsets: []ToolsetDistributionEntry{
			{Toolset: "terminal", Probability: 100},
			{Toolset: "file", Probability: 100},
		},
	},
	{
		Name:        "terminal_web",
		Description: "Terminal and file tools with web search for documentation lookup",
		Toolsets: []ToolsetDistributionEntry{
			{Toolset: "terminal", Probability: 100},
			{Toolset: "file", Probability: 100},
			{Toolset: "web", Probability: 100},
		},
	},
	{
		Name:        "creative",
		Description: "Image generation and vision analysis focus",
		Toolsets: []ToolsetDistributionEntry{
			{Toolset: "image_gen", Probability: 90},
			{Toolset: "vision", Probability: 90},
			{Toolset: "web", Probability: 30},
		},
	},
	{
		Name:        "reasoning",
		Description: "Heavy mixture of agents usage with minimal other tools",
		Toolsets: []ToolsetDistributionEntry{
			{Toolset: "moa", Probability: 90},
			{Toolset: "web", Probability: 30},
			{Toolset: "terminal", Probability: 20},
		},
	},
	{
		Name:        "browser_use",
		Description: "Full browser-based web interaction with search, vision, and page control",
		Toolsets: []ToolsetDistributionEntry{
			{Toolset: "browser", Probability: 100},
			{Toolset: "web", Probability: 80},
			{Toolset: "vision", Probability: 70},
		},
	},
	{
		Name:        "browser_only",
		Description: "Only browser automation tools for pure web interaction tasks",
		Toolsets:    []ToolsetDistributionEntry{{Toolset: "browser", Probability: 100}},
	},
	{
		Name:        "browser_tasks",
		Description: "Browser-focused distribution (browser toolset includes web_search for finding URLs since Google blocks direct browser searches)",
		Toolsets: []ToolsetDistributionEntry{
			{Toolset: "browser", Probability: 97},
			{Toolset: "vision", Probability: 12},
			{Toolset: "terminal", Probability: 15},
		},
	},
	{
		Name:        "terminal_tasks",
		Description: "Terminal-focused distribution with high terminal/file availability, occasional other tools",
		Toolsets: []ToolsetDistributionEntry{
			{Toolset: "terminal", Probability: 97},
			{Toolset: "file", Probability: 97},
			{Toolset: "web", Probability: 97},
			{Toolset: "browser", Probability: 75},
			{Toolset: "vision", Probability: 50},
			{Toolset: "image_gen", Probability: 10},
		},
	},
	{
		Name:        "mixed_tasks",
		Description: "Mixed distribution with high browser, terminal, and file availability for complex tasks",
		Toolsets: []ToolsetDistributionEntry{
			{Toolset: "browser", Probability: 92},
			{Toolset: "terminal", Probability: 92},
			{Toolset: "file", Probability: 92},
			{Toolset: "web", Probability: 35},
			{Toolset: "vision", Probability: 15},
			{Toolset: "image_gen", Probability: 15},
		},
	},
}

// ListToolsetDistributions returns the ordered Hermes distribution manifest.
func ListToolsetDistributions() []ToolsetDistribution {
	out := make([]ToolsetDistribution, 0, len(hermesToolsetDistributions))
	for _, distribution := range hermesToolsetDistributions {
		out = append(out, cloneToolsetDistribution(distribution))
	}
	return out
}

// GetToolsetDistribution returns one distribution by name.
func GetToolsetDistribution(name string) (ToolsetDistribution, bool) {
	for _, distribution := range hermesToolsetDistributions {
		if distribution.Name == name {
			return cloneToolsetDistribution(distribution), true
		}
	}
	return ToolsetDistribution{}, false
}

// SampleToolsetsFromDistribution samples each toolset independently using the
// Hermes probability contract. If every valid roll misses, it selects the
// highest-probability valid toolset so the result is non-empty when possible.
func SampleToolsetsFromDistribution(name string, opts ToolsetDistributionSampleOptions) (ToolsetDistributionSample, error) {
	distribution, ok := GetToolsetDistribution(name)
	if !ok {
		return ToolsetDistributionSample{
			Distribution: name,
			Issues: []ToolsetDistributionIssue{{
				Kind:         ToolsetDistributionIssueUnknownDistribution,
				Distribution: name,
				Toolset:      name,
				Detail:       fmt.Sprintf("unknown toolset distribution %q", name),
			}},
		}, fmt.Errorf("%w: %s", ErrUnknownToolsetDistribution, name)
	}

	validator, err := resolveToolsetDistributionValidator(opts.ValidateToolset)
	if err != nil {
		return ToolsetDistributionSample{Distribution: name}, err
	}
	random := opts.Random
	if random == nil {
		random = rand.Float64
	}

	sample := ToolsetDistributionSample{Distribution: name}
	validEntries := make([]ToolsetDistributionEntry, 0, len(distribution.Toolsets))
	for _, entry := range distribution.Toolsets {
		if !validator(entry.Toolset) {
			sample.Issues = append(sample.Issues, ToolsetDistributionIssue{
				Kind:         ToolsetDistributionIssueInvalidToolsetSkipped,
				Distribution: name,
				Toolset:      entry.Toolset,
				Detail:       fmt.Sprintf("toolset %q is not present in the active toolset catalog", entry.Toolset),
			})
			continue
		}
		validEntries = append(validEntries, entry)
		if random()*100 < float64(entry.Probability) {
			sample.Toolsets = append(sample.Toolsets, entry.Toolset)
		}
	}

	if len(sample.Toolsets) == 0 && len(validEntries) > 0 {
		fallback := highestProbabilityToolset(validEntries)
		sample.Toolsets = append(sample.Toolsets, fallback.Toolset)
		sample.Issues = append(sample.Issues, ToolsetDistributionIssue{
			Kind:         ToolsetDistributionIssueFallbackSelected,
			Distribution: name,
			Toolset:      fallback.Toolset,
			Detail:       fmt.Sprintf("all distribution rolls missed; selected highest-probability valid toolset %q", fallback.Toolset),
		})
	}

	return sample, nil
}

func cloneToolsetDistribution(distribution ToolsetDistribution) ToolsetDistribution {
	clone := distribution
	clone.Toolsets = append([]ToolsetDistributionEntry(nil), distribution.Toolsets...)
	return clone
}

func resolveToolsetDistributionValidator(injected func(string) bool) (func(string) bool, error) {
	if injected != nil {
		return injected, nil
	}
	manifest, err := LoadUpstreamToolParityManifest()
	if err != nil {
		return nil, err
	}
	valid := make(map[string]bool, len(manifest.Toolsets))
	for _, row := range manifest.Toolsets {
		valid[row.Name] = true
	}
	return func(name string) bool { return valid[name] }, nil
}

func highestProbabilityToolset(entries []ToolsetDistributionEntry) ToolsetDistributionEntry {
	highest := entries[0]
	for _, entry := range entries[1:] {
		if entry.Probability > highest.Probability {
			highest = entry
		}
	}
	return highest
}
