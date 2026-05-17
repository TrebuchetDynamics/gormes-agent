package progress

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

func TestAllowedModulesFeatureTaxonomy(t *testing.T) {
	want := []string{
		"browser",
		"builder",
		"channels",
		"cli",
		"config",
		"cross-cutting",
		"doctor",
		"docs",
		"fleet",
		"gateway",
		"goncho",
		"install",
		"kanban",
		"landing",
		"learning-loop",
		"memory",
		"navivox",
		"planner",
		"profiles",
		"progress",
		"providers",
		"release",
		"runtime",
		"sessions",
		"skills",
		"stt",
		"tools",
		"tts",
		"tui",
	}
	if got := AllowedModules(); !reflect.DeepEqual(got, want) {
		t.Fatalf("AllowedModules() = %#v, want %#v", got, want)
	}

	for _, old := range []string{
		"commands",
		"gateway-channels",
		"memory-sessions-skills",
		"orchestrator",
		"providers-auth",
		"setup-config-install",
		"unclassified",
	} {
		if slices.Contains(AllowedModules(), old) {
			t.Fatalf("old transitional module %q must not be allowed", old)
		}
	}
}

// (1) Module() is deterministic: an explicit per-row module always wins;
// otherwise it derives from execution_owner to an approved feature module;
// with no owner it is the explicit "unclassified" fallback bucket (never a
// silent mis-bucket, never an error). The fallback is compatibility only:
// C5 requires explicit valid modules on every row.
func TestModuleDeterministicDerivation(t *testing.T) {
	if got := Module(Item{Module: "custom-x", ExecutionOwner: ExecutionOwnerProvider}, "4", "4.A"); got != "custom-x" {
		t.Fatalf("explicit module must win, got %q", got)
	}

	cases := []struct {
		owner ExecutionOwner
		want  string
	}{
		{ExecutionOwnerDocs, "docs"},
		{ExecutionOwnerGateway, "gateway"},
		{ExecutionOwnerMemory, "memory"},
		{ExecutionOwnerProvider, "providers"},
		{ExecutionOwnerTools, "tools"},
		{ExecutionOwnerSkills, "skills"},
		{ExecutionOwnerOrchestrator, "fleet"},
		{ExecutionOwnerTui, "tui"},
		{ExecutionOwnerGoncho, "goncho"},
	}
	for _, c := range cases {
		if got := Module(Item{ExecutionOwner: c.owner}, "1", "1.A"); got != c.want {
			t.Fatalf("owner %q => module %q, want %q", c.owner, got, c.want)
		}
	}

	if got := Module(Item{}, "3", "3.G"); got != "unclassified" {
		t.Fatalf("no owner, no module => %q, want \"unclassified\"", got)
	}
	// Deterministic: same input, same output.
	if Module(Item{}, "3", "3.G") != Module(Item{}, "3", "3.G") {
		t.Fatal("Module must be deterministic")
	}
}

func moduleFixture() *Progress {
	return &Progress{
		Meta: Meta{Version: "2.0"},
		Phases: map[string]Phase{
			"1": {Name: "P1", Deliverable: "d1", Subphases: map[string]Subphase{
				"1.A": {Name: "A", Items: []Item{
					{Name: "r1", Status: StatusComplete, ExecutionOwner: ExecutionOwnerProvider},
					{Name: "r2", Status: StatusPlanned}, // no owner -> not backfilled
				}},
			}},
			"2": {Name: "P2", Deliverable: "d2", Subphases: map[string]Subphase{
				"2.A": {Name: "A", Items: []Item{
					{Name: "r3", Status: StatusComplete, ExecutionOwner: ExecutionOwnerTui},
				}},
			}},
		},
	}
}

func saveBytesM(t *testing.T, p *Progress) []byte {
	t.Helper()
	path := filepath.Join(t.TempDir(), "p.json")
	if err := SaveProgress(path, p); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return b
}

// (2) The new field is marshal-neutral: with no Item.Module set, SaveProgress
// output is byte-identical to a model that never had the field; the
// `module` JSON key only appears when explicitly set (omitempty).
func TestModuleFieldMarshalNeutral(t *testing.T) {
	p := moduleFixture()
	before := saveBytesM(t, p)

	// Round-trip: load the just-saved bytes and re-save — byte-identical.
	path := filepath.Join(t.TempDir(), "rt.json")
	if err := SaveProgress(path, p); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := saveBytesM(t, reloaded); !reflect.DeepEqual(got, before) {
		t.Fatal("no-module model must round-trip byte-identically (omitempty neutrality)")
	}

	// Setting module on one row adds exactly the `module` key, nothing else.
	p.Phases["1"].Subphases["1.A"].Items[0].Module = "providers"
	withMod := saveBytesM(t, p)
	if string(withMod) == string(before) {
		t.Fatal("setting module must change the output")
	}
	back, err := Load(filepathWrite(t, withMod))
	if err != nil {
		t.Fatalf("Load withMod: %v", err)
	}
	if back.Phases["1"].Subphases["1.A"].Items[0].Module != "providers" {
		t.Fatalf("module must round-trip, got %q", back.Phases["1"].Subphases["1.A"].Items[0].Module)
	}
}

func filepathWrite(t *testing.T, b []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "w.json")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return p
}

// (3) Backfill is validator-neutral, sets `module` only where derivable from
// execution_owner, changes only that field, and is idempotent.
func TestBackfillModulesIdempotentAndValidatorNeutral(t *testing.T) {
	p := moduleFixture()
	p.Meta.Version = "2.0"

	validBefore := Validate(p)

	n := BackfillModules(p)
	if n != 2 {
		t.Fatalf("backfill should set module on the 2 owner-bearing rows, got n=%d", n)
	}
	got := p.Phases["1"].Subphases["1.A"].Items
	if got[0].Module != "providers" {
		t.Fatalf("r1 (provider) module = %q, want providers", got[0].Module)
	}
	if got[1].Module != "" {
		t.Fatalf("r2 (no owner) must NOT be backfilled, got %q", got[1].Module)
	}
	if p.Phases["2"].Subphases["2.A"].Items[0].Module != "tui" {
		t.Fatalf("r3 (tui) module wrong: %q", p.Phases["2"].Subphases["2.A"].Items[0].Module)
	}

	// Idempotent: second pass is a no-op.
	if n2 := BackfillModules(p); n2 != 0 {
		t.Fatalf("second backfill must be a no-op, got n=%d", n2)
	}

	// Validator verdict unchanged (nil before and after).
	if (validBefore == nil) != (Validate(p) == nil) {
		t.Fatalf("backfill must be validator-neutral: before=%v after=%v", validBefore, Validate(p))
	}

	// Only `module` changed: clearing it restores deep equality with a fresh fixture.
	fresh := moduleFixture()
	for _, ph := range p.Phases {
		for _, sp := range ph.Subphases {
			for i := range sp.Items {
				sp.Items[i].Module = ""
			}
		}
	}
	if !reflect.DeepEqual(p, fresh) {
		t.Fatal("backfill must change ONLY the module field on each row")
	}
}

func TestSuggestedModuleRoutesFeatureRows(t *testing.T) {
	cases := []struct {
		name     string
		phase    string
		subphase string
		row      Item
		want     string
	}{
		{
			name:     "provider setup belongs to providers",
			phase:    "5",
			subphase: "5.O",
			row:      Item{Name: "Gormes setup model step uses the dynamic provider-tracked model picker", ExecutionOwner: ExecutionOwnerTools},
			want:     ModuleProviders,
		},
		{
			name:     "navivox is its own module",
			phase:    "9",
			subphase: "9.F",
			row:      Item{Name: "Navivox continuous voice command mode", ExecutionOwner: ExecutionOwnerGateway},
			want:     ModuleNavivox,
		},
		{
			name:     "kanban is its own module",
			phase:    "5",
			subphase: "5.M",
			row:      Item{Name: "Hermes Kanban durable board core", ExecutionOwner: ExecutionOwnerOrchestrator},
			want:     ModuleKanban,
		},
		{
			name:     "tts and stt split",
			phase:    "5",
			subphase: "5.E",
			row:      Item{Name: "TTS tool contract + media delivery seam", ExecutionOwner: ExecutionOwnerTools},
			want:     ModuleTTS,
		},
		{
			name:     "transcription is stt",
			phase:    "5",
			subphase: "5.E",
			row:      Item{Name: "Transcription tool contract", ExecutionOwner: ExecutionOwnerTools},
			want:     ModuleSTT,
		},
		{
			name:     "progress split row",
			phase:    "8",
			subphase: "8.F",
			row:      Item{Name: "Backlog split C5g: explicitly classify every row into a valid feature module", ExecutionOwner: ExecutionOwnerTools},
			want:     ModuleProgress,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := SuggestedModule(tc.phase, "", tc.subphase, "", tc.row)
			if got != tc.want {
				t.Fatalf("SuggestedModule() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAssignModulesChangesOnlyModuleAndAuditsCounts(t *testing.T) {
	fixture := func() *Progress {
		return &Progress{
			Meta: Meta{Version: "2.0"},
			Phases: map[string]Phase{
				"1": {Name: "P1", Deliverable: "d1", Subphases: map[string]Subphase{
					"1.A": {Name: "Core TUI", Items: []Item{
						{Name: "Bubble Tea shell", Status: StatusComplete},
						{Name: "OpenRouter compatible-provider routing", Status: StatusPlanned, ExecutionOwner: ExecutionOwnerProvider},
						{Name: "Manual review", Status: StatusPlanned, Module: ModuleDocs},
					}},
				}},
			},
		}
	}
	p := fixture()

	audit := AssignModules(p)
	if audit.Total != 3 || audit.Changed != 2 || audit.Preserved != 1 {
		t.Fatalf("audit counts = total %d changed %d preserved %d, want 3/2/1", audit.Total, audit.Changed, audit.Preserved)
	}
	items := p.Phases["1"].Subphases["1.A"].Items
	if got := []string{items[0].Module, items[1].Module, items[2].Module}; !reflect.DeepEqual(got, []string{ModuleTUI, ModuleProviders, ModuleDocs}) {
		t.Fatalf("modules = %#v", got)
	}
	if audit.ByModule[ModuleTUI] != 1 || audit.ByModule[ModuleProviders] != 1 || audit.ByModule[ModuleDocs] != 1 {
		t.Fatalf("audit module counts = %#v", audit.ByModule)
	}

	for _, ph := range p.Phases {
		for _, sp := range ph.Subphases {
			for i := range sp.Items {
				sp.Items[i].Module = ""
			}
		}
	}
	want := fixture()
	want.Phases["1"].Subphases["1.A"].Items[2].Module = ""
	if !reflect.DeepEqual(p, want) {
		t.Fatal("AssignModules must change only Item.Module fields")
	}
}
