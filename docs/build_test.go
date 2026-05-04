package docs_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestAstroBuild runs `npm run build` in a temp directory and asserts
// the full set of expected pages are emitted. Guards against:
//   - Starlight regressions (build fails silently)
//   - Broken front-matter (page doesn't render)
//   - Missing content files (section landing without children)
func TestAstroBuild(t *testing.T) {
	tmp := t.TempDir()
	runDocsAstroBuild(t, tmp)

	wantPages := []string{
		"index.html",
		"getting-started/index.html",
		"getting-started/installation/index.html",
		"getting-started/first-run/index.html",
		"getting-started/configuration/index.html",
		"getting-started/troubleshooting/index.html",
		"guides/index.html",
		"guides/gateway-operations/index.html",
		"guides/telegram-bot/index.html",
		"guides/provider-setup/index.html",
		"guides/web-tools/index.html",
		"guides/browser-cdp/index.html",
		"guides/debugging/index.html",
		"reference/index.html",
		"reference/cli/index.html",
		"reference/config/index.html",
		"reference/environment/index.html",
		"reference/providers/index.html",
		"reference/web-backends/index.html",
		"reference/paths-and-logs/index.html",
		"architecture/index.html",
		"architecture/runtime-model/index.html",
		"architecture/gateway-pipeline/index.html",
		"architecture/tool-execution/index.html",
		"architecture/memory-and-sessions/index.html",
		"architecture/hermes-parity/index.html",
		"development/index.html",
		"development/repo-layout/index.html",
		"development/testing/index.html",
		"development/parity-workflow/index.html",
		"parity/index.html",
		"parity/current-status/index.html",
		"parity/command-surface/index.html",
		"parity/roadmap/index.html",
		"using-gormes/index.html",
		"using-gormes/quickstart/index.html",
		"using-gormes/install/index.html",
		"using-gormes/tui-mode/index.html",
		"using-gormes/telegram-adapter/index.html",
		"using-gormes/configuration/index.html",
		"using-gormes/wire-doctor/index.html",
		"using-gormes/faq/index.html",
		"building-gormes/index.html",
		"building-gormes/core-systems/index.html",
		"building-gormes/core-systems/learning-loop/index.html",
		"building-gormes/core-systems/memory/index.html",
		"building-gormes/core-systems/tool-execution/index.html",
		"building-gormes/core-systems/gateway/index.html",
		"building-gormes/contract-readiness/index.html",
		"building-gormes/builder-loop/index.html",
		"building-gormes/builder-loop/builder-loop-handoff/index.html",
		"building-gormes/builder-loop/agent-queue/index.html",
		"building-gormes/builder-loop/next-slices/index.html",
		"building-gormes/builder-loop/blocked-slices/index.html",
		"building-gormes/builder-loop/umbrella-cleanup/index.html",
		"building-gormes/builder-loop/progress-schema/index.html",
		"building-gormes/autoloop-handoff/index.html",
		"building-gormes/agent-queue/index.html",
		"building-gormes/next-slices/index.html",
		"building-gormes/blocked-slices/index.html",
		"building-gormes/umbrella-cleanup/index.html",
		"building-gormes/progress-schema/index.html",
		"building-gormes/upstream-lessons/index.html",
		"building-gormes/what-hermes-gets-wrong/index.html",
		"building-gormes/porting-a-subsystem/index.html",
		"building-gormes/testing/index.html",
		"building-gormes/architecture_plan/index.html",
		"building-gormes/gateway-donor-map/index.html",
		"building-gormes/gateway-donor-map/shared-adapter-patterns/index.html",
		"building-gormes/gateway-donor-map/telegram/index.html",
		"building-gormes/gateway-donor-map/discord/index.html",
		"building-gormes/gateway-donor-map/slack/index.html",
		"building-gormes/gateway-donor-map/whatsapp/index.html",
		"building-gormes/gateway-donor-map/matrix/index.html",
		"building-gormes/gateway-donor-map/irc/index.html",
		"building-gormes/gateway-donor-map/line/index.html",
		"building-gormes/gateway-donor-map/onebot/index.html",
		"building-gormes/gateway-donor-map/qq/index.html",
		"building-gormes/gateway-donor-map/wecom/index.html",
		"building-gormes/gateway-donor-map/weixin/index.html",
		"building-gormes/gateway-donor-map/feishu/index.html",
		"building-gormes/gateway-donor-map/dingtalk/index.html",
		"building-gormes/gateway-donor-map/vk/index.html",
		"building-gormes/gateway-donor-map/webhook/index.html",
		"building-gormes/goncho_honcho_memory/index.html",
		"building-gormes/goncho_honcho_memory/01-prompts/index.html",
		"building-gormes/goncho_honcho_memory/02-tool-schemas/index.html",
		"building-gormes/architecture_plan/phase-1-dashboard/index.html",
		"building-gormes/architecture_plan/phase-2-gateway/index.html",
		"building-gormes/architecture_plan/phase-3-memory/index.html",
		"building-gormes/architecture_plan/phase-4-brain-transplant/index.html",
		"building-gormes/architecture_plan/phase-5-final-purge/index.html",
		"building-gormes/architecture_plan/phase-6-learning-loop/index.html",
		"building-gormes/architecture_plan/subsystem-inventory/index.html",
		"building-gormes/architecture_plan/mirror-strategy/index.html",
		"building-gormes/architecture_plan/technology-radar/index.html",
		"building-gormes/architecture_plan/boundaries/index.html",
		"building-gormes/architecture_plan/why-go/index.html",
		"upstream-hermes/index.html",
	}

	for _, p := range wantPages {
		full := filepath.Join(tmp, p)
		if _, err := os.Stat(full); err != nil {
			t.Errorf("expected built page missing: %s", p)
		}
	}
}

// TestAstroBuild_IndexHasSidebarSections asserts the rendered home page
// prioritizes user/operator docs before roadmap and upstream archives.
func TestAstroBuild_IndexHasSidebarSections(t *testing.T) {
	tmp := t.TempDir()
	runDocsAstroBuild(t, tmp)
	body, err := os.ReadFile(filepath.Join(tmp, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{
		"Getting Started",
		"Operate",
		"Using Gormes",
		"Reference",
		"Architecture",
		"Development",
		"Parity",
		"Building Gormes",
		`href="/getting-started/"`,
		`href="/guides/"`,
		`href="/using-gormes/"`,
		`href="/reference/"`,
		`href="/architecture/"`,
		`href="/development/"`,
		`href="/parity/"`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("built index.html missing %q", want)
		}
	}
}

func TestAstroBuild_IndexQuickstartUsesCurrentInstallCommand(t *testing.T) {
	tmp := t.TempDir()
	runDocsAstroBuild(t, tmp)

	body, err := os.ReadFile(filepath.Join(tmp, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)

	if !strings.Contains(text, "git clone https://github.com/TrebuchetDynamics/gormes-agent.git") {
		t.Fatalf("built index.html missing current install command")
	}
	if strings.Contains(text, "curl -fsSL https://gormes.ai/install.sh | sh") {
		t.Fatalf("built index.html still contains curl-pipe install command")
	}
	if strings.Contains(text, "brew install trebuchet/gormes") {
		t.Fatalf("built index.html still contains stale Homebrew install command")
	}
}

func TestAstroBuild_IndexUsesOperatorFirstDocsStructure(t *testing.T) {
	tmp := t.TempDir()
	runDocsAstroBuild(t, tmp)

	body, err := os.ReadFile(filepath.Join(tmp, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{
		"Gormes runs AI agents as one Go-native agent runtime.",
		"Start offline, prove the machine works",
		"What is Gormes?",
		"Go-native runtime",
		"Offline proof path",
		"What you can do today",
		"Browse sessions, config, skills, and logs",
		"What lives here?",
		"Start here",
		"Getting Started",
		"Roadmap &amp; Parity",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("built index.html missing operator-first docs token %q", want)
		}
	}
	for _, reject := range []string{
		"connect a Hermes backend",
		"Run Hermes Through a Go Operator Console",
		"curl -fsSL https://gormes.ai/install.sh | sh",
	} {
		if strings.Contains(text, reject) {
			t.Fatalf("built index.html contains stale token %q", reject)
		}
	}
}

func runDocsAstroBuild(t *testing.T, dest string) {
	t.Helper()
	if err := os.RemoveAll(filepath.Join(".astro", ".prerender")); err != nil {
		t.Fatalf("clean Astro prerender cache: %v", err)
	}
	cmd := exec.Command("npm", "run", "build")
	cmd.Dir = "."
	cmd.Env = append(os.Environ(),
		"ASTRO_OUT_DIR="+dest,
		"ASTRO_TELEMETRY_DISABLED=1",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("astro build failed: %v\noutput:\n%s", err, string(out))
	}
}

func TestDocsDeployWorkflowUsesCloudflarePages(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", ".github", "workflows", "deploy-gormes-docs.yml"))
	if err != nil {
		t.Fatalf("read deploy workflow: %v", err)
	}
	text := string(raw)

	wants := []string{
		"name: Deploy docs.gormes.ai",
		"paths:",
		"- 'docs/**'",
		"workflow_dispatch:",
		"actions/setup-node@v4",
		"npm ci",
		"npm run build",
		"Verify homepage content",
		`grep -F "git clone https://github.com/TrebuchetDynamics/gormes-agent.git" dist/index.html >/dev/null`,
		`! grep -F "curl -fsSL https://gormes.ai/install.sh | sh" dist/index.html >/dev/null`,
		`! grep -F "brew install trebuchet/gormes" dist/index.html >/dev/null`,
		"cloudflare/wrangler-action@v3",
		"command: pages project create gormes-docs --production-branch=main",
		"command: pages deploy docs/dist --project-name=gormes-docs --branch=main --commit-dirty=true",
		"domain=docs.gormes.ai",
	}
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("deploy workflow missing %q", want)
		}
	}
}

func TestCIWorkflowInstallsDocsNodeDependenciesBeforeGoTests(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read CI workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"actions/setup-node@v4",
		"node-version: '22'",
		"name: Install docs dependencies",
		"working-directory: docs",
		"npm ci",
		"name: Run Go tests",
		"go test ./... -count=1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("CI workflow missing %q", want)
		}
	}
	installIdx := strings.Index(text, "name: Install docs dependencies")
	testIdx := strings.Index(text, "name: Run Go tests")
	if installIdx < 0 || testIdx < 0 || installIdx > testIdx {
		t.Fatalf("CI workflow must install docs dependencies before Go tests")
	}
}
