package internal_test

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

func defaultTopologyMigrations() []topologyMigration {
	return []topologyMigration{
		{
			Name:        "cli-surface-rehome",
			Active:      true,
			OwnerModule: "cli",
			OldRoots:    []string{"internal/app/gormescli"},
			NewRoots:    []string{"internal/cli/gormescli"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Topology Guard Contract",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"cmd/gormes/main.go:rootCommandFactories",
				"internal/cli/gormescli/root.go:NewRootCommand",
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
			OldRoots:    []string{"internal/sessionsearchtool"},
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
				"internal/acp/browser_bootstrap.go:BootstrapBrowserHarness",
				"internal/progress/plannerloop/run.go:Run",
			},
		},
		{
			Name:        "progressctl-rehome",
			Active:      true,
			OwnerModule: "progress",
			OldRoots:    []string{"internal/progressctl"},
			NewRoots:    []string{"internal/progress/ctl"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Progress Delivery Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"cmd/progress/main.go:main",
				"internal/progress/ctl/progressctl.go:Validate",
				"internal/progress/ctl/progressctl.go:Write",
			},
		},
		{
			Name:        "plannertriggers-rehome",
			Active:      true,
			OwnerModule: "progress",
			OldRoots:    []string{"internal/plannertriggers"},
			NewRoots:    []string{"internal/progress/triggers"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Progress Delivery Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"internal/progress/triggers/triggers.go:TriggerEvent",
				"internal/progress/builderloop/run.go:emitPlannerTriggers",
				"internal/progress/plannerloop/run.go:Run",
			},
		},
		{
			Name:        "builderloop-rehome",
			Active:      true,
			OwnerModule: "progress",
			OldRoots:    []string{"internal/builderloop"},
			NewRoots:    []string{"internal/progress/builderloop"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Progress Delivery Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"internal/progress/builderloop/run.go:RunOnce",
				"internal/progress/ctl/progressctl.go:NextWorkWithOptions",
				"internal/progress/plannerloop/run.go:Run",
			},
		},
		{
			Name:        "plannerloop-rehome",
			Active:      true,
			OwnerModule: "progress",
			OldRoots:    []string{"internal/plannerloop"},
			NewRoots:    []string{"internal/progress/plannerloop"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Progress Delivery Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"internal/progress/plannerloop/run.go:Run",
				"internal/progress/plannerloop/implscan.go:ScanImplementation",
				"internal/progress/builderloop/candidates.go:NormalizeCandidates",
			},
		},
		{
			Name:        "repoctl-rehome",
			Active:      true,
			OwnerModule: "progress",
			OldRoots:    []string{"internal/repoctl"},
			NewRoots:    []string{"internal/progress/repoctl"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Progress Delivery Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"cmd/repoctl/main.go:run",
				"internal/progress/repoctl/bench.go:RecordBenchmark",
				"internal/progress/repoctl/hermes_contract_inventory.go:WriteHermesContractInventory",
			},
		},
		{
			Name:        "fidelity-rehome",
			Active:      true,
			OwnerModule: "progress",
			OldRoots:    []string{"internal/fidelity"},
			NewRoots:    []string{"internal/progress/fidelity"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Progress Delivery Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"cmd/gormes/fidelity.go:runFidelityHermesCommand",
				"internal/progress/fidelity/report.go:GenerateHermesReport",
				"internal/progress/repoctl/hermes_contract_inventory.go:BuildHermesContractInventory",
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
			NewRoots:    []string{"internal/channels/slack"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Channel Gateway Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"cmd/gormes/gateway.go:runGateway",
				"internal/channels/slack/channel.go:NewChannel",
				"internal/channels/slack/bus_adapter.go:NewBusAdapter",
			},
		},
		{
			Name:        "discord-legacy-rehome",
			Active:      true,
			OwnerModule: "channels",
			OldRoots:    []string{"internal/discord"},
			NewRoots:    []string{"internal/channels/discord/legacy"},
			SourceRefs: []string{
				"internal/REFACTOR-CMD-PLAN.md:Channel Gateway Enclave",
				"internal/REFACTOR-CMD-PLAN.md:Package Move Playbook",
				"internal/channels/discord/bot.go:New",
				"internal/channels/discord/legacy/bot.go:New",
				"internal/channels/discord/legacy/render.go:formatStream",
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
	internalDir := filepath.Dir(file)
	if filepath.Base(internalDir) != "internal" {
		t.Fatalf("expected topology test under internal/, got %s", internalDir)
	}
	return filepath.Dir(internalDir)
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
