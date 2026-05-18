package docs_test

import (
	"strings"
	"testing"
)

func TestProfileDocsDisambiguateGormesRuntimeAndUpstreamHermes(t *testing.T) {
	cli := readDoc(t, "content/cli/profile.md")
	assertContainsAll(t, "content/cli/profile.md", cli, []string{
		"## Profile model",
		"A Gormes profile is a Gormes home root.",
		"Profile selection alone is state separation, not a blanket filesystem sandbox.",
		"`agents.defaults.workspaces` is empty, the default project workspace is the\noperator home",
		"model-facing tools do not get\nblanket project access to the whole profile root",
		"identity files such as `SOUL.md` and `IDENTITY.md`, plus\nthe active profile's `skills/` directory",
		"`.env`, `auth.json`, session databases, memory databases, logs, and sibling\nprofiles",
		"File tools,\nproject-mode `execute_code`, and coding-agent delegation share that resolver.",
		"A local terminal fails closed under a non-empty allow-list",
		"uses `GORMES_HOME/home` as subprocess `HOME`",
		"## Runtime-ready subcommands",
		"## Compatibility aliases",
		"## Row-backed placeholders",
		"`gormes profile set`",
		"`action: \"profile_command_unavailable\"`",
		"`status: \"row_backed\"`",
	})

	recipe := readDoc(t, "content/recipes/profiles.md")
	assertContainsAll(t, "content/recipes/profiles.md", recipe, []string{
		"separate Gormes profile homes",
		"Creating a profile does not make it active.",
		"A non-empty workspace list is enforced as the\n   model-facing project allow-list.",
		"gormes setup profiles",
		"`agents.defaults.workspaces` plus `agents.defaults.channels`",
		"empty\n   `agents.defaults.workspaces` list as the operator home",
		"shell subprocesses use the active profile's `home/`\n   directory as `HOME`",
		"model-facing profile edits are\n   limited to explicit profile-owned content",
		"identity files such as `SOUL.md`\n   and `IDENTITY.md`, plus the active profile's `skills/` directory",
		"Profile files are still broad state",
		"Terminal is blocked after adding workspaces",
		"Confirm the active profile has a `home/` directory",
		"migrate into a separate profile home",
		"root: .../client-acme",
	})
	if strings.Contains(recipe, "root: .../.gormes/profiles/client-acme") {
		t.Fatal("profile recipe must use the redacted root printed by gormes profile show")
	}

	upstreamGuide := readDoc(t, "content/upstream-hermes/user-guide/profiles.md")
	assertContainsAll(t, "content/upstream-hermes/user-guide/profiles.md", upstreamGuide, []string{
		"Upstream Hermes reference",
		"`HERMES_HOME/home/`",
		"Local terminal subprocesses use\nthat directory as `HOME`",
		"## Sharing profiles as distributions",
		"hermes profile install github.com/you/research-bot --alias",
		"hermes profile update research-bot",
	})

	upstreamReference := readDoc(t, "content/upstream-hermes/reference/profile-commands.md")
	assertContainsAll(t, "content/upstream-hermes/reference/profile-commands.md", upstreamReference, []string{
		"Upstream Hermes reference",
		"| `install` | Install a profile distribution",
		"| `update` | Re-pull a distribution-managed profile",
		"| `info` | Show distribution metadata",
		"### `hermes profile install`",
		"### `hermes profile update`",
		"### `hermes profile info`",
		"`bash`, `zsh`, or `fish`",
	})
}

func TestSetupAndConfigDocsDescribeProfileWorkspaceListPolicy(t *testing.T) {
	setup := readDoc(t, "content/cli/setup.md")
	assertContainsAll(t, "content/cli/setup.md", setup, []string{
		"| `profiles` | Manage profiles and persist per-profile workspace/channel lists |",
		"`gormes setup profiles` writes `agents.defaults.workspaces`",
		"An empty list means the operator home is the\ndefault project workspace",
		"model-facing profile edits are limited to explicit\nprofile-owned content such as `SOUL.md`, `IDENTITY.md`, and `skills/`",
		"project-mode\n`execute_code`, and coding-agent delegation share this resolver",
	})

	config := readDoc(t, "content/configure/config-file.md")
	assertContainsAll(t, "content/configure/config-file.md", config, []string{
		"| `agents.defaults.workspaces` | Per-profile project workspace list persisted by `gormes setup profiles`.",
		"Empty list means operator home; non-empty list is the model-facing project read/write allow-list",
		"model-facing profile edits are limited to explicit profile-owned content such as `SOUL.md`, `IDENTITY.md`, and `skills/`",
		"The local terminal fails closed under a non-empty list",
		"| `agents.defaults.channels` | Per-profile messaging-channel list persisted by `gormes setup profiles`",
		"Per-agent primary workspace path. This is different from Goncho's memory workspace id.",
	})

	home := readDoc(t, "content/_index.md")
	assertContainsAll(t, "content/_index.md", home, []string{
		"Configure providers, models, agents, workspaces, profiles, and bindings from the CLI",
		"Separate state by named profile",
	})
	if strings.Contains(home, "Isolate work by named profile") {
		t.Fatal("docs home must not imply named profiles are an isolation boundary")
	}

	contract := readDoc(t, "content/building-gormes/contract-readiness.md")
	assertContainsAll(t, "content/building-gormes/contract-readiness.md", contract, []string{
		"Profile workspace allow-list enforcement policy",
		"model-facing tools must not treat the whole profile root as a project workspace",
		"explicit profile-owned content: identity files (`SOUL.md`, `IDENTITY.md` when present) and the active profile `skills/` directory",
		"`profile_workspace_scope_violation`",
	})

	nextSlices := readDoc(t, "content/building-gormes/builder-loop/next-slices.md")
	profilesProgress := readDoc(t, "content/building-gormes/architecture_plan/progress.json/modules/profiles.json")
	assertContainsAll(t, "content/building-gormes/architecture_plan/progress.json/modules/profiles.json", profilesProgress, []string{
		"Long-term plan: profile fleet supervisor and single control-plane gateway",
		"preserving Hermes-compatible profile state separation",
		"profile-scoped workers",
	})
	if strings.Contains(nextSlices, "Profile workspace allow-list enforcement policy") {
		t.Fatal("completed profile workspace allow-list row must not remain in next slices")
	}
	if strings.Contains(nextSlices, "preserving Hermes-compatible profile isolation") ||
		strings.Contains(nextSlices, "isolated profile workers") {
		t.Fatal("profile fleet roadmap must distinguish state separation from workspace sandboxing")
	}
}

func TestWebsiteRoadmapDoesNotMarkPartialSetupProfilesUmbrellaComplete(t *testing.T) {
	navivox := readDoc(t, "content/building-gormes/modules/navivox.md")
	assertContainsAll(t, "content/building-gormes/modules/navivox.md", navivox, []string{
		"| `planned` | `P2` | `navivox` | gormes setup profiles: per-profile workspaces + channels + navivox-default (Gormes-owned) |",
		"| `planned` | `P3` | `navivox` | gormes setup profiles — all profiles navivox-accessible by default |",
	})
	if strings.Contains(navivox, "| `complete` | `P2` | `navivox` | gormes setup profiles: per-profile workspaces + channels + navivox-default (Gormes-owned) |") {
		t.Fatal("navivox module roadmap must not mark setup-profiles umbrella complete while navivox child remains planned")
	}

	roadmap := readDoc(t, "content/building-gormes/architecture_plan/_index.md")
	assertContainsAll(t, "content/building-gormes/architecture_plan/_index.md", roadmap, []string{
		"- [ ] `navivox` gormes setup profiles: per-profile workspaces + channels + navivox-default (Gormes-owned)",
		"- [x] `profiles` gormes setup profiles — section scaffold + per-profile workspace list",
		"- [x] `profiles` gormes setup profiles — per-profile channels (telegram/whatsapp/discord/slack)",
		"- [ ] `navivox` gormes setup profiles — all profiles navivox-accessible by default",
	})
	if strings.Contains(roadmap, "- [x] `navivox` gormes setup profiles: per-profile workspaces + channels + navivox-default (Gormes-owned)") {
		t.Fatal("architecture roadmap must not check off the setup-profiles umbrella while navivox child remains planned")
	}
}
