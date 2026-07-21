package docs_test

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestUpstreamCoverageLedgerMatchesSourceClasses(t *testing.T) {
	ledgerPath := filepath.Join("content", "building-gormes", "architecture_plan", "upstream-coverage-ledger.md")
	raw, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatalf("read coverage ledger: %v", err)
	}
	ledger := string(raw)

	checks := []struct {
		name        string
		root        string
		represented map[string]string
		ignored     map[string]struct{}
	}{
		{
			name: "Hermes",
			root: filepath.Join("..", "..", "..", "hermes-agent"),
			represented: map[string]string{
				".plans":                         "`.plans/**`",
				"AGENTS.md":                      "`AGENTS.md`",
				"CONTRIBUTING.es.md":             "CONTRIBUTING",
				"README.es.md":                   "README",
				"README.ur-pk.md":                "README",
				"README.zh-CN.md":                "README",
				"SECURITY.es.md":                 "SECURITY",
				"CLAUDE.md":                      "`CLAUDE.md`",
				"CONTRIBUTING.md":                "CONTRIBUTING",
				"Dockerfile":                     "`Dockerfile`",
				"README.md":                      "README",
				"SECURITY.md":                    "SECURITY",
				"acp_adapter":                    "`acp_adapter/auth.py:detect_provider`",
				"acp_registry":                   "`acp_registry/**`",
				"agent":                          "`agent/*.py`",
				"apps":                           "`apps/{desktop,bootstrap-installer,shared}/**`",
				"batch_runner.py":                "`batch_runner.py`",
				"cli-config.yaml.example":        "`cli-config.yaml.example`",
				"cli.py":                         "`cli.py`",
				"constraints-termux.txt":         "`constraints-termux.txt`",
				"cron":                           "`cron/*.py`",
				"datagen-config-examples":        "`datagen-config-examples/**`",
				"docker":                         "`docker/**`",
				"docker-compose.windows.yml":     "`docker-compose.windows.yml`",
				"docker-compose.yml":             "`docker-compose.yml`",
				"docs":                           "`docs/**`",
				"hermes_bootstrap.py":            "`hermes_bootstrap.py`",
				"flake.lock":                     "`flake.*`",
				"flake.nix":                      "`flake.*`",
				"gateway":                        "`gateway/**/*.py`",
				"hermes":                         "`hermes`",
				"hermes-already-has-routines.md": "`hermes-already-has-routines.md`",
				"hermes_cli":                     "`hermes_cli/**/*.py`",
				"hermes_constants.py":            "`hermes_constants.py`",
				"hermes_logging.py":              "`hermes_logging.py`",
				"hermes_state.py":                "`hermes_state.py`",
				"hermes_time.py":                 "`hermes_time.py`",
				"mcp_serve.py":                   "`mcp_serve.py`",
				"locales":                        "`locales/**`",
				"mini_swe_runner.py":             "`mini_swe_runner.py`",
				"model_tools.py":                 "`model_tools.py`",
				"nix":                            "`nix/**`",
				"optional-mcps":                  "`optional-mcps/**`",
				"optional-skills":                "`optional-skills/**`",
				"package-lock.json":              "`package-lock.json`",
				"package.json":                   "`package.json`",
				"packaging":                      "`packaging/**`",
				"plans":                          "`plans/**`",
				"plugins":                        "`plugins/**`",
				"providers":                      "`providers/**`",
				"pyproject.toml":                 "`pyproject.toml`",
				"rl_cli.py":                      "`rl_cli.py`",
				"run_agent.py":                   "`run_agent.py`",
				"scripts":                        "`scripts/**`",
				"setup-hermes.sh":                "`setup-hermes.sh`",
				"setup-llm.sh":                   "`setup-llm.sh`",
				"setup.py":                       "`setup.py`",
				"skills":                         "`skills/**`",
				"tests":                          "`tests/**`",
				"tools":                          "`tools/*.py`",
				"toolset_distributions.py":       "`toolset_distributions.py`",
				"toolsets.py":                    "`toolsets.py`",
				"trajectory_compressor.py":       "`trajectory_compressor.py`",
				"tui_gateway":                    "`tui_gateway/**`",
				"ui-tui":                         "`ui-tui/**`",
				"utils.py":                       "`utils.py`",
				"uv.lock":                        "`uv.lock`",
				"web":                            "`web/**`",
				"website":                        "`website/**`",
			},
			ignored: ignoredCoverageClasses(
				".dockerignore", ".env.example", ".envrc", ".gitattributes", ".hadolint.yaml",
				".github", ".gitignore", ".gitmodules", ".mailmap", "LICENSE",
				"MANIFEST.in", "assets", "tinker-atropos",
			),
		},
		{
			name: "Honcho",
			root: filepath.Join("..", "..", "..", "honcho"),
			represented: map[string]string{
				".claude":                    "`.claude/skills/**`",
				"AGENTS.md":                  "`AGENTS.md`",
				"CHANGELOG.md":               "CHANGELOG",
				"CLAUDE.md":                  "CLAUDE",
				"CONTRIBUTING.md":            "CONTRIBUTING",
				"Dockerfile":                 "`Dockerfile`",
				"README.md":                  "README",
				"alembic.ini":                "`alembic.ini`",
				"config.toml.example":        "`config.toml.example`",
				"database":                   "`database/init.sql`",
				"docker":                     "`docker/**`",
				"docker-compose.yml.example": "`docker-compose.yml.example`",
				".env.template":              "`.env.template`",
				"docs":                       "`docs/v3/**`",
				"examples":                   "`examples/**`",
				"fly.toml":                   "`fly.toml`",
				"honcho-cli":                 "`honcho-cli/src/honcho_cli/{main,config,output,validation,_help,branding}.py`",
				"mcp":                        "`mcp/**`",
				"migrations":                 "`migrations/**`",
				"pyproject.toml":             "`pyproject.toml`",
				"scripts":                    "`scripts/**`",
				"sdks":                       "`sdks/python/src/honcho/{client,aio,api_types,mixins}.py`",
				"src":                        "`src/models.py:MessageEmbedding,Collection,Document,QueueItem,ActiveQueueSession,WebhookEndpoint,SessionPeer`",
				"tests":                      "`tests/**`",
				"uv.lock":                    "`uv.lock`",
			},
			ignored: ignoredCoverageClasses(
				".dockerignore", ".github", ".gitignore",
				".markdownlint.json", ".pre-commit-config.yaml", ".python-version",
				".vscode", "LICENSE", "assets",
			),
		},
	}

	for _, check := range checks {
		check.root = preferredUpstreamRoot(check.name, check.root)
		t.Run(check.name, func(t *testing.T) {
			if _, err := os.Stat(check.root); err != nil {
				if os.IsNotExist(err) {
					t.Skipf("%s checkout not present at %s", check.name, check.root)
				}
				t.Fatalf("stat %s checkout: %v", check.name, err)
			}

			classes := collectUpstreamCoverageClasses(t, check.root)
			var unknown []string
			for class := range classes {
				if strings.HasPrefix(class, "RELEASE_") {
					continue
				}
				if _, ok := check.ignored[class]; ok {
					continue
				}
				evidence, ok := check.represented[class]
				if !ok {
					unknown = append(unknown, class)
					continue
				}
				if !strings.Contains(ledger, evidence) {
					t.Errorf("%s class %q missing ledger evidence %q", check.name, class, evidence)
				}
			}
			sort.Strings(unknown)
			if len(unknown) > 0 {
				t.Fatalf("%s coverage ledger has no classification for upstream source classes: %s", check.name, strings.Join(unknown, ", "))
			}
		})
	}
}

func TestNestedUpstreamFeatureCoverage(t *testing.T) {
	corpus := readNestedCoverageCorpus(t)

	type nestedFeatureCheck struct {
		name           string
		repo           string
		root           string
		path           string
		classification string
		evidence       []string
	}

	checks := []nestedFeatureCheck{
		{
			name:           "Hermes provider transports",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "agent/transports/base.py",
			classification: "still row-backed",
			evidence:       []string{"agent/transports/base.py", "Provider transport layer", "Phase 4.A"},
		},
		{
			name:           "Hermes chat completions transport",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "agent/transports/chat_completions.py",
			classification: "still row-backed",
			evidence:       []string{"agent/transports/chat_completions.py", "Provider transport layer", "Phase 4.A"},
		},
		{
			name:           "Hermes transport tests",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "tests/agent/transports/test_transport.py",
			classification: "still row-backed",
			evidence:       []string{"tests/agent/transports/test_transport.py", "Provider transport layer"},
		},
		{
			name:           "Hermes normal turn runtime",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "run_agent.py",
			classification: "still row-backed",
			evidence:       []string{"run_agent.py", "Normal agent loop", "Python-free normal agent turn e2e harness"},
		},
		{
			name:           "Hermes prompt builder",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "agent/prompt_builder.py",
			classification: "still row-backed",
			evidence:       []string{"agent/prompt_builder.py", "Prompt/context/compression", "Phase 4.C"},
		},
		{
			name:           "Hermes context compressor",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "agent/context_compressor.py",
			classification: "still row-backed",
			evidence:       []string{"agent/context_compressor.py", "Prompt/context/compression", "Phase 4.B"},
		},
		{
			name:           "Hermes context engine",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "agent/context_engine.py",
			classification: "still row-backed",
			evidence:       []string{"agent/context_engine.py", "Context engine and status", "Phase 4.B"},
		},
		{
			name:           "Hermes account usage read model",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "agent/account_usage.py",
			classification: "still row-backed",
			evidence:       []string{"agent/account_usage.py", "render_account_usage_lines", "Provider account usage read model + renderer"},
		},
		{
			name:           "Hermes account usage fixtures",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "tests/test_account_usage.py",
			classification: "still row-backed",
			evidence:       []string{"tests/test_account_usage.py", "Provider account usage reporting"},
		},
		{
			name:           "Hermes gateway usage command fixtures",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "tests/gateway/test_usage_command.py",
			classification: "still row-backed",
			evidence:       []string{"tests/gateway/test_usage_command.py", "Gateway /usage command binding over provider account usage"},
		},
		{
			name:           "Hermes ACP adapter",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "acp_adapter/server.py",
			classification: "mapped-by-contract",
			evidence:       []string{"acp_adapter/server.py", "MCP, managed tool gateway, ACP", "Phase 5.G, 5.H"},
		},
		{
			name:           "Hermes MCP server",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "mcp_serve.py",
			classification: "mapped-by-contract",
			evidence:       []string{"mcp_serve.py", "MCP, managed tool gateway, ACP"},
		},
		{
			name:           "Hermes Moonshot schema",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "agent/moonshot_schema.py",
			classification: "still row-backed",
			evidence:       []string{"agent/moonshot_schema.py", "Provider adapters and model families"},
		},
		{
			name:           "Hermes Yuanbao tools",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "tools/yuanbao_tools.py",
			classification: "still row-backed",
			evidence:       []string{"tools/yuanbao_tools.py", "Yuanbao tool drift", "Yuanbao gateway runtime + toolset registration"},
		},
		{
			name:           "Hermes plugin inventory",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "plugins/memory/honcho/client.py",
			classification: "mapped-by-contract",
			evidence:       []string{"plugins/memory/honcho/client.py", "Plugins and memory plugins"},
		},
		{
			name:           "Hermes memory plugin CLI discovery",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "plugins/memory/__init__.py",
			classification: "still row-backed",
			evidence:       []string{"plugins/memory/__init__.py", "discover_plugin_cli_commands", "Hermes CLI command-tree parity manifest"},
		},
		{
			name:           "Hermes plugin slash command registration",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "plugins/disk-cleanup/__init__.py",
			classification: "still row-backed",
			evidence:       []string{"plugins/disk-cleanup/__init__.py", "PluginContext.register_command", "Hermes CLI command-tree parity manifest"},
		},
		{
			name:           "Hermes skill catalog",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "skills/yuanbao/SKILL.md",
			classification: "mapped-by-contract",
			evidence:       []string{"skills/yuanbao/SKILL.md", "Skills and optional skills"},
		},
		{
			name:           "Hermes release automation",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "scripts/release.py",
			classification: "still row-backed",
			evidence:       []string{"scripts/release.py", "Packaging/release/install", "Phase 5.P"},
		},
		{
			name:           "Hermes CLI main parser",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "hermes_cli/main.py",
			classification: "still row-backed",
			evidence:       []string{"hermes_cli/main.py", "Hermes CLI command-tree parity manifest", "Phase 5.O"},
		},
		{
			name:           "Hermes CLI slash command registry",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "hermes_cli/commands.py",
			classification: "still row-backed",
			evidence:       []string{"hermes_cli/commands.py", "Hermes CLI command-tree parity manifest", "Phase 5.O"},
		},
		{
			name:           "Hermes gateway command handlers",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "gateway/run.py",
			classification: "still row-backed",
			evidence:       []string{"gateway/run.py", "Hermes CLI command-tree parity manifest", "Phase 5.O"},
		},
		{
			name:           "Hermes config command",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "hermes_cli/config.py",
			classification: "still row-backed",
			evidence:       []string{"hermes_cli/config.py", "Gormes config command surface", "Phase 5.O"},
		},
		{
			name:           "Hermes config home resolver",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "hermes_constants.py",
			classification: "still row-backed",
			evidence:       []string{"hermes_constants.py", "Hermes config migration dry-run manifest", "Phase 5.O"},
		},
		{
			name:           "Hermes state migration source",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "hermes_state.py",
			classification: "still row-backed",
			evidence:       []string{"hermes_state.py", "Hermes config migration dry-run manifest", "Phase 5.O"},
		},
		{
			name:           "Hermes OpenClaw CLI migration command",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "hermes_cli/claw.py",
			classification: "still row-backed",
			evidence:       []string{"hermes_cli/claw.py", "OpenClaw migration dry-run manifest", "Phase 5.O"},
		},
		{
			name:           "Hermes OpenClaw migration script",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "optional-skills/migration/openclaw-migration/scripts/openclaw_to_hermes.py",
			classification: "still row-backed",
			evidence:       []string{"openclaw_to_hermes.py", "OpenClaw migration dry-run manifest", "Phase 5.O"},
		},
		{
			name:           "Hermes Nix flake",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "flake.nix",
			classification: "still row-backed",
			evidence:       []string{"flake.nix", "Packaging/release/install", "Phase 5.P"},
		},
		{
			name:           "Honcho OpenAPI routes",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "docs/v3/openapi.json",
			classification: "still row-backed",
			evidence:       []string{"docs/v3/openapi.json", "OpenAPI v3 route manifest", "Phase 3.G, 5.Q"},
		},
		{
			name:           "Honcho models",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "src/models.py",
			classification: "still row-backed",
			evidence:       []string{"src/models.py", "Data model, migrations, CRUD invariants"},
		},
		{
			name:           "Honcho workspace router",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "src/routers/workspaces.py",
			classification: "still row-backed",
			evidence:       []string{"src/routers/workspaces.py", "Workspaces and API keys"},
		},
		{
			name:           "Honcho router surface",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "src/routers/messages.py",
			classification: "still row-backed",
			evidence:       []string{"src/routers/messages.py", "Peers, sessions, messages, files"},
		},
		{
			name:           "Honcho CRUD invariants",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "tests/crud/test_session.py",
			classification: "still row-backed",
			evidence:       []string{"tests/crud/**", "Data model, migrations, CRUD invariants"},
		},
		{
			name:           "Honcho session CRUD",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "src/crud/session.py",
			classification: "still row-backed",
			evidence:       []string{"src/crud/session.py", "Data model, migrations, CRUD invariants"},
		},
		{
			name:           "Honcho representation CRUD",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "src/crud/representation.py",
			classification: "still row-backed",
			evidence:       []string{"src/crud/representation.py", "Conclusions, observations, representations"},
		},
		{
			name:           "Honcho SDK compatibility",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "tests/sdk/test_client.py",
			classification: "still row-backed",
			evidence:       []string{"tests/sdk/**", "Goncho Honcho SDK compatibility e2e harness"},
		},
		{
			name:           "Honcho Python SDK client",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "sdks/python/src/honcho/client.py",
			classification: "mapped-by-contract",
			evidence:       []string{"sdks/python/src/honcho/client.py", "Python SDK"},
		},
		{
			name:           "Honcho TypeScript SDK client",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "sdks/typescript/src/client.ts",
			classification: "mapped-by-contract",
			evidence:       []string{"sdks/typescript/src/client.ts", "TypeScript SDK"},
		},
		{
			name:           "Honcho TypeScript SDK",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "sdks/typescript/__tests__/streaming.test.ts",
			classification: "mapped-by-contract",
			evidence:       []string{"sdks/typescript/__tests__/streaming.test.ts", "TypeScript SDK"},
		},
		{
			name:           "Honcho MCP sessions tool",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "mcp/src/tools/sessions.ts",
			classification: "mapped-by-contract",
			evidence:       []string{"mcp/src/tools/sessions.ts", "Honcho MCP"},
		},
		{
			name:           "Honcho CLI commands",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "honcho-cli/src/honcho_cli/main.py",
			classification: "mapped-by-contract",
			evidence:       []string{"honcho-cli/src/honcho_cli/main.py", "Honcho CLI"},
		},
		{
			name:           "Honcho CLI command fixtures",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "honcho-cli/tests/test_commands.py",
			classification: "mapped-by-contract",
			evidence:       []string{"honcho-cli/tests/test_commands.py", "Honcho CLI"},
		},
		{
			name:           "Honcho webhook route fixture",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "tests/routes/test_webhooks.py",
			classification: "still row-backed",
			evidence:       []string{"tests/routes/test_webhooks.py", "Webhooks"},
		},
		{
			name:           "Honcho webhook delivery worker",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "src/webhooks/webhook_delivery.py",
			classification: "still row-backed",
			evidence:       []string{"src/webhooks/webhook_delivery.py", "Goncho webhook delivery retry worker contract"},
		},
		{
			name:           "Honcho deriver queue",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "src/deriver/queue_manager.py",
			classification: "still row-backed",
			evidence:       []string{"src/deriver/queue_manager.py", "Queue/deriver/reconciler lifecycle"},
		},
		{
			name:           "Honcho dialectic chat",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "src/dialectic/chat.py",
			classification: "mapped-by-contract",
			evidence:       []string{"src/dialectic/chat.py", "Dialectic chat and tool loop"},
		},
		{
			name:           "Honcho dream scheduler",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "src/dreamer/dream_scheduler.py",
			classification: "still row-backed",
			evidence:       []string{"src/dreamer/dream_scheduler.py", "Dreaming"},
		},
		{
			name:           "Honcho dream orchestrator",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "src/dreamer/orchestrator.py",
			classification: "still row-backed",
			evidence:       []string{"src/dreamer/orchestrator.py", "Dreaming"},
		},
		{
			name:           "Honcho reasoning traces",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "src/telemetry/reasoning_traces.py",
			classification: "still row-backed",
			evidence:       []string{"src/telemetry/reasoning_traces.py", "Self-monitoring telemetry"},
		},
		{
			name:           "Honcho vector divergence",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "src/vector_store/lancedb.py",
			classification: "owned",
			evidence:       []string{"src/vector_store/lancedb.py", "Vector stores and embeddings", "owned with divergence contract"},
		},
		{
			name:           "Honcho live LLM exclusion",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "tests/live_llm/README.md",
			classification: "excluded",
			evidence:       []string{"tests/live_llm", "live-provider execution", "excluded"},
		},
		{
			name:           "Honcho Fly deploy divergence",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "fly.toml",
			classification: "owned",
			evidence:       []string{"fly.toml", "Hosted deploy/config", "owned/excluded"},
		},
		{
			name:           "Honcho database bootstrap",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "database/init.sql",
			classification: "owned",
			evidence:       []string{"database/init.sql", "Hosted deploy/config"},
		},
		{
			name:           "Honcho Dockerfile deploy surface",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "Dockerfile",
			classification: "owned",
			evidence:       []string{"Dockerfile", "Hosted deploy/config"},
		},
		{
			name:           "Honcho conclusions MCP tool",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "mcp/src/tools/conclusions.ts",
			classification: "mapped-by-contract",
			evidence:       []string{"mcp/src/tools/conclusions.ts", "Honcho MCP"},
		},
		{
			name:           "Honcho dreamer specialists",
			repo:           "Honcho",
			root:           filepath.Join("..", "..", "..", "honcho"),
			path:           "src/dreamer/specialists.py",
			classification: "still row-backed",
			evidence:       []string{"src/dreamer/specialists.py", "Dreaming"},
		},
		{
			name:           "Hermes browser provider routing",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "plugins/browser/firecrawl/provider.py",
			classification: "mapped-by-contract",
			evidence:       []string{"plugins/browser/firecrawl/provider.py", "Browser, web, media, voice, image"},
		},
		{
			name:           "Hermes config schema example",
			repo:           "Hermes",
			root:           filepath.Join("..", "..", "..", "hermes-agent"),
			path:           "cli-config.yaml.example",
			classification: "still row-backed",
			evidence:       []string{"cli-config.yaml.example", "CLI/config/status/doctor/backup"},
		},
	}

	addChecks := func(namePrefix, repo, root, classification string, evidence []string, paths ...string) {
		for _, path := range paths {
			checks = append(checks, nestedFeatureCheck{
				name:           namePrefix + " " + path,
				repo:           repo,
				root:           root,
				path:           path,
				classification: classification,
				evidence:       append([]string{path}, evidence...),
			})
		}
	}

	hermesRoot := filepath.Join("..", "..", "..", "hermes-agent")
	honchoRoot := filepath.Join("..", "..", "..", "honcho")

	addChecks("Hermes model metadata", "Hermes", hermesRoot, "still row-backed",
		[]string{"Model metadata, pricing, capabilities", "Phase 4.D"},
		"agent/model_metadata.py",
		"agent/models_dev.py",
		"model_tools.py",
	)
	addChecks("Hermes environment runtime", "Hermes", hermesRoot, "still row-backed",
		[]string{"Environment interface + file sync contract", "Phase 5.B"},
		"tools/environments/base.py",
		"tools/environments/file_sync.py",
		"tools/environments/local.py",
	)
	addChecks("Hermes ACP entry", "Hermes", hermesRoot, "mapped-by-contract",
		[]string{"ACP server side", "Phase 5.H"},
		"acp_adapter/auth.py",
		"acp_adapter/entry.py",
	)
	addChecks("Hermes plugin CLI", "Hermes", hermesRoot, "still row-backed",
		[]string{"Hermes CLI command-tree parity manifest", "Phase 5.O"},
		"hermes_cli/plugins.py",
		"hermes_cli/plugins_cmd.py",
	)
	addChecks("Hermes release exact", "Hermes", hermesRoot, "still row-backed",
		[]string{"OCI image", "Homebrew", "Phase 5.P"},
		"docker/entrypoint.sh",
		"packaging/homebrew/hermes-agent.rb",
	)

	addChecks("Honcho router", "Honcho", honchoRoot, "still row-backed",
		[]string{"OpenAPI v3 route manifest", "Phase 3.G"},
		"src/routers/sessions.py",
		"src/routers/peers.py",
		"src/routers/conclusions.py",
		"src/routers/keys.py",
		"src/routers/webhooks.py",
	)
	addChecks("Honcho CRUD", "Honcho", honchoRoot, "still row-backed",
		[]string{"Data model, migrations, CRUD invariants", "Goncho CRUD lifecycle invariants"},
		"src/crud/workspace.py",
		"src/crud/peer.py",
		"src/crud/message.py",
		"src/crud/document.py",
		"src/crud/webhook.py",
		"src/crud/peer_card.py",
		"src/crud/deriver.py",
	)
	addChecks("Honcho collection CRUD", "Honcho", honchoRoot, "still row-backed",
		[]string{"collection_cache_key", "Goncho CRUD lifecycle invariants"},
		"src/crud/collection.py",
	)
	addChecks("Honcho model symbols", "Honcho", honchoRoot, "still row-backed",
		[]string{"MessageEmbedding", "Collection", "Document", "QueueItem", "ActiveQueueSession", "WebhookEndpoint", "SessionPeer"},
		"src/models.py",
	)
	addChecks("Honcho dreamer", "Honcho", honchoRoot, "still row-backed",
		[]string{"Dreaming", "Phase 3.F"},
		"src/dreamer/surprisal.py",
		"src/dreamer/trees/base.py",
		"src/dreamer/trees/covertree.py",
		"src/dreamer/trees/graph.py",
		"src/dreamer/trees/lsh.py",
		"src/dreamer/trees/prototype.py",
		"src/dreamer/trees/rptree.py",
		"src/dreamer/trees/sklearn_wrapper.py",
	)
	addChecks("Honcho dialectic", "Honcho", honchoRoot, "mapped-by-contract",
		[]string{"Dialectic chat and tool loop", "Phase 3.F"},
		"src/dialectic/core.py",
		"src/dialectic/prompts.py",
	)
	addChecks("Honcho webhook adjunct", "Honcho", honchoRoot, "still row-backed",
		[]string{"Goncho webhook delivery retry worker contract", "Webhooks"},
		"src/webhooks/events.py",
		"tests/webhooks/test_webhook_delivery.py",
	)
	addChecks("Honcho telemetry", "Honcho", honchoRoot, "still row-backed",
		[]string{"Self-monitoring telemetry", "Phase 4.E"},
		"src/telemetry/metrics_collector.py",
		"src/telemetry/sentry.py",
		"src/telemetry/events/deletion.py",
	)
	addChecks("Honcho CLI command module", "Honcho", honchoRoot, "mapped-by-contract",
		[]string{"Goncho CLI command-tree parity", "Honcho CLI"},
		"honcho-cli/src/honcho_cli/commands/workspace.py",
		"honcho-cli/src/honcho_cli/commands/peer.py",
		"honcho-cli/src/honcho_cli/commands/session.py",
		"honcho-cli/src/honcho_cli/commands/message.py",
		"honcho-cli/src/honcho_cli/commands/conclusion.py",
		"honcho-cli/src/honcho_cli/commands/config_cmd.py",
		"honcho-cli/src/honcho_cli/commands/setup.py",
	)
	addChecks("Honcho CLI helper", "Honcho", honchoRoot, "mapped-by-contract",
		[]string{"Goncho CLI command-tree parity", "Honcho CLI"},
		"honcho-cli/src/honcho_cli/config.py",
		"honcho-cli/src/honcho_cli/output.py",
		"honcho-cli/src/honcho_cli/validation.py",
		"honcho-cli/src/honcho_cli/_help.py",
		"honcho-cli/src/honcho_cli/branding.py",
	)
	addChecks("Honcho SDK exact", "Honcho", honchoRoot, "mapped-by-contract",
		[]string{"Goncho Honcho SDK compatibility e2e harness", "Python SDK", "TypeScript SDK"},
		"sdks/python/src/honcho/aio.py",
		"sdks/python/src/honcho/api_types.py",
		"sdks/python/src/honcho/mixins.py",
		"sdks/typescript/src/validation.ts",
		"sdks/typescript/src/http/streaming.ts",
	)
	addChecks("Honcho deploy exact", "Honcho", honchoRoot, "owned",
		[]string{"Hosted deploy/config", "owned/excluded"},
		"config.toml.example",
		"docker-compose.yml.example",
		"docker/entrypoint.sh",
		"docker/prometheus.yml",
		"docker/grafana-datasource.yml",
	)

	for _, check := range checks {
		check.root = preferredUpstreamRoot(check.repo, check.root)
		t.Run(check.repo+"/"+check.name, func(t *testing.T) {
			if _, err := os.Stat(check.root); err != nil {
				if os.IsNotExist(err) {
					t.Skipf("%s checkout not present at %s", check.repo, check.root)
				}
				t.Fatalf("stat %s checkout: %v", check.repo, err)
			}

			upstreamPath := filepath.Join(check.root, filepath.FromSlash(check.path))
			if _, err := os.Stat(upstreamPath); err != nil {
				if os.IsNotExist(err) {
					t.Fatalf("nested_feature_path_missing repo=%s path=%s hint=update nested matrix or upstream refs", check.repo, filepath.ToSlash(upstreamPath))
				}
				t.Fatalf("stat nested upstream path %s: %v", upstreamPath, err)
			}

			if check.classification == "unknown/gap" || check.classification == "unknown-gap" {
				t.Fatalf("nested_feature_unmapped repo=%s path=%s classification=%s", check.repo, filepath.ToSlash(upstreamPath), check.classification)
			}
			if hasOnlyBroadNestedEvidence(check.evidence) {
				t.Fatalf("nested_feature_broad_only repo=%s path=%s evidence=%q", check.repo, filepath.ToSlash(upstreamPath), strings.Join(check.evidence, "|"))
			}
			for _, evidence := range check.evidence {
				if !strings.Contains(corpus, evidence) {
					t.Fatalf("nested_feature_unmapped repo=%s path=%s classification=%s missing_evidence=%q", check.repo, filepath.ToSlash(upstreamPath), check.classification, evidence)
				}
			}
		})
	}

	t.Run("rejects_broad_only_evidence", func(t *testing.T) {
		if !hasOnlyBroadNestedEvidence([]string{"tests/**"}) {
			t.Fatalf("broad tests/** evidence should not satisfy nested feature coverage")
		}
		if hasOnlyBroadNestedEvidence([]string{"tests/**", "tests/agent/transports/test_transport.py"}) {
			t.Fatalf("exact nested evidence should satisfy nested feature coverage despite a broad companion")
		}
	})

	t.Run("requires_builder_ready_progress_rows", func(t *testing.T) {
		requireBuilderReadyProgressRow(t, "Nested feature-level coverage test matrix for swarm gaps")
		requireBuilderReadyProgressRow(t, "Hermes and Honcho feature parity map to Go implementation plan")
		requireBuilderReadyProgressRow(t, "Hermes/Honcho Go runtime plan second-wave reconciliation")
		requireBuilderReadyProgressRow(t, "Provider account usage read model + renderer")
		requireBuilderReadyProgressRow(t, "Gateway /usage command binding over provider account usage")
		requireBuilderReadyProgressRow(t, "Goncho webhook delivery retry worker contract")
		requireBuilderReadyProgressRow(t, "Self-monitoring telemetry")
		requireBuilderReadyProgressRow(t, "Environment interface + file sync contract")
		requireBuilderReadyProgressRow(t, "Raw tool-call parser fixture matrix")
		requireBuilderReadyProgressRow(t, "ACP server side")
		requireBuilderReadyProgressRow(t, "Hermes CLI command-tree parity manifest")
		requireBuilderReadyProgressRow(t, "Goncho Honcho SDK compatibility e2e harness")
		requireBuilderReadyProgressRow(t, "Goncho CLI command-tree parity")
		requireBuilderReadyProgressRow(t, "Goncho CRUD lifecycle invariants")
		requireBuilderReadyProgressRow(t, "OCI image")
		requireBuilderReadyProgressRow(t, "Homebrew")
	})
}

func preferredUpstreamRoot(repo, fallback string) string {
	name := "honcho"
	if strings.EqualFold(repo, "Hermes") {
		name = "hermes-agent"
	}
	candidate := filepath.Join("..", "..", name)
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate
	}
	return fallback
}

func ignoredCoverageClasses(names ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(names))
	for _, name := range names {
		out[name] = struct{}{}
	}
	return out
}

func collectUpstreamCoverageClasses(t *testing.T, root string) map[string]struct{} {
	t.Helper()

	classes := map[string]struct{}{}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read upstream root %s: %v", root, err)
	}

	for _, entry := range entries {
		name := entry.Name()
		full := filepath.Join(root, name)
		if entry.IsDir() {
			if directoryHasCoverageSource(t, full) {
				classes[name] = struct{}{}
			}
			continue
		}
		if isCoverageSourceFile(name) {
			classes[name] = struct{}{}
		}
	}
	return classes
}

func directoryHasCoverageSource(t *testing.T, root string) bool {
	t.Helper()

	found := false
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", "__pycache__", ".venv", "venv", "dist", "build":
				return filepath.SkipDir
			}
			return nil
		}
		if isCoverageSourceFile(entry.Name()) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return found
}

func isCoverageSourceFile(name string) bool {
	switch name {
	case "Dockerfile", "docker-compose.yml", "docker-compose.yml.example",
		"hermes", "setup-llm.sh", "config.toml.example", "alembic.ini",
		"cli-config.yaml.example", "constraints-termux.txt":
		return true
	}
	switch filepath.Ext(name) {
	case ".py", ".md", ".mdx", ".ts", ".tsx", ".json", ".yaml", ".yml", ".toml", ".lock", ".nix", ".sh", ".sql":
		return true
	default:
		return false
	}
}

func readNestedCoverageCorpus(t *testing.T) string {
	t.Helper()

	paths := []string{
		filepath.Join("content", "building-gormes", "architecture_plan", "hermes-honcho-feature-map.md"),
		filepath.Join("content", "building-gormes", "architecture_plan", "hermes-honcho-go-runtime-plan.md"),
		filepath.Join("content", "building-gormes", "architecture_plan", "upstream-coverage-ledger.md"),
		filepath.Join("content", "building-gormes", "architecture_plan", "swarm-feature-parity-audit.md"),
		filepath.Join("content", "building-gormes", "architecture_plan", "progress.json"),
	}
	var b strings.Builder
	for _, path := range paths {
		var raw []byte
		if path == canonicalProgressPath {
			raw = canonicalProgressBytes(t, path)
		} else {
			var err error
			raw, err = os.ReadFile(path)
			if err != nil {
				t.Fatalf("read nested coverage corpus %s: %v", path, err)
			}
		}
		b.Write(raw)
		b.WriteByte('\n')
	}
	return b.String()
}

func hasOnlyBroadNestedEvidence(evidence []string) bool {
	if len(evidence) == 0 {
		return true
	}
	for _, item := range evidence {
		switch item {
		case "tests/**", "agent/**", "tools/**", "gateway/**", "hermes_cli/**",
			"plugins/**", "skills/**", "optional-skills/**", "acp_adapter/**",
			"docs/**", "src/**", "sdks/**", "mcp/**", "honcho-cli/**",
			"migrations/**", "examples/**", "scripts/**", "docker/**",
			"packaging/**", "nix/**":
			continue
		default:
			return false
		}
	}
	return true
}

func requireBuilderReadyProgressRow(t *testing.T, rowName string) {
	t.Helper()

	// Canonical backlog is read via internal/progress.Load (canonicalProgressBytes)
	// so this row-readiness check stays green whether the canonical path is a
	// monolithic file or a module-keyed split directory (module-split C5/C5c).
	var data any
	if err := json.Unmarshal(canonicalProgressBytes(t, canonicalProgressPath), &data); err != nil {
		t.Fatalf("decode canonical progress.json: %v", err)
	}
	row, ok := findProgressRowByName(data, rowName)
	if !ok {
		t.Fatalf("nested_feature_row_unready row=%q missing=row", rowName)
	}
	var missing []string
	for _, field := range []string{"source_refs", "write_scope", "acceptance", "done_signal"} {
		if !nonEmptyProgressField(row[field]) {
			missing = append(missing, field)
		}
	}
	if !nonEmptyProgressField(row["test_commands"]) && !nonEmptyProgressField(row["no_test_required"]) {
		missing = append(missing, "test_commands|no_test_required")
	}
	if len(missing) > 0 {
		t.Fatalf("nested_feature_row_unready row=%q missing=%s", rowName, strings.Join(missing, "|"))
	}
}

func findProgressRowByName(v any, name string) (map[string]any, bool) {
	switch x := v.(type) {
	case map[string]any:
		if x["name"] == name {
			return x, true
		}
		for _, child := range x {
			if row, ok := findProgressRowByName(child, name); ok {
				return row, true
			}
		}
	case []any:
		for _, child := range x {
			if row, ok := findProgressRowByName(child, name); ok {
				return row, true
			}
		}
	}
	return nil, false
}

func nonEmptyProgressField(v any) bool {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x) != ""
	case []any:
		return len(x) > 0
	case []string:
		return len(x) > 0
	default:
		return v != nil
	}
}
