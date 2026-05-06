package docs_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

var staleGormesIORefPattern = regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?gormes\.io`)

func TestTargetsIncludeLandingPageDocs(t *testing.T) {
	want := map[string]bool{
		"superpowers/specs/2026-04-19-gormes-landing-page-design.md": false,
		"superpowers/plans/2026-04-19-gormes-landing-page.md":        false,
	}

	for _, target := range targets {
		if _, ok := want[target]; ok {
			want[target] = true
		}
	}

	for rel, seen := range want {
		if !seen {
			t.Fatalf("docs target missing %s", rel)
		}
	}
}

func TestTargetsIncludeAICutoverDocs(t *testing.T) {
	want := map[string]bool{
		"superpowers/specs/2026-04-19-gormes-ai-cutover-design.md": false,
		"superpowers/plans/2026-04-19-gormes-ai-cutover.md":        false,
	}

	for _, target := range targets {
		if _, ok := want[target]; ok {
			want[target] = true
		}
	}

	for rel, seen := range want {
		if !seen {
			t.Fatalf("docs target missing %s", rel)
		}
	}
}

func TestTargetsIncludeManifestoSyncDocs(t *testing.T) {
	want := map[string]bool{
		"superpowers/specs/2026-04-19-gormes-doc-sync-manifesto-design.md": false,
		"superpowers/plans/2026-04-19-gormes-doc-sync-manifesto.md":        false,
	}

	for _, target := range targets {
		if _, ok := want[target]; ok {
			want[target] = true
		}
	}

	for rel, seen := range want {
		if !seen {
			t.Fatalf("docs target missing %s", rel)
		}
	}
}

func TestTargetsIncludePhase2CPersistenceDocs(t *testing.T) {
	want := map[string]bool{
		"superpowers/specs/2026-04-19-gormes-phase2c-persistence-design.md": false,
		"superpowers/plans/2026-04-19-gormes-phase2c-persistence.md":        false,
	}

	for _, target := range targets {
		if _, ok := want[target]; ok {
			want[target] = true
		}
	}

	for rel, seen := range want {
		if !seen {
			t.Fatalf("docs target missing %s", rel)
		}
	}
}

func TestDocsHarnessAllowsNativeGormesManifestoPage(t *testing.T) {
	if _, ok := nativeDocsPages["why-gormes.md"]; !ok {
		t.Fatalf("nativeDocsPages should explicitly allow why-gormes.md")
	}
	// Native Gormes pages live under why-gormes.md OR the building-gormes/
	// (contributor-facing) and using-gormes/ (operator-facing) sections.
	// Everything else is a mirrored upstream doc under upstream-hermes/.
	for page := range nativeDocsPages {
		if page == "_index.md" || page == "why-gormes.md" {
			continue
		}
		if strings.HasPrefix(page, "building-gormes/") || strings.HasPrefix(page, "using-gormes/") {
			continue
		}
		t.Fatalf("nativeDocsPages contains unexpected entry (must be _index.md, why-gormes.md, or under building-gormes/ or using-gormes/): %q", page)
	}
}

func TestDocsHomePageIsGormesBranded(t *testing.T) {
	raw := readDoc(t, "content/_index.md")
	wants := []string{
		`title: "Gormes Documentation"`,
		"# Gormes",
		"[Getting Started](getting-started/)",
		"Go-native runtime",
		"## What is Gormes?",
		"Offline proof path",
		"Two install paths",
		"## Support labels",
		"Runtime-ready",
		"## Trust posture",
		"Source build and inspectable",
		"Upstream Hermes Archive",
		"Roadmap & Parity",
	}
	for _, want := range wants {
		if !strings.Contains(raw, want) {
			t.Fatalf("docs home is missing %q", want)
		}
	}

	rejects := []string{
		"Hermes Agent Documentation",
		"# Hermes Agent",
		"The self-improving AI agent built by Nous Research.",
	}
	for _, reject := range rejects {
		if strings.Contains(raw, reject) {
			t.Fatalf("docs home should not contain stale copy %q", reject)
		}
	}
}

func TestGormesOperatorSetupChannelProviderDocs(t *testing.T) {
	rootHelp := runGormesHelp(t, "--help")
	setupHelp := runGormesHelp(t, "setup", "--help")
	modelHelp := runGormesHelp(t, "model", "--help")
	gatewayHelp := runGormesHelp(t, "gateway", "--help")

	assertContainsAll(t, "gormes --help", rootHelp, []string{
		"gormes setup provider",
		"gormes setup model",
		"gormes --oneshot \"hello\"",
		"gormes gateway",
		"gormes gateway status",
		"gormes telegram",
		"gormes whatsapp",
		"Interactively select the active model/provider",
	})
	assertContainsAll(t, "gormes setup --help", setupHelp, []string{
		"gormes setup [section]",
		"--non-interactive",
		"--quick",
		"--reconfigure",
		"--reset",
	})
	assertContainsAll(t, "gormes model --help", modelHelp, []string{
		"Interactively select the active model/provider",
		"gormes model [flags]",
	})
	assertContainsAll(t, "gormes gateway --help", gatewayHelp, []string{
		"gormes gateway [command]",
		"Inspect configured gateway channels",
		"Stop the live Gormes gateway",
	})

	home := readDoc(t, "content/_index.md")
	gettingStarted := readDoc(t, "content/getting-started/_index.md")
	for label, raw := range map[string]string{
		"content/_index.md":                 home,
		"content/getting-started/_index.md": gettingStarted,
	} {
		assertContainsAll(t, label, raw, []string{
			"First Run",
			"Configuration",
			"Provider Setup",
			"Gateway Operations",
			"Telegram Bot",
			"CLI",
			"Config",
			"Environment",
			"Providers",
		})
	}

	firstRun := readDoc(t, "content/getting-started/first-run.md")
	providerSetup := readDoc(t, "content/guides/provider-setup.md")
	for label, raw := range map[string]string{
		"content/getting-started/first-run.md": firstRun,
		"content/guides/provider-setup.md":     providerSetup,
	} {
		assertContainsAll(t, label, raw, []string{
			"gormes setup provider",
			"gormes setup model",
			"gormes model",
			"gormes config set",
			"gormes auth",
			".env",
			"gormes --oneshot",
			"gormes doctor",
		})
	}

	gatewayOps := readDoc(t, "content/guides/gateway-operations.md")
	telegram := readDoc(t, "content/guides/telegram-bot.md")
	cli := readDoc(t, "content/reference/cli.md")
	for label, raw := range map[string]string{
		"content/guides/gateway-operations.md": gatewayOps,
		"content/guides/telegram-bot.md":       telegram,
		"content/reference/cli.md":             cli,
	} {
		assertContainsAll(t, label, raw, []string{
			"Runtime-ready",
			"row-backed",
			"gormes gateway status",
			"gormes whatsapp",
		})
	}

	providers := readDoc(t, "content/reference/providers.md")
	assertContainsAll(t, "content/reference/providers.md", providers, []string{
		"upstream-hermes/integrations/providers",
		"upstream-hermes/reference/model-catalog",
		"runtime-implemented providers",
		"row-backed parity entries",
	})

	for _, rel := range []string{
		"content/getting-started/first-run.md",
		"content/getting-started/configuration.md",
		"content/guides/telegram-bot.md",
		"content/reference/config.md",
	} {
		raw := readDoc(t, rel)
		if strings.Contains(raw, `bot_token = "`) || strings.Contains(raw, `app_token = "`) {
			t.Fatalf("%s should not document channel secrets inside config.toml examples", rel)
		}
	}
}

func TestPublicDocsExcludeHighRiskUpstreamSkill(t *testing.T) {
	if _, err := os.Stat(filepath.Join("content", "upstream-hermes", "user-guide", "skills", "godmode.md")); err == nil {
		t.Fatalf("public docs must not publish the upstream godmode skill page")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat upstream godmode page: %v", err)
	}

	raw := readDoc(t, "content/upstream-hermes/reference/skills-catalog.md")
	rejects := []string{
		"G0DM0D3",
		"Godmode Jailbreaking",
		"`godmode`",
		"red-teaming/godmode",
	}
	for _, reject := range rejects {
		if strings.Contains(raw, reject) {
			t.Fatalf("public skills catalog should not expose high-risk upstream skill token %q", reject)
		}
	}
}

func TestWhyGormesManifestoPageExistsAndCarriesApprovedSections(t *testing.T) {
	raw := readDoc(t, "content/why-gormes.md")
	wants := []string{
		`title: "Why Gormes"`,
		"## Operational Moat",
		"## Wire Doctor",
		"## Chaos Resilience",
		"## Surgical Architecture",
		"thundering herd",
		"Tool Registry",
		"Telegram Scout",
	}
	for _, want := range wants {
		if !strings.Contains(raw, want) {
			t.Fatalf("why-gormes page is missing %q", want)
		}
	}
}

func TestAICutoverDocsExistAndCarryExpectedTitles(t *testing.T) {
	spec := readDoc(t, "superpowers/specs/2026-04-19-gormes-ai-cutover-design.md")
	if !strings.Contains(spec, "Gormes.ai Hard Cutover Design Spec") {
		t.Fatalf("cutover spec missing its title")
	}

	plan := readDoc(t, "superpowers/plans/2026-04-19-gormes-ai-cutover.md")
	if !strings.Contains(plan, "Gormes.ai Hard Cutover Implementation Plan") {
		t.Fatalf("cutover plan missing its title")
	}
}

func TestSpecIndexAndPhase2SpecsCrossLink(t *testing.T) {
	indexRaw := readDoc(t, "superpowers/specs/README.md")
	indexWants := []string{
		"# Gormes Specs Index",
		"2026-04-19-gormes-doc-sync-manifesto-design.md",
		"2026-04-19-gormes-phase2-tools-design.md",
		"2026-04-19-gormes-phase2b-telegram.md",
		"2026-04-19-gormes-phase2c-persistence-design.md",
		"../../ARCH_PLAN.md",
	}
	for _, want := range indexWants {
		if !strings.Contains(indexRaw, want) {
			t.Fatalf("spec index is missing %q", want)
		}
	}

	toolsRaw := readDoc(t, "superpowers/specs/2026-04-19-gormes-phase2-tools-design.md")
	telegramRaw := readDoc(t, "superpowers/specs/2026-04-19-gormes-phase2b-telegram.md")
	for _, raw := range []string{toolsRaw, telegramRaw} {
		for _, want := range []string{
			"## Related Documents",
			"../../ARCH_PLAN.md",
			"README.md",
		} {
			if !strings.Contains(raw, want) {
				t.Fatalf("phase-2 spec is missing %q", want)
			}
		}
	}

	if !strings.Contains(toolsRaw, "2026-04-19-gormes-phase2b-telegram.md") {
		t.Fatalf("phase2 tools spec should link to the telegram spec")
	}
	if !strings.Contains(telegramRaw, "2026-04-19-gormes-phase2-tools-design.md") {
		t.Fatalf("telegram spec should link to the tool registry spec")
	}
}

func TestArchPlanTracksShippedPhase2Ledger(t *testing.T) {
	// ARCH_PLAN.md is now a stub (Task 2 of docs redesign).
	// Content has moved to content/building-gormes/architecture_plan/.
	stub := readDoc(t, "ARCH_PLAN.md")
	if !strings.Contains(stub, "content/building-gormes/architecture_plan/") {
		t.Fatalf("ARCH_PLAN.md stub should reference the new split location")
	}

	// Verify the Phase 2 ledger content lives in the split page.
	phase2 := readDoc(t, "content/building-gormes/architecture_plan/phase-2-gateway.md")
	wants := []string{
		"| Phase 2.A — Tool Registry | ✅ complete |",
		"| Phase 2.B.1 — Telegram Scout | ✅ complete |",
		"| Phase 2.C — Thin Mapping Persistence | ✅ complete |",
		"Phase 2.C is intentionally not Phase 3.",
		"Python still owns transcript memory",
	}
	for _, want := range wants {
		if !strings.Contains(phase2, want) {
			t.Fatalf("phase-2-gateway.md is missing %q", want)
		}
	}

	// Verify the roadmap overview lives in the _index.
	index := readDoc(t, "content/building-gormes/architecture_plan/_index.md")
	indexWants := []string{
		"**Public site:** https://gormes.ai",
		"Phase 3 — The Black Box (Memory)",
	}
	for _, want := range indexWants {
		if !strings.Contains(index, want) {
			t.Fatalf("architecture_plan/_index.md is missing %q", want)
		}
	}
}

func TestLandingPageDesignDocDescribesCurrentGormesAIModule(t *testing.T) {
	raw := readDoc(t, "superpowers/specs/2026-04-19-gormes-landing-page-design.md")
	wants := []string{
		"# Gormes.ai Landing Page Design Spec",
		"`www.gormes.ai`",
		"The Agent That GOes With You.",
		"Phase 1 uses your existing Hermes backend.",
		"Pure single-binary Go arrives later in the roadmap.",
		"Visual Direction",
		"Roadmap Block",
	}
	for _, want := range wants {
		if !strings.Contains(raw, want) {
			t.Fatalf("design doc is missing %q", want)
		}
	}

	if hasStaleGormesIOReference(raw) {
		t.Fatalf("design doc should not contain stale .io identity")
	}
}

func TestLandingPagePlanDocDocumentsCurrentAICutoverImplementation(t *testing.T) {
	raw := readDoc(t, "superpowers/plans/2026-04-19-gormes-landing-page.md")
	wants := []string{
		"# Gormes.ai Landing Page Implementation Plan",
		"webpages/landing/src/pages/index.astro",
		"webpages/landing/src/data/landing.js",
		"webpages/landing/src/styles/global.css",
		"webpages/landing/scripts/sync-assets.mjs",
		"webpages/landing/legacy/go-renderer/",
		"webpages/landing/README.md",
		"webpages/landing/tests/home.spec.mjs",
		"go test ./webpages/docs",
		"npm run build",
		"npm run test:e2e",
	}
	for _, want := range wants {
		if !strings.Contains(raw, want) {
			t.Fatalf("landing-page plan doc for the upcoming .ai cutover is missing %q", want)
		}
	}

	if hasStaleGormesIOReference(raw) {
		t.Fatalf("landing-page plan doc should not contain stale .io identity")
	}

	for _, rel := range []string{
		"../landing/src/data/landing.js",
		"../landing/src/pages/index.astro",
		"../landing/src/styles/global.css",
		"../landing/src/data/progress.json",
		"../landing/legacy/go-renderer/internal/site/content.go",
		"../landing/legacy/go-renderer/internal/site/assets.go",
		"../landing/tests/home.spec.mjs",
	} {
		if _, err := os.Stat(filepath.Join(".", rel)); err != nil {
			t.Fatalf("expected implementation file %s to exist: %v", rel, err)
		}
	}

	script := readPlaywrightE2EScript(t, "../landing/package.json")
	if !strings.Contains(script, "playwright") {
		t.Fatalf("www.gormes.ai package.json test:e2e script should reference Playwright")
	}
}

func TestReadmeDocumentsDoctorAndArchitecturalEdge(t *testing.T) {
	raw := readDoc(t, "../../README.md")
	// Assert stable README invariants only — things that should not drift
	// across phase boundaries. Binary size, Go minor version, and specific
	// architectural terms (coalescing mailbox ms, Route-B) are too brittle;
	// they belong in docs/ARCH_PLAN.md or per-phase specs, not README.
	// Assert stable invariants of the repo-root README.
	// Brittle assertions removed:
	//   - "7.9 MB static binary" / specific sizes — drifts with each phase
	//   - "Go 1.22+" — drifts with Go toolchain bumps (now 1.25+)
	//   - Specific phase spec links — drift with phase progression
	wants := []string{
		"single static binary",
		"Go-native runtime",
		"gormes doctor",
		"Hermes-Agent, with upstream Git history preserved for attribution",
	}
	for _, want := range wants {
		if !strings.Contains(raw, want) {
			t.Fatalf("README is missing %q", want)
		}
	}
}

func TestArchitectureIndexCrossLinksPrePhase4Gate(t *testing.T) {
	raw := readDoc(t, "content/building-gormes/architecture_plan/_index.md")
	wants := []string{
		"## Phase 4 Entry Gate",
		"Pre-Phase-4 E2E Gate",
		"phase-3-memory",
		"phase-4-brain-transplant",
	}
	for _, want := range wants {
		if !strings.Contains(raw, want) {
			t.Fatalf("architecture index is missing %q", want)
		}
	}
}

func TestGatewayCoreDocReflectsSharedPhase2Surface(t *testing.T) {
	raw := readDoc(t, "content/building-gormes/core-systems/gateway.md")
	wants := []string{
		"Discord adapter",
		"Slack Socket Mode bot",
		"slash-command registry",
		"SessionContext prompt injection",
		"BOOT.md startup hook",
		"Gateway stream consumer",
	}
	for _, want := range wants {
		if !strings.Contains(raw, want) {
			t.Fatalf("gateway core-systems doc is missing %q", want)
		}
	}
}

func TestMemoryDocsReflectShippedOperatorSurfacesAndGonchoNaming(t *testing.T) {
	core := readDoc(t, "content/building-gormes/core-systems/memory.md")
	for _, want := range []string{
		"Tool audit JSONL",
		"Transcript export",
		"gormes memory status",
	} {
		if !strings.Contains(core, want) {
			t.Fatalf("memory core-systems doc is missing %q", want)
		}
	}

	phase3 := readDoc(t, "content/building-gormes/architecture_plan/phase-3-memory.md")
	for _, want := range []string{
		"GONCHO",
		"Honcho-compatible interfaces",
		"| 3.E.2 — Tool Execution Audit Log | ✅ shipped | P0 |",
		"| 3.E.3 — Transcript Export Command | ✅ shipped | P2 |",
		"| 3.E.4 — Extraction State Visibility | ✅ shipped | P1 |",
	} {
		if !strings.Contains(phase3, want) {
			t.Fatalf("phase-3-memory doc is missing %q", want)
		}
	}
}

func TestMirrorStrategyReflectsShippedAuditAndOperatorSurfaces(t *testing.T) {
	raw := readDoc(t, "content/building-gormes/architecture_plan/mirror-strategy.md")
	wants := []string{
		"✅ **Shipped in Gormes (3.E.2)**",
		"✅ **Shipped in Gormes (3.E.3)**",
		"Phase 2.D",
		"Phase 2.G",
		"Phase 2.F.2",
	}
	for _, want := range wants {
		if !strings.Contains(raw, want) {
			t.Fatalf("mirror strategy doc is missing %q", want)
		}
	}
	for _, reject := range []string{
		"Gap: no tool audit trail",
		"Cron not yet implemented in Gormes (Phase 4)",
		"Skills not yet implemented (Phase 5)",
		"Boot hooks not yet implemented",
	} {
		if strings.Contains(raw, reject) {
			t.Fatalf("mirror strategy doc still contains stale claim %q", reject)
		}
	}
}

func TestSubsystemInventoryReflectsShippedPhase2AndPhase3Reality(t *testing.T) {
	raw := readDoc(t, "content/building-gormes/architecture_plan/subsystem-inventory.md")
	wants := []string{
		"typed session-context prompt injection landed",
		"typed delivery-target parsing landed",
		"deterministic frame fan-out landed",
		"Tool execution audit log | None (exceeds Hermes) | 3.E.2 | ✅ shipped",
		"Transcript export command | None (exceeds Hermes; Hermes has no text export) | 3.E.3 | ✅ shipped",
		"Extraction state visibility | None (debug visibility) | 3.E.4 | ✅ shipped",
		"Memory decay | None (Gormes-original) | 3.E.6 | ✅ shipped",
	}
	for _, want := range wants {
		if !strings.Contains(raw, want) {
			t.Fatalf("subsystem inventory is missing %q", want)
		}
	}
}

func TestPhase3IdentityLineagePlanIsLinkedAndContractsAreDocumented(t *testing.T) {
	plan := readDoc(t, "superpowers/plans/2026-04-22-gormes-phase3-identity-lineage-plan.md")
	for _, want := range []string{
		"# Phase 3 Identity + Lineage Implementation Plan",
		"user_id > chat_id > session_id",
		"same-chat default",
		"opt-in cross-chat",
		"parent_session_id",
		"source-filtered session search",
	} {
		if !strings.Contains(plan, want) {
			t.Fatalf("identity/lineage plan is missing %q", want)
		}
	}

	index := readDoc(t, "content/building-gormes/architecture_plan/_index.md")
	if !strings.Contains(index, "2026-04-22-gormes-phase3-identity-lineage-plan") {
		t.Fatalf("architecture index is missing the phase 3 identity/lineage plan link")
	}

	phase3 := readDoc(t, "content/building-gormes/architecture_plan/phase-3-memory.md")
	for _, want := range []string{
		"2026-04-22-gormes-phase3-identity-lineage-plan",
		"user_id > chat_id > session_id",
		"same-chat default, opt-in cross-chat",
		"parent_session_id",
	} {
		if !strings.Contains(phase3, want) {
			t.Fatalf("phase-3-memory doc is missing %q", want)
		}
	}

	core := readDoc(t, "content/building-gormes/core-systems/memory.md")
	for _, want := range []string{
		"GONCHO identity hierarchy",
		"same-chat by default",
		"parent_session_id",
	} {
		if !strings.Contains(core, want) {
			t.Fatalf("memory core-systems doc is missing %q", want)
		}
	}
}

func TestPhase3IdentityLineageExecutionPlanIsLinkedAndSequenced(t *testing.T) {
	plan := readDoc(t, "superpowers/plans/2026-04-22-gormes-phase3-identity-lineage-execution-plan.md")
	for _, want := range []string{
		"# Phase 3 Identity + Lineage Execution Plan",
		"Relationship last_seen tracking",
		"Cross-chat entity merge + recall fence",
		"parent_session_id lineage for compression splits",
		"Source-filtered FTS/session search across chats",
		"same-chat default",
		"opt-in cross-chat",
		"Rollback and failure containment",
		"go test ./internal/memory ./internal/session ./internal/goncho -count=1",
	} {
		if !strings.Contains(plan, want) {
			t.Fatalf("identity/lineage execution plan is missing %q", want)
		}
	}

	index := readDoc(t, "content/building-gormes/architecture_plan/_index.md")
	if !strings.Contains(index, "2026-04-22-gormes-phase3-identity-lineage-execution-plan") {
		t.Fatalf("architecture index is missing the phase 3 identity/lineage execution plan link")
	}

	phase3 := readDoc(t, "content/building-gormes/architecture_plan/phase-3-memory.md")
	for _, want := range []string{
		"2026-04-22-gormes-phase3-identity-lineage-execution-plan",
		"3.E.7 SillyTavern persona/group-chat mapping fixtures -> 3.E.7 operator evidence -> 3.E.8 parent_session_id -> 3.E.8 lineage-aware hits/evidence",
	} {
		if !strings.Contains(phase3, want) {
			t.Fatalf("phase-3-memory doc is missing %q", want)
		}
	}
}

func readDoc(t *testing.T, rel string) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(".", rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(raw)
}

func runGormesHelp(t *testing.T, args ...string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", append([]string{"run", "./cmd/gormes"}, args...)...)
	cmd.Dir = filepath.Join("..", "..")
	cmd.Env = append(os.Environ(), "GORMES_HOME="+t.TempDir())

	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("go run ./cmd/gormes %s timed out", strings.Join(args, " "))
	}
	if err != nil {
		t.Fatalf("go run ./cmd/gormes %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func assertContainsAll(t *testing.T, label, raw string, wants []string) {
	t.Helper()

	for _, want := range wants {
		if !strings.Contains(raw, want) {
			t.Fatalf("%s is missing %q", label, want)
		}
	}
}

func readPlaywrightE2EScript(t *testing.T, rel string) string {
	t.Helper()

	type packageJSON struct {
		Scripts map[string]string `json:"scripts"`
	}

	var pkg packageJSON
	if err := json.Unmarshal([]byte(readDoc(t, rel)), &pkg); err != nil {
		t.Fatalf("parse %s as json: %v", rel, err)
	}

	script, ok := pkg.Scripts["test:e2e"]
	if !ok {
		t.Fatalf("%s does not define scripts.test:e2e", rel)
	}

	return script
}

func hasStaleGormesIOReference(raw string) bool {
	return staleGormesIORefPattern.FindStringIndex(raw) != nil
}
