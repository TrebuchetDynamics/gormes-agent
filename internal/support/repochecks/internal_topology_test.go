package repochecks_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

const gormesModulePath = "github.com/TrebuchetDynamics/gormes-agent"

type internalTopologyReport struct {
	InternalTopLevelDirsMeasured       bool
	InternalTopLevelDirs               int
	CmdGormesDirectInternalImportsSeen bool
	CmdGormesDirectInternalImports     int
	DirectInternalImports              []string
}

type topologyMigration struct {
	Name        string
	Active      bool
	OwnerModule string
	OldRoots    []string
	NewRoots    []string
	SourceRefs  []string
}

type hermesOwnerSeam struct {
	OwnerModule string
	HermesRefs  []string
	GormesRoots []string
}

type topologyViolation struct {
	Migration   string
	OwnerModule string
	OldRoot     string
	Scope       string
	Detail      string
}

func (v topologyViolation) String() string {
	return fmt.Sprintf("%s owner=%s old_root=%s scope=%s detail=%s", v.Migration, v.OwnerModule, v.OldRoot, v.Scope, v.Detail)
}

func TestInternalTopologyReportsCurrentMetrics(t *testing.T) {
	report := loadInternalTopologyReport(t)
	if !report.InternalTopLevelDirsMeasured || report.InternalTopLevelDirs == 0 {
		t.Fatalf("expected internal top-level directory metric to be reported, got %#v", report)
	}
	if !report.CmdGormesDirectInternalImportsSeen {
		t.Fatalf("expected cmd/gormes direct internal import metric to be reported, got %#v", report)
	}
	t.Logf("internal_top_level_dirs=%d cmd_gormes_direct_internal_imports=%d", report.InternalTopLevelDirs, report.CmdGormesDirectInternalImports)
}

func TestInternalTopologyPrimaryDirectoryBudget(t *testing.T) {
	report := loadInternalTopologyReport(t)
	const primaryTarget = 46
	if report.InternalTopLevelDirs > primaryTarget {
		t.Fatalf("internal top-level dirs = %d, want <= %d per internal/REFACTOR-CMD-PLAN.md primary target", report.InternalTopLevelDirs, primaryTarget)
	}
}

func TestInternalTopologyPrimaryImportBudget(t *testing.T) {
	report := loadInternalTopologyReport(t)
	const primaryTarget = 35
	if report.CmdGormesDirectInternalImports > primaryTarget {
		t.Fatalf("cmd/gormes direct internal imports = %d, want <= %d per internal/REFACTOR-CMD-PLAN.md closeout target\n%s", report.CmdGormesDirectInternalImports, primaryTarget, strings.Join(report.DirectInternalImports, "\n"))
	}
}

func TestInternalTopologyMigrationEntryShape(t *testing.T) {
	entry := findTopologyMigration(t, defaultTopologyMigrations(), "cli-surface-rehome")
	if entry.OwnerModule == "" || len(entry.OldRoots) == 0 || len(entry.NewRoots) == 0 || len(entry.SourceRefs) == 0 {
		t.Fatalf("migration entry must carry owner_module, old_roots, new_roots, and source_refs: %#v", entry)
	}
	if !entry.Active {
		t.Fatal("CLI surface rehome migration must be active after the package-move row starts")
	}
}

func TestInternalTopologyActiveMigrationsPass(t *testing.T) {
	violations := evaluateTopologyMigrations(t, defaultTopologyMigrations())
	if len(violations) > 0 {
		t.Fatalf("active topology migrations failed:\n%s", formatTopologyViolations(violations))
	}
}

func TestInternalTopologyDetectsForbiddenLegacyRoot(t *testing.T) {
	entry := findTopologyMigration(t, defaultTopologyMigrations(), "cli-surface-rehome")
	entry.Name = "synthetic-stale-cli-root"
	entry.OldRoots = []string{"internal/cli/gormescli"}
	violations := evaluateTopologyMigrations(t, []topologyMigration{entry})
	if len(violations) == 0 {
		t.Fatal("expected topology guard to fail when an active migration still has a forbidden legacy root")
	}
	if !topologyViolationsMention(violations, "internal/cli/gormescli") {
		t.Fatalf("expected synthetic red signal to mention internal/cli/gormescli, got:\n%s", formatTopologyViolations(violations))
	}
}

func TestInternalTopologyProtectsHermesSourceFamilyOwners(t *testing.T) {
	repoRoot := topologyRepoRoot(t)
	for _, seam := range protectedHermesOwnerSeams() {
		for _, root := range seam.GormesRoots {
			if !pathExists(filepath.Join(repoRoot, filepath.FromSlash(root))) {
				t.Fatalf("Hermes source-family owner seam %q for %s is missing Gormes root %s", seam.OwnerModule, strings.Join(seam.HermesRefs, ", "), root)
			}
		}
		if len(seam.HermesRefs) == 0 {
			t.Fatalf("Hermes source-family owner seam %q must carry source refs", seam.OwnerModule)
		}
	}
}

func protectedHermesOwnerSeams() []hermesOwnerSeam {
	return []hermesOwnerSeam{
		{
			OwnerModule: "plugins",
			HermesRefs:  []string{"hermes-agent/hermes_cli/plugins.py", "hermes-agent/plugins/*", "hermes-agent/agent/*_registry.py"},
			GormesRoots: []string{"internal/extensibility/plugins"},
		},
		{
			OwnerModule: "providers",
			HermesRefs:  []string{"hermes-agent/providers/__init__.py", "hermes-agent/plugins/model-providers/*"},
			GormesRoots: []string{"internal/provider", "internal/llm", "internal/extensibility/plugins"},
		},
		{
			OwnerModule: "tools",
			HermesRefs:  []string{"hermes-agent/tools/registry.py", "hermes-agent/tools/*"},
			GormesRoots: []string{"internal/tools"},
		},
		{
			OwnerModule: "channels",
			HermesRefs:  []string{"hermes-agent/gateway/run.py", "hermes-agent/gateway/platforms/*", "hermes-agent/plugins/platforms/*"},
			GormesRoots: []string{"internal/gateway", "internal/adapters/channels", "internal/extensibility/plugins"},
		},
		{
			OwnerModule: "acp",
			HermesRefs:  []string{"hermes-agent/acp_adapter/*", "hermes-agent/acp_registry/agent.json"},
			GormesRoots: []string{"internal/protocols/acp"},
		},
		{
			OwnerModule: "cron",
			HermesRefs:  []string{"hermes-agent/cron/scheduler.py", "hermes-agent/cron/jobs.py"},
			GormesRoots: []string{"internal/automation/cron"},
		},
		{
			OwnerModule: "tui-gateway",
			HermesRefs:  []string{"hermes-agent/ui-tui/src/app.tsx", "hermes-agent/ui-tui/src/lib/gatewayClient.ts"},
			GormesRoots: []string{"internal/tui", "internal/adapters/tuigateway", "internal/adapters/apiserver"},
		},
		{
			OwnerModule: "skills",
			HermesRefs:  []string{"hermes-agent/skills/*", "hermes-agent/optional-skills/*", "hermes-agent/tools/skills_tool.py"},
			GormesRoots: []string{"internal/extensibility/skills"},
		},
		{
			OwnerModule: "progress",
			HermesRefs:  []string{"Gormes-owned: no Hermes source-family analogue"},
			GormesRoots: []string{"internal/planning/progress"},
		},
	}
}

func defaultTopologyMigrations() []topologyMigration {
	return []topologyMigration{
		{
			Name:        "cli-surface-rehome",
			Active:      true,
			OwnerModule: "cli",
			OldRoots:    []string{"internal/app/gormescli"},
			NewRoots:    []string{"internal/platform/cli/gormescli"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Topology Guard Contract",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"cmd/gormes/main.go:rootCommandFactories",
				"internal/platform/cli/gormescli/root.go:NewRootCommand",
				"hermes-agent/hermes_cli/_parser.py:build_top_level_parser",
				"hermes-agent/hermes_cli/commands.py:COMMAND_REGISTRY",
			},
		},
		{
			Name:        "toolcompact-rehome",
			Active:      true,
			OwnerModule: "tools",
			OldRoots:    []string{"internal/toolcompact"},
			NewRoots:    []string{"internal/tools/compact"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Tool Adapter Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"hermes-agent/tools/registry.py",
				"hermes-agent/tools/terminal_tool.py",
				"hermes-agent/tools/tool_output_limits.py",
				"internal/tools/compact/compact.go:Compact",
			},
		},
		{
			Name:        "tooltrace-rehome",
			Active:      true,
			OwnerModule: "tools",
			OldRoots:    []string{"internal/tooltrace"},
			NewRoots:    []string{"internal/tools/trace"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Tool Adapter Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"hermes-agent/gateway/run.py",
				"hermes-agent/agent/display.py",
				"internal/tools/trace/tooltrace.go:FormatPlain",
			},
		},
		{
			Name:        "sessionsearchtool-rehome",
			Active:      true,
			OwnerModule: "tools",
			OldRoots:    []string{"internal/persistence/sessionsearchtool"},
			NewRoots:    []string{"internal/tools/sessionsearch"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Tool Adapter Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md:Tool registry and toolsets",
				"hermes-agent/tools/session_search_tool.py",
				"hermes-agent/tests/tools/test_session_search.py",
				"internal/tools/sessionsearch/session_search_tool.go:NewSessionSearchTool",
				"cmd/gormes/registry.go:buildDefaultRegistry",
			},
		},
		{
			Name:        "cmdrunner-runtime-rehome",
			Active:      true,
			OwnerModule: "runtime",
			OldRoots:    []string{"internal/cmdrunner"},
			NewRoots:    []string{"internal/runtime/cmdrunner"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Progress Delivery Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Runtime Mechanics Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"internal/runtime/cmdrunner/runner.go:Runner",
				"internal/protocols/acp/browser_bootstrap.go:BootstrapBrowserHarness",
				"internal/planning/progress/plannerloop/run.go:Run",
			},
		},
		{
			Name:        "progressctl-rehome",
			Active:      true,
			OwnerModule: "progress",
			OldRoots:    []string{"internal/planning/progressctl"},
			NewRoots:    []string{"internal/planning/progress/ctl"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Progress Delivery Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"cmd/progress/main.go:main",
				"internal/planning/progress/ctl/progressctl.go:Validate",
				"internal/planning/progress/ctl/progressctl.go:Write",
			},
		},
		{
			Name:        "plannertriggers-rehome",
			Active:      true,
			OwnerModule: "progress",
			OldRoots:    []string{"internal/plannertriggers"},
			NewRoots:    []string{"internal/planning/progress/triggers"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Progress Delivery Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"internal/planning/progress/triggers/triggers.go:TriggerEvent",
				"internal/planning/progress/triggers/triggers.go:AppendTriggerEvent",
				"internal/planning/progress/plannerloop/run.go:Run",
			},
		},
		{
			Name:        "plannerloop-rehome",
			Active:      true,
			OwnerModule: "progress",
			OldRoots:    []string{"internal/plannerloop"},
			NewRoots:    []string{"internal/planning/progress/plannerloop"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Progress Delivery Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"internal/planning/progress/plannerloop/run.go:Run",
				"internal/planning/progress/plannerloop/implscan.go:ScanImplementation",
				"internal/planning/progress/projections.go:ProjectActiveHandoffs",
			},
		},
		{
			Name:        "repoctl-rehome",
			Active:      true,
			OwnerModule: "progress",
			OldRoots:    []string{"internal/repoctl"},
			NewRoots:    []string{"internal/planning/progress/repoctl"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Progress Delivery Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"cmd/repoctl/main.go:run",
				"internal/planning/progress/repoctl/bench.go:RecordBenchmark",
				"internal/planning/progress/repoctl/hermes_contract_inventory.go:WriteHermesContractInventory",
			},
		},
		{
			Name:        "fidelity-rehome",
			Active:      true,
			OwnerModule: "progress",
			OldRoots:    []string{"internal/fidelity"},
			NewRoots:    []string{"internal/planning/progress/fidelity"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Progress Delivery Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"cmd/gormes/fidelity.go:runFidelityHermesCommand",
				"internal/planning/progress/fidelity/report.go:GenerateHermesReport",
				"internal/planning/progress/repoctl/hermes_contract_inventory.go:BuildHermesContractInventory",
			},
		},
		{
			Name:        "kanbantools-rehome",
			Active:      true,
			OwnerModule: "tools",
			OldRoots:    []string{"internal/kanbantools"},
			NewRoots:    []string{"internal/tools/kanban"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Tool Adapter Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"cmd/gormes/registry.go:buildDefaultRegistry",
				"internal/tools/kanban/kanban_tools.go:NewTools",
				"internal/tools/kanban/kanban_tools.go:ConfigFromEnv",
			},
		},
		{
			Name:        "gonchotools-rehome",
			Active:      true,
			OwnerModule: "tools",
			OldRoots:    []string{"internal/gonchotools"},
			NewRoots:    []string{"internal/tools/goncho"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Tool Adapter Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"cmd/gormes/goncho.go:runGonchoDoctor",
				"internal/tools/goncho/honcho_tools.go:RegisterHonchoTools",
				"internal/tools/goncho/turn_integration.go:NewTurnIntegration",
			},
		},
		{
			Name:        "slack-channel-rehome",
			Active:      true,
			OwnerModule: "channels",
			OldRoots:    []string{"internal/slack"},
			NewRoots:    []string{"internal/adapters/channels/slack"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Channel Gateway Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"cmd/gormes/gateway.go:runGateway",
				"internal/adapters/channels/slack/channel.go:NewChannel",
				"internal/adapters/channels/slack/bus_adapter.go:NewBusAdapter",
			},
		},
		{
			Name:        "discord-legacy-rehome",
			Active:      true,
			OwnerModule: "channels",
			OldRoots:    []string{"internal/discord"},
			NewRoots:    []string{"internal/adapters/channels/discord/legacy"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Channel Gateway Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"internal/adapters/channels/discord/bot.go:New",
				"internal/adapters/channels/discord/legacy/bot.go:New",
				"internal/adapters/channels/discord/legacy/render.go:formatStream",
			},
		},
		{
			Name:        "lsp-tools-rehome",
			Active:      true,
			OwnerModule: "tools",
			OldRoots:    []string{"internal/lsp"},
			NewRoots:    []string{"internal/tools/lsp"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Tool Adapter Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"internal/tools/file_task_tools.go:executeCode",
				"internal/tools/lsp/diagnostics.go:ReadDiagnostics",
				"internal/tools/file_lsp_diagnostics_test.go:TestExecuteCodeIncludesLSPDiagnostics",
			},
		},
		{
			Name:        "whisper-tools-rehome",
			Active:      true,
			OwnerModule: "tools",
			OldRoots:    []string{"internal/wasi/whisper"},
			NewRoots:    []string{"internal/tools/whisper"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Tool Adapter Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"cmd/gormes/telegram_transcriber.go:buildTelegramTranscriberFactory",
				"internal/tools/transcription_providers_local.go:NewLocalTranscriptionProvider",
				"internal/tools/whisper/transcriber.go:NewTranscriber",
			},
		},
		{
			Name:        "runtimebridge-rehome",
			Active:      true,
			OwnerModule: "runtime",
			OldRoots:    []string{"internal/runtimebridge"},
			NewRoots:    []string{"internal/runtime/bridge"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Runtime Mechanics Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"internal/runtime/bridge/runtimebridge.go:Runtime",
				"internal/runtime/bridge/runtimebridge.go:NoRuntime",
			},
		},
		{
			Name:        "profileseed-cli-rehome",
			Active:      true,
			OwnerModule: "cli",
			OldRoots:    []string{"internal/profileseed"},
			NewRoots:    []string{"internal/platform/cli/profileseed"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:CLI Surface Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"cmd/gormes/profile_seed.go:runProfileSeedDraft",
				"internal/adapters/channels/navivox/channel.go:handleProfileSeed",
				"internal/platform/cli/profileseed/seed.go:NewDraft",
			},
		},
		{
			Name:        "modelcatalog-cli-rehome",
			Active:      true,
			OwnerModule: "cli",
			OldRoots:    []string{"internal/modelcatalog"},
			NewRoots:    []string{"internal/platform/cli/modelcatalog"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:CLI Surface Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"internal/platform/cli/provider_catalog.go:ProviderCatalogEntries",
				"internal/tui/model_picker_catalog.go:ModelProviderCatalogEntries",
				"internal/platform/cli/modelcatalog/catalog.go:HermesProviderCatalog",
			},
		},
		{
			Name:        "contextrefs-transcript-rehome",
			Active:      true,
			OwnerModule: "transcript",
			OldRoots:    []string{"internal/contextrefs"},
			NewRoots:    []string{"internal/persistence/transcript/contextrefs"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Second-pass leaf triage",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"internal/llm/context_references.go:AttachContextReferenceHandles",
				"internal/persistence/transcript/refs.go:NewContextReferenceStore",
				"internal/persistence/transcript/contextrefs/refs.go:NewStore",
			},
		},
		{
			Name:        "loopcost-progress-rehome",
			Active:      true,
			OwnerModule: "progress",
			OldRoots:    []string{"internal/loopcost"},
			NewRoots:    []string{"internal/planning/progress/loopcost"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Second-pass leaf triage",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"internal/planning/progress/loopcost/cost.go:ParseRunCost",
				"internal/planning/progress/loopcost/cost.go:DailyRollup",
			},
		},
		{
			Name:        "i18n-tui-rehome",
			Active:      true,
			OwnerModule: "tui",
			OldRoots:    []string{"internal/i18n"},
			NewRoots:    []string{"internal/tui/i18n"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Second-pass leaf triage",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"internal/tui/i18n/i18n.go:SupportedLanguages",
				"internal/tui/i18n/i18n.go:T",
			},
		},
		{
			Name:        "events-gateway-rehome",
			Active:      true,
			OwnerModule: "gateway",
			OldRoots:    []string{"internal/events"},
			NewRoots:    []string{"internal/gateway/events"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Second-pass leaf triage",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"internal/gateway/event_dispatch.go:EventDispatcher",
				"internal/gateway/events/bus.go:NewInProcessEventBus",
				"internal/gateway/events/types.go:NewEvent",
			},
		},
	}
}

func loadInternalTopologyReport(t *testing.T) internalTopologyReport {
	t.Helper()
	repoRoot := topologyRepoRoot(t)
	internalDirs := countInternalTopLevelDirs(t, filepath.Join(repoRoot, "internal"))
	imports := cmdGormesDirectInternalImports(t, repoRoot)
	return internalTopologyReport{
		InternalTopLevelDirsMeasured:       true,
		InternalTopLevelDirs:               internalDirs,
		CmdGormesDirectInternalImportsSeen: true,
		CmdGormesDirectInternalImports:     len(imports),
		DirectInternalImports:              imports,
	}
}

func evaluateTopologyMigrations(t *testing.T, migrations []topologyMigration) []topologyViolation {
	t.Helper()
	repoRoot := topologyRepoRoot(t)
	var violations []topologyViolation
	for _, migration := range migrations {
		if !migration.Active {
			continue
		}
		if err := validateTopologyMigrationShape(migration); err != nil {
			violations = append(violations, topologyViolation{Migration: migration.Name, OwnerModule: migration.OwnerModule, Scope: "migration", Detail: err.Error()})
			continue
		}
		for _, oldRoot := range migration.OldRoots {
			oldRoot = filepath.ToSlash(strings.TrimSpace(oldRoot))
			if oldRoot == "" {
				continue
			}
			if pathExists(filepath.Join(repoRoot, filepath.FromSlash(oldRoot))) {
				violations = append(violations, topologyViolation{Migration: migration.Name, OwnerModule: migration.OwnerModule, OldRoot: oldRoot, Scope: "filesystem", Detail: "legacy root still exists"})
			}
			for _, ref := range legacyRootReferences(t, repoRoot, oldRoot) {
				violations = append(violations, topologyViolation{Migration: migration.Name, OwnerModule: migration.OwnerModule, OldRoot: oldRoot, Scope: "reference", Detail: ref})
			}
		}
		for _, newRoot := range migration.NewRoots {
			newRoot = filepath.ToSlash(strings.TrimSpace(newRoot))
			if newRoot == "" {
				continue
			}
			if !pathExists(filepath.Join(repoRoot, filepath.FromSlash(newRoot))) {
				violations = append(violations, topologyViolation{Migration: migration.Name, OwnerModule: migration.OwnerModule, Scope: "filesystem", Detail: "new root missing: " + newRoot})
			}
		}
	}
	sort.Slice(violations, func(i, j int) bool {
		return violations[i].String() < violations[j].String()
	})
	return violations
}

func validateTopologyMigrationShape(migration topologyMigration) error {
	if strings.TrimSpace(migration.Name) == "" {
		return fmt.Errorf("missing migration name")
	}
	if strings.TrimSpace(migration.OwnerModule) == "" {
		return fmt.Errorf("missing owner_module")
	}
	if len(migration.OldRoots) == 0 {
		return fmt.Errorf("missing old_roots")
	}
	if len(migration.NewRoots) == 0 {
		return fmt.Errorf("missing new_roots")
	}
	if len(migration.SourceRefs) == 0 {
		return fmt.Errorf("missing source_refs")
	}
	return nil
}

func countInternalTopLevelDirs(t *testing.T, internalDir string) int {
	t.Helper()
	entries, err := os.ReadDir(internalDir)
	if err != nil {
		t.Fatalf("read %s: %v", internalDir, err)
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		count++
	}
	return count
}

func cmdGormesDirectInternalImports(t *testing.T, repoRoot string) []string {
	t.Helper()
	cmd := exec.Command("go", "list", "-f", "{{join .Imports \"\\n\"}}", "./cmd/gormes")
	cmd.Dir = repoRoot
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("go list ./cmd/gormes imports failed: %v\n%s", err, out.String())
	}
	var imports []string
	for _, line := range strings.Split(out.String(), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, gormesModulePath+"/internal") {
			imports = append(imports, line)
		}
	}
	sort.Strings(imports)
	return imports
}

func legacyRootReferences(t *testing.T, repoRoot, oldRoot string) []string {
	t.Helper()
	needles := []string{oldRoot, gormesModulePath + "/" + oldRoot}
	var refs []string
	for _, root := range []string{"cmd", "internal"} {
		walkRoot := filepath.Join(repoRoot, root)
		err := filepath.WalkDir(walkRoot, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if strings.HasPrefix(entry.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Base(path) == "internal_topology_test.go" || !strings.HasSuffix(path, ".go") {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(raw)
			for _, needle := range needles {
				if strings.Contains(text, needle) {
					rel, relErr := filepath.Rel(repoRoot, path)
					if relErr != nil {
						rel = path
					}
					refs = append(refs, filepath.ToSlash(rel)+" contains "+needle)
					break
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan legacy root references under %s: %v", walkRoot, err)
		}
	}
	sort.Strings(refs)
	return refs
}

func findTopologyMigration(t *testing.T, migrations []topologyMigration, name string) topologyMigration {
	t.Helper()
	for _, migration := range migrations {
		if migration.Name == name {
			return migration
		}
	}
	t.Fatalf("topology migration %q not found", name)
	return topologyMigration{}
}

func formatTopologyViolations(violations []topologyViolation) string {
	if len(violations) == 0 {
		return "none"
	}
	lines := make([]string, len(violations))
	for i, violation := range violations {
		lines[i] = violation.String()
	}
	return strings.Join(lines, "\n")
}

func topologyViolationsMention(violations []topologyViolation, needle string) bool {
	for _, violation := range violations {
		if strings.Contains(violation.String(), needle) {
			return true
		}
	}
	return false
}

func topologyRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve topology test file path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	if !pathExists(filepath.Join(repoRoot, "go.mod")) {
		t.Fatalf("expected topology repo root to contain go.mod, got %s", repoRoot)
	}
	return repoRoot
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
