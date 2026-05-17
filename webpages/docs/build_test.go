package docs_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var docsAstroBuild = struct {
	once sync.Once
	dir  string
	err  error
}{}

func TestMain(m *testing.M) {
	code := m.Run()
	if docsAstroBuild.dir != "" {
		_ = os.RemoveAll(docsAstroBuild.dir)
	}
	os.Exit(code)
}

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
		"install/index.html",
		"install/linux-macos/index.html",
		"install/windows/index.html",
		"install/from-source/index.html",
		"start-here/index.html",
		"troubleshooting/index.html",
		"troubleshooting/doctor/index.html",
		"troubleshooting/common-errors/index.html",
		"troubleshooting/logs/index.html",
		"guides/telegram-bot/index.html",
		"guides/provider-setup/index.html",
		"recipes/index.html",
		"recipes/doctor-offline/index.html",
		"recipes/first-turn/index.html",
		"recipes/telegram-bot/index.html",
		"recipes/diagnose/index.html",
		"recipes/migrate-hermes/index.html",
		"recipes/profiles/index.html",
		"recipes/multi-channel/index.html",
		"recipes/bindings/index.html",
		"recipes/fallback/index.html",
		"recipes/local-ollama/index.html",
		"reference/config/index.html",
		"reference/environment/index.html",
		"reference/providers/index.html",
		"reference/web-backends/index.html",
		"reference/paths-and-logs/index.html",
		"cli/index.html",
		"cli/acp/index.html",
		"cli/agent/index.html",
		"cli/auth/index.html",
		"cli/channels/index.html",
		"cli/chat/index.html",
		"cli/checkpoints/index.html",
		"cli/claw/index.html",
		"cli/completion/index.html",
		"cli/config/index.html",
		"cli/cron/index.html",
		"cli/curator/index.html",
		"cli/dashboard/index.html",
		"cli/doctor/index.html",
		"cli/fallback/index.html",
		"cli/gateway/index.html",
		"cli/goncho/index.html",
		"cli/kanban/index.html",
		"cli/logout/index.html",
		"cli/logs/index.html",
		"cli/mcp/index.html",
		"cli/memory/index.html",
		"cli/migrate/index.html",
		"cli/model/index.html",
		"cli/navivox/index.html",
		"cli/plugins/index.html",
		"cli/profile/index.html",
		"cli/restore/index.html",
		"cli/secrets/index.html",
		"cli/security/index.html",
		"cli/session/index.html",
		"cli/setup/index.html",
		"cli/skills/index.html",
		"cli/slack/index.html",
		"cli/status/index.html",
		"cli/system/index.html",
		"cli/telegram/index.html",
		"cli/uninstall/index.html",
		"cli/update/index.html",
		"cli/usage/index.html",
		"cli/version/index.html",
		"cli/whatsapp/index.html",
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
		"using-gormes/telegram-adapter/index.html",
		"using-gormes/configuration/index.html",
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
		"Start here",
		"Install",
		"Configure",
		"CLI reference",
		"Recipes",
		"Troubleshooting",
		"Why Gormes",
		`href="/start-here/"`,
		`href="/install/"`,
		`href="/troubleshooting/"`,
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
	if !strings.Contains(text, "CGO_ENABLED=0 go build -trimpath -o bin/gormes ./cmd/gormes") {
		t.Fatalf("built index.html missing current source build command")
	}
	if !strings.Contains(text, "curl -fsSLO https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh") {
		t.Fatalf("built index.html missing canonical install.sh command")
	}
	if !strings.Contains(text, "irm https://gormes.ai/install.ps1 -OutFile install.ps1") {
		t.Fatalf("built index.html missing canonical install.ps1 command")
	}
	if strings.Contains(text, "make build") {
		t.Fatalf("built index.html still contains stale make build command")
	}
	if strings.Contains(text, "curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh | sh") {
		t.Fatalf("built index.html still contains curl-pipe install command")
	}
	if strings.Contains(text, "https://gormes.ai/"+"install.sh") {
		t.Fatalf("built index.html still contains gormes.ai install.sh URL")
	}
	if strings.Contains(text, "https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh") {
		t.Fatalf("built index.html still contains raw GitHub install.sh URL")
	}
	if strings.Contains(text, "https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.ps1") {
		t.Fatalf("built index.html still contains raw GitHub install.ps1 URL")
	}
	if strings.Contains(text, "brew install trebuchet/gormes") {
		t.Fatalf("built index.html still contains stale Homebrew install command")
	}
}

func TestAstroBuild_InstallPagesUseCurrentCommands(t *testing.T) {
	tmp := t.TempDir()
	runDocsAstroBuild(t, tmp)

	pages := map[string]string{
		"start-here/index.html":                    "",
		"install/linux-macos/index.html":           "curl -fsSL https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh | sh",
		"install/windows/index.html":               "irm https://gormes.ai/install.ps1 | iex",
		"install/from-source/index.html":           "CGO_ENABLED=0 go build -trimpath -o bin/gormes ./cmd/gormes",
		"troubleshooting/common-errors/index.html": "$HOME/.local/bin",
	}
	for pagePath, want := range pages {
		raw, err := os.ReadFile(filepath.Join(tmp, pagePath))
		if err != nil {
			t.Fatalf("read built %s: %v", pagePath, err)
		}
		text := string(raw)
		if want != "" && !strings.Contains(text, want) {
			t.Fatalf("built %s missing %q", pagePath, want)
		}
		for _, reject := range []string{
			"https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh",
			"https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.ps1",
			"https://gormes.ai/" + "install.sh",
			"go install github.com/TrebuchetDynamics/gormes-agent/cmd/gormes@latest",
			"$HOME/go/bin",
		} {
			if strings.Contains(text, reject) {
				t.Fatalf("built %s contains stale token %q", pagePath, reject)
			}
		}
	}

	fromSource, err := os.ReadFile(filepath.Join(tmp, "install/from-source/index.html"))
	if err != nil {
		t.Fatalf("read built install/from-source/index.html: %v", err)
	}
	if strings.Contains(string(fromSource), "make build") {
		t.Fatalf("built install/from-source/index.html still contains stale make build command")
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
		"Gormes runs AI agents from one Go-native runtime.",
		"Choose source build,",
		"install.sh",
		"install.ps1",
		"What is Gormes?",
		"Go-native runtime",
		"Offline proof path",
		"Three install paths",
		"What you can do today",
		"Support labels",
		"Runtime-ready",
		"Trust posture",
		"Source build and release-first",
		"Progress data is generated from the canonical",
		"Users and operators",
		"Browse sessions, config, skills, logs, and audits",
		"What lives here?",
		"Start here",
		"Install",
		"Configure",
		"CLI reference",
		"Recipes",
		"Troubleshooting",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("built index.html missing operator-first docs token %q", want)
		}
	}
	for _, reject := range []string{
		"connect a Hermes backend",
		"Run Hermes Through a Go Operator Console",
		"curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh | sh",
		"https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh",
		"https://gormes.ai/" + "install.sh",
	} {
		if strings.Contains(text, reject) {
			t.Fatalf("built index.html contains stale token %q", reject)
		}
	}
}

func TestAstroBuild_IndexShowsBlueGormesAgentLogo(t *testing.T) {
	tmp := t.TempDir()
	runDocsAstroBuild(t, tmp)

	body, err := os.ReadFile(filepath.Join(tmp, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{
		`src="/gormes-agent-logo-blue.svg"`,
		`alt="GORMES-AGENT"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("built index.html missing blue GORMES-AGENT logo token %q", want)
		}
	}

	logoPath := filepath.Join(tmp, "gormes-agent-logo-blue.svg")
	if _, err := os.Stat(logoPath); err != nil {
		t.Fatalf("built docs output missing blue GORMES-AGENT logo asset: %v", err)
	}
	logo, err := os.ReadFile(logoPath)
	if err != nil {
		t.Fatalf("read built blue GORMES-AGENT logo asset: %v", err)
	}
	logoText := string(logo)
	for _, want := range []string{
		`fill="#73cedd"`,
		`shape-rendering="crispEdges"`,
		"Straight block-grid GORMES-AGENT logo",
	} {
		if !strings.Contains(logoText, want) {
			t.Fatalf("built logo asset missing %q", want)
		}
	}
	for _, reject := range []string{"<text", "<tspan", "font-family"} {
		if strings.Contains(logoText, reject) {
			t.Fatalf("built logo asset still depends on font-rendered text token %q", reject)
		}
	}
}

func runDocsAstroBuild(t *testing.T, dest string) {
	t.Helper()
	docsAstroBuild.once.Do(func() {
		docsAstroBuild.dir, docsAstroBuild.err = os.MkdirTemp(".", ".gormes-docs-astro-build-*")
		if docsAstroBuild.err != nil {
			return
		}
		if err := os.RemoveAll(".astro"); err != nil {
			docsAstroBuild.err = fmt.Errorf("clean Astro cache: %w", err)
			return
		}
		cmd := exec.Command("npm", "run", "build")
		cmd.Dir = "."
		cmd.Env = append(os.Environ(),
			"ASTRO_OUT_DIR="+docsAstroBuild.dir,
			"ASTRO_TELEMETRY_DISABLED=1",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			docsAstroBuild.err = fmt.Errorf("astro build failed: %w\noutput:\n%s", err, string(out))
		}
	})
	if docsAstroBuild.err != nil {
		t.Fatal(docsAstroBuild.err)
	}
	if err := os.CopyFS(dest, os.DirFS(docsAstroBuild.dir)); err != nil {
		t.Fatalf("copy cached Astro build: %v", err)
	}
}

func TestDocsDeployWorkflowUsesCloudflarePages(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "deploy-gormes-docs.yml"))
	if err != nil {
		t.Fatalf("read deploy workflow: %v", err)
	}
	text := string(raw)

	wants := []string{
		"name: Deploy docs.gormes.ai",
		"paths:",
		"- 'webpages/docs/**'",
		"workflow_dispatch:",
		"actions/setup-node@v4",
		"cache-dependency-path: webpages/docs/package-lock.json",
		"actions/setup-go@v6",
		"go-version-file: go.mod",
		"working-directory: webpages/docs",
		"npm ci",
		"npm run build",
		"Verify homepage content",
		`grep -F "git clone https://github.com/TrebuchetDynamics/gormes-agent.git" dist/index.html >/dev/null`,
		`grep -F "CGO_ENABLED=0 go build -trimpath -o bin/gormes ./cmd/gormes" dist/index.html >/dev/null`,
		`grep -F "curl -fsSLO https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh" dist/index.html >/dev/null`,
		`! grep -E 'https://gormes[.]ai/install[.]sh' dist/index.html >/dev/null`,
		`! grep -F "https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh" dist/index.html >/dev/null`,
		`! grep -F "brew install trebuchet/gormes" dist/index.html >/dev/null`,
		"cloudflare/wrangler-action@v3",
		"command: pages project create gormes-docs --production-branch=main",
		"command: pages deploy webpages/docs/dist --project-name=gormes-docs --branch=main --commit-dirty=true",
		"domain=docs.gormes.ai",
	}
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("deploy workflow missing %q", want)
		}
	}
}

func TestDocsProgressArtifactUsesSplitSafeEmitter(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("scripts", "copy-progress-json.mjs"))
	if err != nil {
		t.Fatalf("read copy-progress-json script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"execFile",
		"'go'",
		"'./cmd/progress'",
		"'emit'",
		"maxBuffer",
		"fs.writeFile(target, stdout)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("copy-progress-json script missing %q", want)
		}
	}
	for _, reject := range []string{
		"fs.copyFile(source, target)",
		"'content', 'building-gormes', 'architecture_plan', 'progress.json'",
	} {
		if strings.Contains(text, reject) {
			t.Fatalf("copy-progress-json script must not raw-copy the canonical path; found %q", reject)
		}
	}
}

func TestCIWorkflowInstallsDocsNodeDependenciesBeforeGoTests(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read CI workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"actions/setup-node@v4",
		"node-version: '22'",
		"webpages/docs/package-lock.json",
		"webpages/blog/package-lock.json",
		"name: Install docs dependencies",
		"working-directory: webpages/docs",
		"name: Install blog dependencies",
		"working-directory: webpages/blog",
		"npm ci",
		"name: Run Go tests",
		"go test ./... -count=1",
		"name: Test engineering blog",
		"npm run test",
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
