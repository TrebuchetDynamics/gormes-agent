package docs_test

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

const (
	docsContentRoot = "./content"
)

type sourceDocsRootCandidate struct {
	label string
	path  string
}

var (
	bannedPatterns = []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{"docusaurus-admonition", regexp.MustCompile(`(?m)^:::`)},
		{"jsx-comment", regexp.MustCompile(`\{/\*.*\*/\}`)},
		{"jsx-class-name", regexp.MustCompile(`className=`)},
		{"root-relative-link", regexp.MustCompile(`\]\(/[^)]+\)`)},
		{"raw-react-component", regexp.MustCompile(`(?m)^<[A-Z][A-Za-z0-9]*(?:\s|/|>)`)},
	}
	markdownLinkPattern = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
)

var upstreamHermesSupplementPages = map[string]struct{}{
	"upstream-hermes/good-and-bad.md":     {},
	"upstream-hermes/gormes-takeaways.md": {},
	"upstream-hermes/source-study.md":     {},
}

var (
	gatewayDonorMapRequiredHeadings = []string{
		"## Status",
		"## Why This Adapter Is Reusable",
		"## Picoclaw Donor Files",
		"## What To Copy vs What To Rebuild",
		"## Gormes Mapping",
		"## Implementation Notes",
		"## Risks / Mismatches",
		"## Port Order Recommendation",
		"## Code References",
	}
	gatewayDonorMapAllowedRecommendations = map[string]struct{}{
		"copy candidate":     {},
		"adapt pattern only": {},
		"not worth reusing":  {},
	}
	gatewayDonorMapPinnedProvenance = []string{
		"<picoclaw donor repo>",
		"6421f146a99df1bebcd4b1ca8de2a289dfca3622",
		"https://github.com/sipeed/picoclaw",
		"relative to that donor root, not relative to the Gormes repo",
	}
	gatewayDonorMapHubRowPattern         = regexp.MustCompile(`(?m)^\| ([^|]+) \| ` + "`" + `([^` + "`" + `]+)` + "`" + ` \| [^|]+ \| \[([^\]]+)\]\(\./([^/]+)/\) \|$`)
	gatewayDonorMapRecommendationPattern = regexp.MustCompile("Recommendation: `([^`]+)`\\.")
)

var targets = []string{
	"ARCH_PLAN.md",
	"THEORETICAL_ADVANTAGES_GORMES_HERMES.md",
	"superpowers/specs/2026-04-18-gormes-frontend-adapter-design.md",
	"superpowers/plans/2026-04-18-gormes-phase1-frontend-adapter.md",
	"superpowers/specs/2026-04-19-gormes-landing-page-design.md",
	"superpowers/plans/2026-04-19-gormes-landing-page.md",
	"superpowers/specs/2026-04-19-gormes-ai-cutover-design.md",
	"superpowers/plans/2026-04-19-gormes-ai-cutover.md",
	"superpowers/specs/2026-04-19-gormes-doc-sync-manifesto-design.md",
	"superpowers/plans/2026-04-19-gormes-doc-sync-manifesto.md",
	"superpowers/specs/2026-04-19-gormes-phase2c-persistence-design.md",
	"superpowers/plans/2026-04-19-gormes-phase2c-persistence.md",
}

var nativeDocsPages = map[string]struct{}{
	"_index.md":                                                            {},
	"why-gormes.md":                                                        {},
	"building-gormes/_index.md":                                            {},
	"building-gormes/contract-readiness.md":                                {},
	"building-gormes/builder-loop/_index.md":                               {},
	"building-gormes/builder-loop/builder-loop-handoff.md":                 {},
	"building-gormes/builder-loop/agent-queue.md":                          {},
	"building-gormes/builder-loop/next-slices.md":                          {},
	"building-gormes/builder-loop/blocked-slices.md":                       {},
	"building-gormes/builder-loop/umbrella-cleanup.md":                     {},
	"building-gormes/builder-loop/progress-schema.md":                      {},
	"building-gormes/upstream-lessons.md":                                  {},
	"building-gormes/gateway-donor-map/_index.md":                          {},
	"building-gormes/gateway-donor-map/shared-adapter-patterns.md":         {},
	"building-gormes/porting-a-subsystem.md":                               {},
	"building-gormes/testing.md":                                           {},
	"building-gormes/what-hermes-gets-wrong.md":                            {},
	"building-gormes/gateway-donor-map/telegram.md":                        {},
	"building-gormes/gateway-donor-map/discord.md":                         {},
	"building-gormes/gateway-donor-map/slack.md":                           {},
	"building-gormes/gateway-donor-map/whatsapp.md":                        {},
	"building-gormes/gateway-donor-map/matrix.md":                          {},
	"building-gormes/gateway-donor-map/irc.md":                             {},
	"building-gormes/gateway-donor-map/line.md":                            {},
	"building-gormes/gateway-donor-map/onebot.md":                          {},
	"building-gormes/gateway-donor-map/qq.md":                              {},
	"building-gormes/gateway-donor-map/wecom.md":                           {},
	"building-gormes/gateway-donor-map/weixin.md":                          {},
	"building-gormes/gateway-donor-map/feishu.md":                          {},
	"building-gormes/gateway-donor-map/dingtalk.md":                        {},
	"building-gormes/gateway-donor-map/vk.md":                              {},
	"building-gormes/gateway-donor-map/webhook.md":                         {},
	"building-gormes/goncho_honcho_memory/_index.md":                       {},
	"building-gormes/goncho_honcho_memory/01-prompts.md":                   {},
	"building-gormes/goncho_honcho_memory/02-tool-schemas.md":              {},
	"building-gormes/architecture_plan/_index.md":                          {},
	"building-gormes/architecture_plan/hermes-gormes-contract-pairings.md": {},
	"building-gormes/architecture_plan/phase-1-dashboard.md":               {},
	"building-gormes/architecture_plan/phase-2-gateway.md":                 {},
	"building-gormes/architecture_plan/phase-3-memory.md":                  {},
	"building-gormes/architecture_plan/phase-4-brain-transplant.md":        {},
	"building-gormes/architecture_plan/phase-5-final-purge.md":             {},
	"building-gormes/architecture_plan/phase-6-learning-loop.md":           {},
	"building-gormes/architecture_plan/subsystem-inventory.md":             {},
	"building-gormes/architecture_plan/mirror-strategy.md":                 {},
	"building-gormes/architecture_plan/technology-radar.md":                {},
	"building-gormes/architecture_plan/procfile-process-managers.md":       {},
	"building-gormes/architecture_plan/boundaries.md":                      {},
	"building-gormes/architecture_plan/why-go.md":                          {},
	"building-gormes/core-systems/_index.md":                               {},
	"building-gormes/core-systems/gateway.md":                              {},
	"building-gormes/core-systems/learning-loop.md":                        {},
	"building-gormes/core-systems/memory.md":                               {},
	"building-gormes/core-systems/tool-execution.md":                       {},
	"install/_index.md":                                                    {},
	"install/linux-macos.md":                                               {},
	"install/windows.md":                                                   {},
	"install/from-source.md":                                               {},
	"start-here/_index.md":                                                 {},
	"troubleshooting/_index.md":                                            {},
	"troubleshooting/doctor.md":                                            {},
	"troubleshooting/common-errors.md":                                     {},
	"troubleshooting/logs.md":                                              {},
}

func TestMirroredDocsCoverage(t *testing.T) {
	sourceRoot, sourceLabel := selectedSourceDocsRoot(t)
	sourcePaths := collectSourceDocs(t, sourceRoot)
	contentPaths := collectContentDocs(t)

	expected := make(map[string]struct{}, len(sourcePaths))
	for _, rel := range sourcePaths {
		expected[mapSourceToContent(rel)] = struct{}{}
	}

	seen := make(map[string]struct{}, len(contentPaths))
	for _, rel := range contentPaths {
		if !strings.HasPrefix(rel, "upstream-hermes/") {
			continue
		}
		seen[rel] = struct{}{}
	}

	var missing []string
	for rel := range expected {
		if _, ok := seen[rel]; !ok {
			missing = append(missing, rel)
		}
	}
	sort.Strings(missing)

	var unexpected []string
	for rel := range seen {
		if _, ok := expected[rel]; ok {
			continue
		}
		if _, ok := upstreamHermesSupplementPages[rel]; ok {
			continue
		}
		unexpected = append(unexpected, rel)
	}
	sort.Strings(unexpected)

	if len(missing) > 0 || len(unexpected) > 0 {
		t.Fatalf(
			"mirrored docs coverage mismatch using source root %s (%s): missing %d [%s]; unexpected %d [%s]",
			sourceLabel,
			sourceRoot,
			len(missing),
			strings.Join(firstStrings(missing, 12), ", "),
			len(unexpected),
			strings.Join(firstStrings(unexpected, 12), ", "),
		)
	}
}

func TestProgressJsonHasSingleCanonicalDocsCopy(t *testing.T) {
	canonical := filepath.Join(docsContentRoot, "building-gormes", "architecture_plan", "progress.json")
	if _, err := os.Stat(canonical); err != nil {
		t.Fatalf("canonical progress.json missing: %v", err)
	}
	// "Single canonical" means loadable as the one logical backlog regardless
	// of on-disk layout: internal/progress.Load handles a monolithic file or a
	// module-keyed split directory transparently (module-split umbrella C5/C5c).
	if prog, err := progress.Load(canonical); err != nil {
		t.Fatalf("canonical progress.json not loadable as the logical backlog: %v", err)
	} else if prog == nil || len(prog.Phases) == 0 {
		t.Fatalf("canonical progress.json loaded empty: prog=%v", prog)
	}

	duplicate := filepath.Join("data", "progress.json")
	if _, err := os.Stat(duplicate); err == nil {
		t.Fatalf("non-canonical progress copy must not exist: %s", duplicate)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat duplicate progress.json: %v", err)
	}
}

func TestRootWWWGormesAIPathDoesNotExist(t *testing.T) {
	legacyRoot := filepath.Join("..", "www.gormes.ai")
	if _, err := os.Lstat(legacyRoot); err == nil {
		t.Fatalf("root www.gormes.ai path must not exist; active landing site lives under webpages/landing")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat root www.gormes.ai path: %v", err)
	}
}

func TestDocsContentRendersViaGoldmark(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(extension.GFM, extension.Table, extension.Strikethrough))

	for _, rel := range collectContentDocs(t) {
		t.Run(rel, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(docsContentRoot, rel))
			if err != nil {
				t.Fatalf("read %s: %v", rel, err)
			}

			var buf bytes.Buffer
			if err := md.Convert(raw, &buf); err != nil {
				t.Fatalf("goldmark render %s: %v", rel, err)
			}
			if buf.Len() == 0 {
				t.Fatalf("goldmark produced empty output for %s", rel)
			}
		})
	}
}

func TestDocsContentAvoidsPortabilityHazards(t *testing.T) {
	for _, rel := range collectContentDocs(t) {
		t.Run(rel, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(docsContentRoot, rel))
			if err != nil {
				t.Fatalf("read %s: %v", rel, err)
			}

			scanForHazards(t, rel, string(raw))
		})
	}
}

func TestDocsInternalLinksResolve(t *testing.T) {
	for _, rel := range collectContentDocs(t) {
		raw, err := os.ReadFile(filepath.Join(docsContentRoot, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}

		inFence := false
		for _, line := range strings.Split(string(raw), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				inFence = !inFence
				continue
			}
			if inFence {
				continue
			}

			linkScanLine := stripInlineCodeSpans(line)
			for _, match := range markdownLinkPattern.FindAllStringSubmatch(linkScanLine, -1) {
				if strings.HasPrefix(strings.TrimSpace(linkScanLine), "![") {
					continue
				}
				link := strings.TrimSpace(match[1])
				if link == "" || isExternalLink(link) || strings.HasPrefix(link, "#") {
					continue
				}
				if strings.HasPrefix(link, "/") {
					t.Fatalf("%s: root-relative internal link %q", rel, link)
				}
				if err := resolveContentLink(rel, link); err != nil {
					if strings.HasPrefix(rel, "upstream-hermes/") {
						continue
					}
					t.Fatalf("%s: unresolved internal link %q: %v", rel, link, err)
				}
			}
		}
	}
}

func TestGatewayDonorMapInvariants(t *testing.T) {
	const donorMapDir = "content/building-gormes/gateway-donor-map"

	entries, err := os.ReadDir(donorMapDir)
	if err != nil {
		t.Fatalf("read donor map dir: %v", err)
	}

	dossierRecommendations := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		name := entry.Name()
		path := filepath.Join(donorMapDir, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		content := string(raw)

		switch name {
		case "_index.md", "shared-adapter-patterns.md":
			for _, want := range gatewayDonorMapPinnedProvenance {
				if !strings.Contains(content, want) {
					t.Fatalf("%s missing pinned provenance string %q", path, want)
				}
			}
			continue
		}

		for _, heading := range gatewayDonorMapRequiredHeadings {
			if !strings.Contains(content, heading) {
				t.Fatalf("%s missing heading %q", path, heading)
			}
		}

		match := gatewayDonorMapRecommendationPattern.FindStringSubmatch(content)
		if len(match) != 2 {
			t.Fatalf("%s missing final recommendation label", path)
		}
		recommendation := match[1]
		if _, ok := gatewayDonorMapAllowedRecommendations[recommendation]; !ok {
			t.Fatalf("%s has unsupported recommendation %q", path, recommendation)
		}

		dossierRecommendations[strings.TrimSuffix(name, ".md")] = recommendation
	}

	indexPath := filepath.Join(donorMapDir, "_index.md")
	indexRaw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read %s: %v", indexPath, err)
	}

	matches := gatewayDonorMapHubRowPattern.FindAllStringSubmatch(string(indexRaw), -1)
	if len(matches) == 0 {
		t.Fatalf("%s missing triage table rows", indexPath)
	}

	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		channel := match[1]
		recommendation := match[2]
		label := match[3]
		slug := match[4]

		if channel != label {
			t.Fatalf("%s row channel %q does not match dossier label %q", indexPath, channel, label)
		}
		want, ok := dossierRecommendations[slug]
		if !ok {
			t.Fatalf("%s row for %q points to unknown dossier %q", indexPath, channel, slug)
		}
		if recommendation != want {
			t.Fatalf("%s row for %q has recommendation %q, dossier has %q", indexPath, channel, recommendation, want)
		}
		seen[slug] = struct{}{}
	}

	if len(seen) != len(dossierRecommendations) {
		var missing []string
		for slug := range dossierRecommendations {
			if _, ok := seen[slug]; !ok {
				missing = append(missing, slug)
			}
		}
		sort.Strings(missing)
		t.Fatalf("%s missing triage rows for dossiers: %s", indexPath, strings.Join(missing, ", "))
	}
}

func TestFirstRunProofPathDocumentsOfflineDoctorBeforeCredentials(t *testing.T) {
	pages := map[string]string{
		"README":        snippetAfter(t, "README", readDoc(t, "../../README.md"), "After installation:"),
		"start-here":    snippetAfter(t, "start-here", readDoc(t, "content/start-here/_index.md"), "## Fast path"),
		"install":       snippetAfter(t, "install", readDoc(t, "content/install/_index.md"), "## First-run proof order"),
		"doctor-recipe": snippetAfter(t, "doctor-recipe", readDoc(t, "content/recipes/doctor-offline.md"), "## First-run proof order"),
		"landing":       snippetAfter(t, "landing", readDoc(t, "../landing/src/data/landing.js"), "installIntro:"),
	}

	for name, raw := range pages {
		assertOrdered(t, name, raw,
			"gormes version",
			"gormes doctor --offline",
			"gormes setup",
		)
		for _, want := range []string{
			"no pip",
			"no venv",
			"no Docker daemon",
			"before credentials",
		} {
			if !strings.Contains(strings.ToLower(raw), strings.ToLower(want)) {
				t.Fatalf("%s missing first-run proof wording %q", name, want)
			}
		}
	}
}

func TestChannelCapabilityMatrixDocs(t *testing.T) {
	channels := readDoc(t, "content/cli/channels.md")
	for _, want := range []string{
		"## Capability Matrix",
		"Runtime-ready",
		"Fixture-backed",
		"Planned",
		"gormes channels capabilities",
		"gormes gateway status --json",
		"[Telegram](../../operate/telegram-bot/)",
		"[Discord](../gateway/)",
		"[Slack](../slack/)",
		"WhatsApp",
	} {
		if !strings.Contains(channels, want) {
			t.Fatalf("channels docs missing %q", want)
		}
	}
	for _, forbidden := range []string{"50+ integrations", "fully runtime-ready"} {
		if strings.Contains(channels, forbidden) {
			t.Fatalf("channels docs overclaim %q", forbidden)
		}
	}

	recipe := readDoc(t, "content/operate/multi-channel-gateway.md")
	assertOrdered(t, "multi-channel recipe", recipe,
		"gormes channels capabilities",
		"gormes gateway status --json",
	)
	for _, want := range []string{"Runtime-ready", "Fixture-backed", "Planned"} {
		if !strings.Contains(recipe, want) {
			t.Fatalf("multi-channel recipe missing readiness label %q", want)
		}
	}
}

func TestLearningLoopOperatorProofDocs(t *testing.T) {
	learningLoop := readDoc(t, "content/building-gormes/core-systems/learning-loop.md")
	for _, want := range []string{
		"## Operator Proof",
		"Task evidence",
		"Skill creation or improvement",
		"Memory recall",
		"Curator maintenance",
		"Repeated-task proof",
		"gormes skills list",
		"gormes memory status",
		"gormes curator status",
		"deterministic local proof",
		"operator review",
	} {
		if !strings.Contains(learningLoop, want) {
			t.Fatalf("learning-loop proof docs missing %q", want)
		}
	}

	whyGormes := readDoc(t, "content/why-gormes.md")
	if !strings.Contains(whyGormes, "building-gormes/core-systems/learning-loop/") ||
		!strings.Contains(whyGormes, "learning loop") {
		t.Fatalf("why-gormes page must link to the learning-loop proof without making it the hero")
	}

	for _, page := range []string{
		"content/cli/curator.md",
		"content/cli/skills.md",
		"content/cli/memory.md",
	} {
		raw := readDoc(t, page)
		for _, want := range []string{"learning loop", "building-gormes/core-systems/learning-loop/"} {
			if !strings.Contains(raw, want) {
				t.Fatalf("%s missing learning-loop proof link text %q", page, want)
			}
		}
	}
}

func selectedSourceDocsRoot(t *testing.T) (string, string) {
	t.Helper()

	repoRoot := findRepoRoot(t)
	candidates := []sourceDocsRootCandidate{
		{label: "./hermes-agent/website/docs", path: filepath.Join(repoRoot, "hermes-agent", "website", "docs")},
		{label: "../hermes-agent/website/docs", path: filepath.Join(repoRoot, "..", "hermes-agent", "website", "docs")},
		{label: "references/hermes-agent/website/docs", path: filepath.Join(repoRoot, "references", "hermes-agent", "website", "docs")},
		{label: "../../../website/docs", path: filepath.Clean(filepath.Join("..", "..", "..", "website", "docs"))},
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate.path); err == nil && info.IsDir() {
			return candidate.path, candidate.label
		}
	}

	var labels []string
	for _, candidate := range candidates {
		labels = append(labels, candidate.label)
	}
	t.Skipf("upstream website docs source not present in standalone Gormes repo; checked %s", strings.Join(labels, ", "))
	return "", ""
}

func findRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate repository root from %s", dir)
		}
		dir = parent
	}
}

func collectSourceDocs(t *testing.T, sourceDocsRoot string) []string {
	t.Helper()

	var paths []string
	err := filepath.WalkDir(sourceDocsRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(sourceDocsRoot, path)
		if err != nil {
			return err
		}
		if !isMirroredSourceFile(rel) {
			return nil
		}
		if isExcludedUpstreamMirrorSource(rel) {
			return nil
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("walk source docs: %v", err)
	}
	sort.Strings(paths)

	return paths
}

func collectContentDocs(t *testing.T) []string {
	t.Helper()

	var paths []string
	err := filepath.WalkDir(docsContentRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		rel, err := filepath.Rel(docsContentRoot, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("walk docs content: %v", err)
	}
	sort.Strings(paths)

	return paths
}

func isMirroredSourceFile(rel string) bool {
	switch filepath.Base(rel) {
	case "_category_.json", "index.md":
		return true
	}
	switch filepath.Ext(rel) {
	case ".md", ".mdx":
		return true
	default:
		return false
	}
}

func isExcludedUpstreamMirrorSource(rel string) bool {
	switch filepath.ToSlash(rel) {
	case "user-guide/skills/godmode.md",
		"user-guide/skills/bundled/red-teaming/red-teaming-godmode.md":
		return true
	default:
		return false
	}
}

func mapSourceToContent(rel string) string {
	rel = filepath.ToSlash(rel)
	// Upstream docs mirror lives under content/upstream-hermes/.
	const mirrorPrefix = "upstream-hermes/"
	if rel == "index.md" {
		return mirrorPrefix + "_index.md"
	}
	if strings.HasSuffix(rel, "/index.md") {
		return mirrorPrefix + strings.TrimSuffix(rel, "index.md") + "_index.md"
	}
	if strings.HasSuffix(rel, "/_category_.json") {
		return mirrorPrefix + strings.TrimSuffix(rel, "_category_.json") + "_index.md"
	}
	if strings.HasSuffix(rel, ".mdx") {
		return mirrorPrefix + strings.TrimSuffix(rel, ".mdx") + ".md"
	}
	return mirrorPrefix + rel
}

func snippetAfter(t *testing.T, name, raw, marker string) string {
	t.Helper()

	idx := strings.Index(raw, marker)
	if idx < 0 {
		t.Fatalf("%s missing marker %q", name, marker)
	}
	return raw[idx:]
}

func assertOrdered(t *testing.T, name, raw string, needles ...string) {
	t.Helper()

	search := strings.ToLower(raw)
	last := -1
	for _, needle := range needles {
		idx := strings.Index(search, strings.ToLower(needle))
		if idx < 0 {
			t.Fatalf("%s missing %q", name, needle)
		}
		if idx <= last {
			t.Fatalf("%s has %q out of first-run order", name, needle)
		}
		last = idx
	}
}

func firstStrings(values []string, limit int) []string {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}

func stripInlineCodeSpans(line string) string {
	var b strings.Builder
	b.Grow(len(line))
	inCode := false
	for _, r := range line {
		if r == '`' {
			inCode = !inCode
			b.WriteRune(' ')
			continue
		}
		if inCode {
			b.WriteRune(' ')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func scanForHazards(t *testing.T, rel, raw string) {
	t.Helper()

	inFence := false
	for i, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		isImageLine := strings.HasPrefix(trimmed, "![")
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		for _, hazard := range bannedPatterns {
			if hazard.name == "root-relative-link" && isImageLine {
				continue
			}
			if hazard.pattern.MatchString(line) {
				t.Fatalf("%s:%d %s: %q", rel, i+1, hazard.name, line)
			}
		}
	}
}

func isExternalLink(link string) bool {
	switch {
	case strings.HasPrefix(link, "http://"),
		strings.HasPrefix(link, "https://"),
		strings.HasPrefix(link, "mailto:"),
		strings.HasPrefix(link, "tel:"),
		strings.HasPrefix(link, "ftp://"),
		strings.HasPrefix(link, "//"):
		return true
	default:
		return false
	}
}

func TestResolveContentLinkUsesRenderedLeafURLBase(t *testing.T) {
	sourceRel := "building-gormes/builder-loop/builder-loop-handoff.md"

	if err := resolveContentLink(sourceRel, "./agent-queue/"); !os.IsNotExist(err) {
		t.Fatalf("leaf-relative sibling link resolved from source directory; got %v", err)
	}
	if err := resolveContentLink(sourceRel, "../agent-queue/"); err != nil {
		t.Fatalf("rendered leaf-relative sibling link should resolve: %v", err)
	}
}

func resolveContentLink(sourceRel, link string) error {
	target := link
	if idx := strings.IndexAny(target, "?#"); idx >= 0 {
		target = target[:idx]
	}

	sourceDir := filepath.Dir(sourceRel)
	renderedDir := sourceDir
	if base := filepath.Base(sourceRel); base != "_index.md" && filepath.Ext(base) == ".md" {
		renderedDir = filepath.Join(sourceDir, strings.TrimSuffix(base, ".md"))
	}

	candidateRel := filepath.Clean(filepath.Join(renderedDir, target))
	candidate := filepath.Join(docsContentRoot, candidateRel)
	checks := []string{
		candidate + ".md",
		filepath.Join(candidate, "_index.md"),
		filepath.Join(candidate, "index.md"),
	}
	if strings.HasSuffix(target, "/") {
		checks = append([]string{filepath.Join(candidate, "_index.md")}, checks...)
	}

	for _, check := range checks {
		if _, err := os.Stat(check); err == nil {
			return nil
		}
	}

	return os.ErrNotExist
}

func TestAstroBuildProducesRenderedContent(t *testing.T) {
	if testing.Short() {
		t.Skip("skip Astro build in short mode")
	}

	dest := t.TempDir()
	runDocsAstroBuild(t, dest)

	// Starlight's filename-based routing preserves section directories:
	// content/upstream-hermes/user-guide/cli.md -> /upstream-hermes/user-guide/cli/.
	checks := map[string][]string{
		"index.html": {
			"Quickstart",
			"Operate",
			"Reference",
			"Concepts",
			"Build Gormes",
			"Archive &amp; Research",
		},
		filepath.Join("why-gormes", "index.html"): {
			"Operational Moat",
			"Wire Doctor",
			"Chaos Resilience",
			"Surgical Architecture",
		},
		filepath.Join("upstream-hermes", "user-guide", "cli", "index.html"): {
			"Stylized preview of the Hermes CLI layout",
			"The Hermes CLI banner, conversation stream, and fixed input prompt",
		},
		filepath.Join("upstream-hermes", "user-guide", "sessions", "index.html"): {
			"Stylized preview of the Previous Conversation recap panel",
			"Resume mode shows a compact recap panel",
		},
	}

	for rel, wants := range checks {
		raw, err := os.ReadFile(filepath.Join(dest, rel))
		if err != nil {
			t.Fatalf("read rendered %s: %v", rel, err)
		}
		content := string(raw)
		for _, want := range wants {
			if !strings.Contains(content, want) {
				t.Fatalf("rendered %s missing %q", rel, want)
			}
		}
	}
}
