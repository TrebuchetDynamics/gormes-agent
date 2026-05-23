package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
)

func TestRunNextWorkReportsPlanDecisionWhenQueueIsEmpty(t *testing.T) {
	root := t.TempDir()
	progressJSON := filepath.Join(root, "webpages", "docs", "content", "building-gormes", "architecture_plan", "progress.json")
	if err := os.MkdirAll(filepath.Dir(progressJSON), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	p := &progress.Progress{
		Meta: progress.Meta{Version: "2.0"},
		Phases: map[string]progress.Phase{
			"1": {Name: "P1", Deliverable: "d1", Subphases: map[string]progress.Subphase{
				"1.A": {Name: "A", Items: []progress.Item{
					{Name: "complete row", Status: progress.StatusComplete, Module: progress.ModuleProgress},
				}},
			}},
		},
	}
	if err := progress.SaveProgress(progressJSON, p); err != nil {
		t.Fatalf("seed progress: %v", err)
	}

	var stdout, stderr bytes.Buffer
	if err := run(&stdout, &stderr, []string{"--repo-root", root, "next-work"}); err != nil {
		t.Fatalf("run next-work: %v\nstderr=%s", err, stderr.String())
	}
	got := stdout.String()
	if !strings.Contains(got, "decision=plan") || !strings.Contains(got, "planner_action=repair one planned/draft row") {
		t.Fatalf("next-work should report a planner decision when the queue is empty:\n%s", got)
	}
}

func TestRunNextWorkRepoOnlyReportsPlanDecisionWhenCandidatesEscapeRepo(t *testing.T) {
	root := t.TempDir()
	progressJSON := filepath.Join(root, "webpages", "docs", "content", "building-gormes", "architecture_plan", "progress.json")
	if err := os.MkdirAll(filepath.Dir(progressJSON), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	p := &progress.Progress{
		Meta: progress.Meta{Version: "2.0"},
		Phases: map[string]progress.Phase{
			"9": {Name: "P9", Deliverable: "d9", Subphases: map[string]progress.Subphase{
				"9.F": {Name: "F", Items: []progress.Item{
					{Name: "navivox app row", Priority: "P1", Status: progress.StatusPlanned, Contract: "ui", ContractStatus: progress.ContractStatusFixtureReady, SliceSize: progress.SliceSizeSmall, NoTestRequiredReason: "fixture", Module: progress.ModuleNavivox, WriteScope: []string{"../navivox-app/app/lib/features/chat/"}},
				}},
			}},
		},
	}
	if err := progress.SaveProgress(progressJSON, p); err != nil {
		t.Fatalf("seed progress: %v", err)
	}

	var stdout, stderr bytes.Buffer
	if err := run(&stdout, &stderr, []string{"--repo-root", root, "next-work", "--repo-only"}); err != nil {
		t.Fatalf("run next-work --repo-only: %v\nstderr=%s", err, stderr.String())
	}
	got := stdout.String()
	if !strings.Contains(got, "decision=plan") || !strings.Contains(got, "scope=repo") || !strings.Contains(got, "write_scope stays under the repo root") {
		t.Fatalf("repo-only next-work should report a scoped planner decision:\n%s", got)
	}
}

func TestRunListModuleScopesOutput(t *testing.T) {
	root := t.TempDir()
	progressJSON := filepath.Join(root, "webpages", "docs", "content", "building-gormes", "architecture_plan", "progress.json")
	if err := os.MkdirAll(filepath.Dir(progressJSON), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	p := &progress.Progress{
		Meta: progress.Meta{Version: "2.0"},
		Phases: map[string]progress.Phase{
			"1": {Name: "P1", Deliverable: "d1", Subphases: map[string]progress.Subphase{
				"1.A": {Name: "A", Items: []progress.Item{
					{Name: "provider row", Priority: "P2", Status: progress.StatusPlanned, Module: progress.ModuleProviders},
					{Name: "tts row", Priority: "P2", Status: progress.StatusPlanned, Module: progress.ModuleTTS},
				}},
			}},
		},
	}
	if err := progress.SaveProgress(progressJSON, p); err != nil {
		t.Fatalf("seed progress: %v", err)
	}

	var stdout, stderr bytes.Buffer
	if err := run(&stdout, &stderr, []string{"--repo-root", root, "list", "--module", progress.ModuleProviders}); err != nil {
		t.Fatalf("run list: %v\nstderr=%s", err, stderr.String())
	}
	got := stdout.String()
	if !strings.Contains(got, "progress: module providers (1 row)") {
		t.Fatalf("output must name selected module scope:\n%s", got)
	}
	if !strings.Contains(got, "provider row") || strings.Contains(got, "tts row") {
		t.Fatalf("output must contain only selected module rows:\n%s", got)
	}
}
