package model

import "context"

type Scenario struct {
	Name                   string
	Prompt                 string
	ExpectedTools          []string
	ExpectedOutcome        string
	RequiredResponseTerms  []string
	ForbiddenResponseTerms []string
}

type Trace struct {
	Tools    []string
	Response string
}

type Result struct {
	VariantID       string
	Scenario        string
	TaskSuccess     bool
	ToolAccuracy    float64
	ResponseQuality float64
	ResponseScore   float64
	AggregateScore  float64
	Error           string
}

type Variant struct {
	ID     string
	Prompt string
	Score  float64
}

type VariantEvaluation struct {
	VariantID      string
	Prompt         string
	Results        []Result
	AggregateScore float64
}

type Runner func(context.Context, Variant, Scenario) (Trace, error)
