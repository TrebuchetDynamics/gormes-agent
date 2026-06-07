package inventory

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/fidelity"
	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/repoctl/inventory/markdown"
)

const (
	hermesContractInventoryJSONRel = "webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.json"
	hermesContractInventoryMDRel   = "webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.md"
)

type HermesContractInventoryOptions struct {
	Root             string
	ProgressPath     string
	HermesSrc        string
	SourcePairsPath  string
	JSONPath         string
	MarkdownPath     string
	CurrentHermesSHA string
	Strict           bool
	Now              func() time.Time
}

type HermesContractInventoryResult struct {
	Report       fidelity.Report
	JSONPath     string
	MarkdownPath string
}

func WriteHermesContractInventory(opts HermesContractInventoryOptions) (HermesContractInventoryResult, error) {
	report, err := BuildHermesContractInventory(opts)
	if err != nil {
		return HermesContractInventoryResult{}, err
	}
	root := hermesContractInventoryRoot(opts.Root)
	jsonPath := opts.JSONPath
	if jsonPath == "" {
		jsonPath = filepath.Join(root, hermesContractInventoryJSONRel)
	}
	mdPath := opts.MarkdownPath
	if mdPath == "" {
		mdPath = filepath.Join(root, hermesContractInventoryMDRel)
	}
	if err := os.MkdirAll(filepath.Dir(jsonPath), 0o755); err != nil {
		return HermesContractInventoryResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(mdPath), 0o755); err != nil {
		return HermesContractInventoryResult{}, err
	}

	report = preserveHermesContractInventoryTimestamp(report, jsonPath, mdPath)

	jsonReport, err := marshalHermesContractInventoryJSON(report)
	if err != nil {
		return HermesContractInventoryResult{}, err
	}
	if err := os.WriteFile(jsonPath, jsonReport, 0o644); err != nil {
		return HermesContractInventoryResult{}, err
	}
	if err := os.WriteFile(mdPath, []byte(RenderHermesContractInventoryMarkdown(report)), 0o644); err != nil {
		return HermesContractInventoryResult{}, err
	}
	return HermesContractInventoryResult{Report: report, JSONPath: jsonPath, MarkdownPath: mdPath}, nil
}

func BuildHermesContractInventory(opts HermesContractInventoryOptions) (fidelity.Report, error) {
	root := hermesContractInventoryRoot(opts.Root)
	return fidelity.GenerateHermesReport(context.Background(), fidelity.Options{
		RepoRoot:        root,
		ProgressPath:    opts.ProgressPath,
		HermesPath:      opts.HermesSrc,
		SourcePairsPath: opts.SourcePairsPath,
		HermesSHA:       opts.CurrentHermesSHA,
		Strict:          opts.Strict,
		Now:             opts.Now,
	})
}

func preserveHermesContractInventoryTimestamp(report fidelity.Report, jsonPath, mdPath string) fidelity.Report {
	existingJSON, err := os.ReadFile(jsonPath)
	if err != nil {
		return report
	}
	existingMD, err := os.ReadFile(mdPath)
	if err != nil {
		return report
	}
	var existing fidelity.Report
	if err := json.Unmarshal(existingJSON, &existing); err != nil {
		return report
	}
	if strings.TrimSpace(existing.GeneratedAt) == "" {
		return report
	}
	candidate := report
	candidate.GeneratedAt = existing.GeneratedAt
	candidateJSON, err := marshalHermesContractInventoryJSON(candidate)
	if err != nil {
		return report
	}
	candidateMD := []byte(RenderHermesContractInventoryMarkdown(candidate))
	if bytes.Equal(existingJSON, candidateJSON) && bytes.Equal(existingMD, candidateMD) {
		return candidate
	}
	return report
}

func marshalHermesContractInventoryJSON(report fidelity.Report) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func RenderHermesContractInventoryMarkdown(report fidelity.Report) string {
	return markdown.Render(report)
}

func hermesContractInventoryRoot(root string) string {
	if strings.TrimSpace(root) == "" {
		return "."
	}
	return root
}
