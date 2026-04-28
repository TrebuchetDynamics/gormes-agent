package docs_test

import (
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
			root: filepath.Join("..", "..", "hermes-agent"),
			represented: map[string]string{
				".plans":                         "`.plans/**`",
				"AGENTS.md":                      "`AGENTS.md`",
				"CONTRIBUTING.md":                "CONTRIBUTING",
				"Dockerfile":                     "`Dockerfile`",
				"README.md":                      "README",
				"SECURITY.md":                    "SECURITY",
				"acp_adapter":                    "`acp_adapter/**`",
				"acp_registry":                   "`acp_registry/**`",
				"agent":                          "`agent/*.py`",
				"batch_runner.py":                "`batch_runner.py`",
				"cli.py":                         "`cli.py`",
				"cron":                           "`cron/*.py`",
				"datagen-config-examples":        "`datagen-config-examples/**`",
				"docker":                         "`docker/**`",
				"docker-compose.yml":             "`docker-compose.yml`",
				"environments":                   "`environments/agent_loop.py`",
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
				"mini_swe_runner.py":             "`mini_swe_runner.py`",
				"model_tools.py":                 "`model_tools.py`",
				"nix":                            "`nix/**`",
				"optional-skills":                "`optional-skills/**`",
				"package-lock.json":              "`package-lock.json`",
				"package.json":                   "`package.json`",
				"packaging":                      "`packaging/**`",
				"plans":                          "`plans/**`",
				"plugins":                        "`plugins/**`",
				"pyproject.toml":                 "`pyproject.toml`",
				"rl_cli.py":                      "`rl_cli.py`",
				"run_agent.py":                   "`run_agent.py`",
				"scripts":                        "`scripts/**`",
				"setup-hermes.sh":                "`setup-hermes.sh`",
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
				".dockerignore", ".env.example", ".envrc", ".gitattributes",
				".github", ".gitignore", ".gitmodules", ".mailmap", "LICENSE",
				"MANIFEST.in",
			),
		},
		{
			name: "Honcho",
			root: filepath.Join("..", "..", "honcho"),
			represented: map[string]string{
				".claude":                    "`.claude/skills/**`",
				"CHANGELOG.md":               "CHANGELOG",
				"CLAUDE.md":                  "CLAUDE",
				"CONTRIBUTING.md":            "CONTRIBUTING",
				"Dockerfile":                 "`Dockerfile`",
				"README.md":                  "README",
				"alembic.ini":                "`alembic.ini`",
				"config.toml.example":        "`config.toml.example`",
				"docker":                     "`docker/**`",
				"docker-compose.yml.example": "`docker-compose.yml.example`",
				"docs":                       "`docs/v1/**`",
				"examples":                   "`examples/**`",
				"fly.toml":                   "`fly.toml`",
				"honcho-cli":                 "`honcho-cli/**`",
				"mcp":                        "`mcp/**`",
				"migrations":                 "`migrations/**`",
				"pyproject.toml":             "`pyproject.toml`",
				"scripts":                    "`scripts/**`",
				"sdks":                       "`sdks/python/**`",
				"src":                        "`src/models.py`",
				"tests":                      "`tests/**`",
				"uv.lock":                    "`uv.lock`",
			},
			ignored: ignoredCoverageClasses(
				".dockerignore", ".env.template", ".github", ".gitignore",
				".markdownlint.json", ".pre-commit-config.yaml", ".python-version",
				".vscode", "LICENSE",
			),
		},
	}

	for _, check := range checks {
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
		"hermes", "setup-hermes.sh", "config.toml.example", "alembic.ini":
		return true
	}
	switch filepath.Ext(name) {
	case ".py", ".md", ".mdx", ".ts", ".tsx", ".json", ".yaml", ".yml", ".toml", ".lock", ".nix", ".sh":
		return true
	default:
		return false
	}
}
