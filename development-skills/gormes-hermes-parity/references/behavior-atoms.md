# Behavior Atoms

Use this reference when the scope is broad, the user gives one example, or a
drift may exist outside one file or command.

## Atom Shape

```text
surface:
trigger:
visible_contract:
state_and_side_effects:
upstream_ref:
gormes_ref:
status:
row:
validation:
risk:
```

Do not classify a behavior as covered from a matching name, route, package, or
stub. Coverage requires the observable contract, state changes, failure paths,
and model/channel-visible effects to preserve upstream behavior or document an
owned divergence.

## Inventory Searches

Start broad enough to find emitters and registration points:

```sh
rg -n "add_parser|subparsers|set_defaults|click|typer|argparse|cobra.Command|AddCommand|Use:|Aliases:" "$HERMES_SRC" cmd internal -g'*.py' -g'*.go'
rg -n "tool|schema|register|config|env|provider|model|gateway|platform|setup|install|profile|session|memory|skill|plugin|prompt|progress|status|typing|reply|send|route" "$HERMES_SRC" cmd internal docs webpages -g'*.py' -g'*.go' -g'*.md' -g'*.json' -g'*.sh'
rg -n "fmt\\.Fprint|fmt\\.Print|print\\(|Prompt|Question|Confirm|Select|Error\\(|status|progress|tool_progress|render|template|default|write|save" "$HERMES_SRC" cmd internal docs webpages -g'*.py' -g'*.go' -g'*.tsx' -g'*.md' -g'*.json' -g'*.sh'
```

For command manifest work, include:

```sh
go test ./cmd/gormes -run '^TestHermesCLIParityManifest' -count=1
```

## Source Buckets

| Bucket | Upstream evidence | Gormes evidence |
|---|---|---|
| Commands and aliases | parser definitions, help/version/default invocation tests, shell entrypoints | Cobra commands, manifests, focused CLI tests |
| Setup, config, install, migration | prompts, env keys, config writers, installer scripts, profile docs/tests | config structs/writers, setup/install commands, temp-home fixtures |
| Gateway and channels | adapters, renderer/progress callbacks, message sequencing tests | adapters, renderers, channel fixtures, transcript tests |
| Tools and schemas | tool registration, JSON schemas, permission gates, tool-loop tests | descriptors, kernel execution, schema tests, provider-visible fixtures |
| Providers and auth | auth commands, provider detection, model picker, fallback/error paths | provider registry, auth/config commands, routing and error tests |
| TUI and terminal UX | components, visible strings, status/progress emitters, snapshots | TUI components, render tests, transcript fixtures |
| Memory, sessions, profiles, skills, plugins | lifecycle commands, stores, defaults, sync/expansion helpers | stores, commands, registries, tests |
| Docs and examples | current docs/examples describing active behavior | Gormes docs/progress rows after source evidence is checked |

When the matrix reveals many gaps, split by behavior family and blast radius.
Prefer many buildable slices over one umbrella row.
