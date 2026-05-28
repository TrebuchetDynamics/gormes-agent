package builderloop

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeCandidatesSkipsCompleteAndSortsActiveFirst(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
				"3": {
					"subphases": {
					"3.E.6": {
							"items": [
								{"item_name": "planned candidate", "status": "planned", "contract": "planned contract", "contract_status": "draft", "no_test_required": "candidate ranking fixture"},
								{"item_name": "complete candidate", "status": "complete", "contract": "complete contract", "contract_status": "draft"},
								{"item_name": "active candidate", "status": "IN_PROGRESS", "contract": "active contract", "no_test_required": "candidate ranking fixture"}
							]
						}
					}
				}
			}
		}`)

	got, err := NormalizeCandidates(path, CandidateOptions{ActiveFirst: true})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	wantNames := []string{"active candidate", "planned candidate"}
	if gotNames := candidateNames(got); !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("candidate names = %#v, want %#v", gotNames, wantNames)
	}
}

func TestNormalizeCandidatesPriorityBoostWins(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
				"3": {
					"subphases": {
					"3.E.7": {
							"items": [
								{"name": "boosted planned candidate", "status": "planned", "contract": "boosted contract", "contract_status": "draft", "no_test_required": "candidate ranking fixture"}
							]
						},
					"3.E.6": {
							"items": [
								{"name": "normal active candidate", "status": "in_progress", "contract": "active contract", "no_test_required": "candidate ranking fixture"}
							]
						}
					}
				}
			}
		}`)

	got, err := NormalizeCandidates(path, CandidateOptions{
		ActiveFirst:   true,
		PriorityBoost: []string{" 3.e.7 "},
	})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	wantNames := []string{"boosted planned candidate", "normal active candidate"}
	if gotNames := candidateNames(got); !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("candidate names = %#v, want %#v", gotNames, wantNames)
	}
}

func TestNormalizeCandidatesPreservesNameFallbackMetadataAndSkipsBlockedUmbrella(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
			"3": {
				"subphases": {
					"3.E.6": {
						"items": [
							{
								"item_name": "ready candidate",
								"status": "planned",
								"priority": " P0 ",
								"contract": " Control plane handoff ",
								"contract_status": " DRAFT ",
								"slice_size": " Small ",
								"execution_owner": " Agent ",
								"trust_class": ["system"],
								"degraded_mode": " use legacy path ",
								"fixture": " progress_fixture.json ",
								"source_refs": [" docs/plan.md ", " "],
								"blocked_by": [],
								"unblocks": [" task 4 ", ""],
								"ready_when": [" tests pass ", " "],
								"not_ready_when": [" schema unsettled "],
								"acceptance": [" metadata retained "],
								"write_scope": [" internal/progress/builderloop "],
								"test_commands": [" go test ./internal/progress/builderloop "],
									"done_signal": ["provider transcript replay passes"],
									"note": " Keep human note casing. "
								},
								{
									"item_name": "blocked candidate",
									"status": "planned",
									"contract": "blocked contract",
									"contract_status": "draft",
									"blocked_by": ["task 2"]
								},
								{
									"item_name": "umbrella candidate",
									"status": "planned",
									"contract": "umbrella contract",
									"contract_status": "draft",
									"slice_size": "umbrella"
								}
							]
						}
					}
				}
		}
	}`)

	got, err := NormalizeCandidates(path, CandidateOptions{})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	want := []Candidate{
		{
			PhaseID:        "3",
			SubphaseID:     "3.E.6",
			ItemName:       "ready candidate",
			Status:         "planned",
			Priority:       "P0",
			Contract:       "Control plane handoff",
			ContractStatus: "draft",
			SliceSize:      "small",
			ExecutionOwner: "agent",
			TrustClass:     []string{"system"},
			DegradedMode:   "use legacy path",
			Fixture:        "progress_fixture.json",
			SourceRefs:     []string{"docs/plan.md"},
			BlockedBy:      nil,
			Unblocks:       []string{"task 4"},
			ReadyWhen:      []string{"tests pass"},
			NotReadyWhen:   []string{"schema unsettled"},
			Acceptance:     []string{"metadata retained"},
			WriteScope:     []string{"internal/progress/builderloop"},
			TestCommands:   []string{"go test ./internal/progress/builderloop"},
			DoneSignal:     []string{"provider transcript replay passes"},
			Note:           "Keep human note casing.",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeCandidates() = %#v, want %#v", got, want)
	}
	if got[0].SelectionReason() != "P0 handoff" {
		t.Fatalf("SelectionReason() = %q, want %q", got[0].SelectionReason(), "P0 handoff")
	}
	if !reflect.DeepEqual(got[0].TrustClass, []string{"system"}) {
		t.Fatalf("TrustClass = %#v, want %#v", got[0].TrustClass, []string{"system"})
	}
	if !reflect.DeepEqual(got[0].DoneSignal, []string{"provider transcript replay passes"}) {
		t.Fatalf("DoneSignal = %#v, want %#v", got[0].DoneSignal, []string{"provider transcript replay passes"})
	}
}

func TestNormalizeCandidatesUsesAgentQueueEligibility(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
			"3": {
				"subphases": {
					"3.E.6": {
						"items": [
							{"item_name": "p0 contract row", "status": "planned", "priority": "P0", "contract": "p0 handoff", "contract_status": "draft", "slice_size": "small", "no_test_required": "candidate eligibility fixture"},
							{"item_name": "active contract row", "status": "in_progress", "contract": "active handoff", "contract_status": "draft", "slice_size": "small", "no_test_required": "candidate eligibility fixture"},
							{"item_name": "draft contract row", "status": "planned", "contract": "draft handoff", "contract_status": "draft", "slice_size": "small", "no_test_required": "candidate eligibility fixture"},
							{"item_name": "missing contract row", "status": "planned", "priority": "P0", "contract_status": "draft", "slice_size": "small"},
							{"item_name": "generic contract row", "status": "planned", "contract": "generic handoff", "slice_size": "small"},
							{"item_name": "blocked contract row", "status": "planned", "priority": "P0", "contract": "blocked handoff", "contract_status": "draft", "slice_size": "small", "blocked_by": ["dependency"]},
							{"item_name": "umbrella contract row", "status": "planned", "priority": "P0", "contract": "umbrella handoff", "contract_status": "draft", "slice_size": "umbrella"},
							{"item_name": "complete contract row", "status": "complete", "priority": "P0", "contract": "complete handoff", "contract_status": "draft", "slice_size": "small"}
						]
					}
				}
			}
		}
	}`)

	got, err := NormalizeCandidates(path, CandidateOptions{ActiveFirst: true})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	wantNames := []string{"p0 contract row", "active contract row", "draft contract row"}
	if gotNames := candidateNames(got); !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("candidate names = %#v, want %#v", gotNames, wantNames)
	}
}

func TestNormalizeCandidatesIncludesRowsWhoseBlockersAreComplete(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
			"1": {
				"subphases": {
					"1.A": {
						"items": [
							{"name": "Foundation", "status": "complete"},
							{"name": "Dependent", "status": "planned", "contract": "dependent handoff", "contract_status": "fixture_ready", "slice_size": "small", "blocked_by": ["Foundation"], "test_commands": ["go test ./dependent"]},
							{"name": "Still blocked", "status": "planned", "contract": "blocked handoff", "contract_status": "fixture_ready", "slice_size": "small", "blocked_by": ["missing"], "test_commands": ["go test ./blocked"]}
						]
					}
				}
			}
		}
	}`)

	got, err := NormalizeCandidates(path, CandidateOptions{ActiveFirst: true})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}
	if gotNames := candidateNames(got); !reflect.DeepEqual(gotNames, []string{"Dependent"}) {
		t.Fatalf("candidate names = %#v, want Dependent", gotNames)
	}
}

func TestNormalizeCandidatesRequiresTestCommandsOrNoTestReason(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
			"3": {
				"subphases": {
					"3.E.6": {
						"items": [
							{"item_name": "missing test proof", "status": "planned", "contract": "draft handoff", "contract_status": "draft", "slice_size": "small"},
							{"item_name": "row-local test", "status": "planned", "contract": "tested handoff", "contract_status": "draft", "slice_size": "small", "test_commands": ["go test ./internal/goncho -count=1"]},
							{"item_name": "explicit testless row", "status": "planned", "contract": "docs handoff", "contract_status": "draft", "slice_size": "small", "no_test_required": "docs-only row; progress validate covers it"}
						]
					}
				}
			}
		}
	}`)

	got, err := NormalizeCandidates(path, CandidateOptions{ActiveFirst: true})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	wantNames := []string{"explicit testless row", "row-local test"}
	if gotNames := candidateNames(got); !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("candidate names = %#v, want %#v", gotNames, wantNames)
	}
	if got[0].NoTestRequired != "docs-only row; progress validate covers it" {
		t.Fatalf("NoTestRequired = %q, want explicit reason", got[0].NoTestRequired)
	}
}

func TestNormalizeCandidatesUsesExecutionBucketsWithItemNameRows(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
				"3": {
					"subphases": {
					"3.E.6": {
							"items": [
								{"item_name": "draft candidate", "status": "planned", "contract": "draft contract", "contract_status": "draft", "fixture": "draft.json", "no_test_required": "candidate bucket fixture"},
								{"item_name": "fixture candidate", "status": "planned", "contract": "fixture contract", "contract_status": "fixture_ready", "fixture": "ready.json", "no_test_required": "candidate bucket fixture"},
								{"item_name": "active candidate", "status": "in_progress", "contract": "active contract", "no_test_required": "candidate bucket fixture"},
								{"item_name": "p0 candidate", "status": "planned", "priority": "P0", "contract": "p0 contract", "contract_status": "draft", "no_test_required": "candidate bucket fixture"},
								{"item_name": "unblocking candidate", "status": "planned", "contract": "unblocking contract", "unblocks": ["task 4"], "no_test_required": "candidate bucket fixture"}
							]
						}
					}
				}
			}
		}`)

	got, err := NormalizeCandidates(path, CandidateOptions{ActiveFirst: true})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	wantNames := []string{"p0 candidate", "active candidate", "fixture candidate", "unblocking candidate", "draft candidate"}
	if gotNames := candidateNames(got); !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("candidate order = %#v, want %#v", gotNames, wantNames)
	}
	if got[4].SelectionReason() != "draft contract" {
		t.Fatalf("draft candidate SelectionReason() = %q, want %q", got[4].SelectionReason(), "draft contract")
	}
}

func TestNormalizeCandidatesActiveFirstFalseStillUsesPriorityTieAfterPriorityBoost(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
				"3": {
					"subphases": {
					"3.E.6": {
							"items": [
								{"item_name": "a draft candidate", "status": "planned", "contract": "draft contract", "contract_status": "draft", "no_test_required": "candidate ordering fixture"},
								{"item_name": "z p0 candidate", "status": "planned", "priority": "P0", "contract": "p0 contract", "contract_status": "draft", "no_test_required": "candidate ordering fixture"}
							]
						},
					"3.E.7": {
							"items": [
								{"item_name": "boosted planned candidate", "status": "planned", "contract": "boosted contract", "contract_status": "draft", "no_test_required": "candidate ordering fixture"}
							]
						}
					}
				}
			}
		}`)

	got, err := NormalizeCandidates(path, CandidateOptions{
		ActiveFirst:   false,
		PriorityBoost: []string{"3.E.7"},
	})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	wantNames := []string{"boosted planned candidate", "z p0 candidate", "a draft candidate"}
	if gotNames := candidateNames(got); !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("candidate order = %#v, want %#v", gotNames, wantNames)
	}
}

func TestNormalizeCandidatesHonorsMaxPhase(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
				"3": {
					"subphases": {
					"3.E": {
							"items": [
								{"name": "phase 3 candidate", "status": "planned", "contract": "phase 3 contract", "contract_status": "draft", "no_test_required": "max phase fixture"}
							]
						}
					}
				},
				"4": {
					"subphases": {
					"4.A": {
							"items": [
								{"name": "phase 4 candidate", "status": "in_progress", "contract": "phase 4 contract"}
							]
						}
					}
				}
			}
		}`)

	got, err := NormalizeCandidates(path, CandidateOptions{ActiveFirst: true, MaxPhase: 3})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	wantNames := []string{"phase 3 candidate"}
	if gotNames := candidateNames(got); !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("candidate names = %#v, want %#v", gotNames, wantNames)
	}
}

func TestNormalizeCandidatesSkipsPausedPhaseSevenByDefault(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
			"2": {
				"subphases": {
					"2.B.3": {
						"items": [
							{"name": "Slack parser wiring", "status": "planned", "priority": "P1", "contract": "Slack handoff", "contract_status": "draft", "no_test_required": "paused phase fixture"}
						]
					}
				}
			},
			"7": {
				"subphases": {
					"7.E": {
						"items": [
							{"name": "DingTalk real SDK binding", "status": "in_progress", "priority": "P0", "contract": "DingTalk handoff", "contract_status": "fixture_ready"}
						]
					}
				}
			}
		}
	}`)

	got, err := NormalizeCandidates(path, CandidateOptions{ActiveFirst: true, MaxPhase: 3})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	wantNames := []string{"Slack parser wiring"}
	if gotNames := candidateNames(got); !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("candidate names = %#v, want %#v", gotNames, wantNames)
	}
}

func TestNormalizeCandidatesUsesSubPhasesFallback(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
				"4": {
					"sub_phases": {
					"4.A.1": {
							"items": [
								{"item_name": "fallback candidate", "status": "planned", "contract": "fallback contract", "contract_status": "draft", "no_test_required": "sub_phases fixture"}
							]
						}
					}
				}
			}
		}`)

	got, err := NormalizeCandidates(path, CandidateOptions{})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	wantNames := []string{"fallback candidate"}
	if gotNames := candidateNames(got); !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("candidate names = %#v, want %#v", gotNames, wantNames)
	}
}

func TestNormalizeCandidatesPrefersSubphasesWhenBothKeysExist(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
				"4": {
					"subphases": {
					"4.A.1": {
							"items": [
								{"item_name": "preferred candidate", "status": "planned", "contract": "preferred contract", "contract_status": "draft", "no_test_required": "subphases fixture"}
							]
						}
					},
					"sub_phases": {
						"4.A.2": {
							"items": [
								{"item_name": "fallback candidate", "status": "planned", "contract": "fallback contract", "contract_status": "draft"}
							]
						}
					}
			}
		}
	}`)

	got, err := NormalizeCandidates(path, CandidateOptions{})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	wantNames := []string{"preferred candidate"}
	if gotNames := candidateNames(got); !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("candidate names = %#v, want %#v", gotNames, wantNames)
	}
}

func TestNormalizeCandidatesItemNameFallbacksAndUnknownStatus(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
				"5": {
					"subphases": {
					"5.B.1": {
							"items": [
								{"item_name": "item-name candidate", "name": "ignored name", "status": "planned", "contract": "item-name contract", "contract_status": "draft", "no_test_required": "name fallback fixture"},
								{"item_name": " ", "name": "name candidate", "title": "ignored title", "status": "planned", "contract": "name contract", "contract_status": "draft", "no_test_required": "name fallback fixture"},
								{"name": " ", "title": "title candidate", "id": "ignored id", "status": "planned", "contract": "title contract", "contract_status": "draft", "no_test_required": "name fallback fixture"},
								{"title": " ", "id": "id candidate", "contract": "id contract", "contract_status": "draft", "no_test_required": "name fallback fixture"},
								{"item_name": " "}
							]
						}
					}
				}
			}
	}`)

	got, err := NormalizeCandidates(path, CandidateOptions{})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	wantNames := []string{"id candidate", "item-name candidate", "name candidate", "title candidate"}
	if gotNames := candidateNames(got); !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("candidate names = %#v, want %#v", gotNames, wantNames)
	}
	if got[0].Status != "unknown" {
		t.Fatalf("first candidate Status = %q, want unknown", got[0].Status)
	}
}

func TestNormalizeCandidatesDeduplicatesByPhaseSubphaseAndItemName(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
				"6": {
					"subphases": {
					"6.C.1": {
							"items": [
								{"item_name": "duplicate candidate", "status": "planned", "contract": "planned contract", "contract_status": "draft", "no_test_required": "dedupe fixture"},
								{"item_name": "duplicate candidate", "status": "in_progress", "contract": "active contract", "no_test_required": "dedupe fixture"}
							]
						}
					}
				}
			}
		}`)

	got, err := NormalizeCandidates(path, CandidateOptions{ActiveFirst: true})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	wantNames := []string{"duplicate candidate"}
	if gotNames := candidateNames(got); !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("candidate names = %#v, want %#v", gotNames, wantNames)
	}
	if got[0].Status != "planned" {
		t.Fatalf("deduplicated candidate Status = %q, want planned", got[0].Status)
	}
}

func TestNormalizeCandidatesRealProgressSkipsCompletedBacklogSplitC5(t *testing.T) {
	path := filepath.Join("..", "..", "..", "docs", "content", "building-gormes", "architecture_plan", "progress.json")
	const c5Name = "Backlog split C5: single atomic operator-gated flip to the module-keyed split directory"

	got, err := NormalizeCandidates(path, CandidateOptions{ActiveFirst: true})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}
	for _, candidate := range got {
		if candidate.ItemName == c5Name {
			t.Fatalf("C5 appeared in default candidate queue; blocked_by=%v ready_when=%v", candidate.BlockedBy, candidate.ReadyWhen)
		}
	}

	blocked, err := NormalizeCandidates(path, CandidateOptions{ActiveFirst: true, IncludeBlocked: true})
	if err != nil {
		t.Fatalf("NormalizeCandidates(IncludeBlocked) error = %v", err)
	}
	for _, candidate := range blocked {
		if candidate.ItemName == c5Name {
			t.Fatalf("completed C5 appeared in IncludeBlocked candidate view; status=%s blocked_by=%v", candidate.Status, candidate.BlockedBy)
		}
	}
}

func candidateNames(candidates []Candidate) []string {
	var names []string
	for _, candidate := range candidates {
		names = append(names, candidate.ItemName)
	}
	return names
}

func TestNormalizeCandidatesReturnsMalformedJSONError(t *testing.T) {
	path := writeProgressJSON(t, `{"phases":`)

	_, err := NormalizeCandidates(path, CandidateOptions{})
	if err == nil {
		t.Fatal("NormalizeCandidates() error = nil, want error")
	}
}

func TestNormalizeCandidatesReturnsMissingFileError(t *testing.T) {
	_, err := NormalizeCandidates(filepath.Join(t.TempDir(), "missing.json"), CandidateOptions{})
	if err == nil {
		t.Fatal("NormalizeCandidates() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "missing.json") {
		t.Fatalf("NormalizeCandidates() error = %q, want missing filename", err)
	}
}

func TestNormalizeCandidatesPreservesExecutionMetadataAndSkipsBlockedUmbrella(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
			"7": {
				"subphases": {
					"7.A": {
						"items": [
							{
								"name": "ready candidate",
								"status": "planned",
								"priority": "P0",
								"contract": "Provider-neutral transcript contract",
								"contract_status": "fixture_ready",
								"slice_size": "medium",
								"execution_owner": "provider",
								"trust_class": ["system"],
								"degraded_mode": "provider status reports missing fixtures",
								"fixture": "internal/hermes/testdata/provider_transcripts",
								"source_refs": ["docs/content/upstream-hermes/source-study.md"],
								"ready_when": ["fixtures replay"],
								"not_ready_when": ["live provider call required"],
								"acceptance": ["go test ./internal/hermes passes"],
								"write_scope": ["internal/hermes/"],
								"test_commands": ["go test ./internal/hermes -count=1"],
								"done_signal": ["provider transcript replay passes"],
								"unblocks": ["Bedrock adapter"],
								"note": "Use captured transcript fixtures."
							},
							{
								"name": "blocked candidate",
								"status": "planned",
								"contract": "blocked",
								"slice_size": "small",
								"blocked_by": ["ready candidate"],
								"ready_when": ["ready candidate completes"]
							},
							{
								"name": "umbrella candidate",
								"status": "planned",
								"contract": "umbrella",
								"slice_size": "umbrella"
							}
						]
					}
				}
			}
		}
	}`)

	got, err := NormalizeCandidates(path, CandidateOptions{ActiveFirst: true, IncludePaused: true})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}
	if gotLen := len(got); gotLen != 1 {
		t.Fatalf("NormalizeCandidates() length = %d, want 1: %#v", gotLen, got)
	}

	candidate := got[0]
	if candidate.ItemName != "ready candidate" {
		t.Fatalf("ItemName = %q, want ready candidate", candidate.ItemName)
	}
	for _, want := range []string{
		candidate.Priority,
		candidate.Contract,
		candidate.ContractStatus,
		candidate.SliceSize,
		candidate.ExecutionOwner,
		candidate.DegradedMode,
		candidate.Fixture,
		candidate.Note,
	} {
		if strings.TrimSpace(want) == "" {
			t.Fatalf("candidate lost scalar metadata: %#v", candidate)
		}
	}
	if !reflect.DeepEqual(candidate.TrustClass, []string{"system"}) {
		t.Fatalf("TrustClass = %#v, want system", candidate.TrustClass)
	}
	if !reflect.DeepEqual(candidate.WriteScope, []string{"internal/hermes/"}) {
		t.Fatalf("WriteScope = %#v, want internal/hermes/", candidate.WriteScope)
	}
	if !reflect.DeepEqual(candidate.TestCommands, []string{"go test ./internal/hermes -count=1"}) {
		t.Fatalf("TestCommands = %#v, want provider test", candidate.TestCommands)
	}
	if got, want := candidate.SelectionReason(), "P0 handoff"; got != want {
		t.Fatalf("SelectionReason() = %q, want %q", got, want)
	}
}

func TestNormalizeCandidatesUsesExecutionBuckets(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
			"8": {
				"subphases": {
					"8.A": {
						"items": [
							{"name": "draft candidate", "status": "planned", "contract": "draft", "contract_status": "draft", "slice_size": "small", "no_test_required": "bucket fixture"},
							{"name": "fixture candidate", "status": "planned", "contract": "fixture", "contract_status": "fixture_ready", "slice_size": "small", "no_test_required": "bucket fixture"},
							{"name": "active candidate", "status": "in_progress", "contract": "active", "contract_status": "draft", "slice_size": "small", "no_test_required": "bucket fixture"},
							{"name": "p0 candidate", "status": "planned", "priority": "P0", "contract": "p0", "contract_status": "draft", "slice_size": "small", "no_test_required": "bucket fixture"},
							{"name": "unblocking candidate", "status": "planned", "contract": "unblock", "contract_status": "missing", "slice_size": "small", "unblocks": ["next row"], "no_test_required": "bucket fixture"}
						]
					}
				}
			}
		}
	}`)

	got, err := NormalizeCandidates(path, CandidateOptions{ActiveFirst: true})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	var names []string
	for _, candidate := range got {
		names = append(names, candidate.ItemName)
	}
	want := []string{
		"p0 candidate",
		"active candidate",
		"fixture candidate",
		"unblocking candidate",
		"draft candidate",
	}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("candidate order = %#v, want %#v", names, want)
	}
}

func TestNormalizeCandidatesUsesSubphasePriorityAsTieBreak(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
			"2": {
				"subphases": {
					"2.B.11": {
						"priority": "P3",
						"items": [
							{"name": "discord forum follow-up", "status": "planned", "contract": "discord forum contract", "contract_status": "draft", "no_test_required": "priority fixture"}
						]
					},
					"2.B.4": {
						"priority": "P1",
						"items": [
							{"name": "whatsapp runtime closeout", "status": "planned", "contract": "whatsapp runtime contract", "contract_status": "draft", "no_test_required": "priority fixture"}
						]
					}
				}
			}
		}
	}`)

	got, err := NormalizeCandidates(path, CandidateOptions{ActiveFirst: true})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	var names []string
	var priorities []string
	for _, candidate := range got {
		names = append(names, candidate.ItemName)
		priorities = append(priorities, candidate.Priority)
	}
	if want := []string{"whatsapp runtime closeout", "discord forum follow-up"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("candidate order = %#v, want %#v", names, want)
	}
	if want := []string{"P1", "P3"}; !reflect.DeepEqual(priorities, want) {
		t.Fatalf("candidate priorities = %#v, want %#v", priorities, want)
	}
}

func TestNormalizeCandidatesIncludesRowsWhenBlockersAreComplete(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
			"2": {
				"subphases": {
					"2.B.3": {
						"items": [
							{"name": "parser wiring", "status": "complete", "contract": "parser contract", "contract_status": "validated"},
							{"name": "channel shim", "status": "planned", "contract": "channel contract", "contract_status": "draft", "blocked_by": ["parser wiring"], "no_test_required": "blocker fixture"},
							{"name": "config registration", "status": "planned", "contract": "config contract", "contract_status": "draft", "blocked_by": ["channel shim"]}
						]
					}
				}
			}
		}
	}`)

	got, err := NormalizeCandidates(path, CandidateOptions{ActiveFirst: true})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("NormalizeCandidates() length = %d, want 1: %#v", len(got), got)
	}
	if got[0].ItemName != "channel shim" {
		t.Fatalf("selected candidate = %q, want channel shim", got[0].ItemName)
	}
	if !reflect.DeepEqual(got[0].BlockedBy, []string{"parser wiring"}) {
		t.Fatalf("BlockedBy = %#v, want parser wiring", got[0].BlockedBy)
	}
}

func TestNormalizeCandidatesSkipsRowsWithBlockerMetadataAndPivots(t *testing.T) {
	path := writeProgressJSON(t, `{
		"phases": {
			"5": {
				"subphases": {
					"5.N": {
						"items": [
							{
								"name": "blocked P0 row",
								"status": "planned",
								"priority": "P0",
								"contract": "blocked contract",
								"contract_status": "draft",
								"no_test_required": "blocker metadata fixture",
								"blocker": {
									"type": "infra",
									"status": "blocker_active",
									"blocker": "gateway lock",
									"evidence": "sessions.db locked",
									"unblocks_when": "lock exits",
									"owner": "operator",
									"pivot": "run next P0 row",
									"next_check": "2026-05-01T12:30:00-06:00"
								}
							},
							{"name": "pivot row", "status": "planned", "priority": "P1", "contract": "pivot contract", "contract_status": "draft", "no_test_required": "blocker metadata fixture"}
						]
					}
				}
			}
		}
	}`)

	got, err := NormalizeCandidates(path, CandidateOptions{ActiveFirst: true})
	if err != nil {
		t.Fatalf("NormalizeCandidates() error = %v", err)
	}
	if gotNames := candidateNames(got); !reflect.DeepEqual(gotNames, []string{"pivot row"}) {
		t.Fatalf("candidate names = %#v, want pivot row", gotNames)
	}
}

func writeProgressJSON(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "progress.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return path
}
