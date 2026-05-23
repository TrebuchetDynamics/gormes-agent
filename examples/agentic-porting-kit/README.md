# Agentic Porting Kit

Validation-gated agentic porting for teams moving behavior from one runtime to another without losing tests, traceability, or source attribution.

This kit packages a small progress schema, six `porting-*` skills, and one tiny Python-to-Go example so an operator can guide an agent from source implementation evidence to target implementation tests without depending on any one product repo.

## What is included

- `schemas/progress.schema.json` — minimal progress-row schema for source-to-target porting work.
- `scripts/validate-example.sh` — local validation command for the included example.
- `skills/porting-skill-manager/SKILL.md` — routes planner, auditor, builder, TDD, and reference work.
- `skills/porting-planner/SKILL.md` — turns source behavior into target progress rows.
- `skills/porting-builder/SKILL.md` — ships one builder-ready row.
- `skills/porting-tdd-slice/SKILL.md` — keeps target changes test-first.
- `skills/porting-parity-auditor/SKILL.md` — compares source and target behavior.
- `skills/porting-references/SKILL.md` — inspects donor patterns without replacing the source oracle.
- `examples/python-greeter-to-go/` — a complete, intentionally small porting fixture.

## Quick start

1. Clone the kit and validate the example.

   ```sh
   git clone https://github.com/TrebuchetDynamics/agentic-porting-kit.git
   cd agentic-porting-kit
   ./scripts/validate-example.sh
   ```

2. Point the skills at the example progress file.

   ```sh
   export PORTING_PROGRESS_PATH=examples/python-greeter-to-go/progress.json
   ```

3. In your agent session, load `porting-skill-manager`, then use `porting-planner` to refine the example row or `porting-builder` plus `porting-tdd-slice` to implement one target behavior.

## Loading the skills

### Codex and other repo-local skill loaders

If your agent harness supports repo-local skills, expose this repository's `skills/` directory through that loader. For harnesses that use an `.agents/skills` view, create a symlink or copy from `skills/` into the target repository's `.agents/skills/` directory.

Start each porting session by loading `porting-skill-manager`; it routes to the smallest useful follow-up skill.

### Claude Code

Claude Code users can copy or symlink the kit's `skills/` directory into a Claude-readable skill directory for the target repository, such as `.claude/skills/` when repo-local skill loading is configured.

Keep the target repository's own branch, commit, release, and safety rules in its local instruction file. The kit skills provide the porting workflow; the host repository owns delivery policy.

## Adapting the progress schema

Use `PORTING_PROGRESS_PATH` to point the skills at the progress file for the active target repository. The default is `progress.json` in the current target repository.

Each executable row should include:

- the behavior to port;
- exact source implementation references;
- target implementation files allowed to change;
- validation commands;
- acceptance checks;
- final evidence expected from the builder.

You may extend `schemas/progress.schema.json` for a larger repository, but keep the same discipline: one logical backlog, no side queues, and no row marked complete without validation evidence.

## Gormes proves the method

Gormes proves the method at larger scale: it used this planner, builder, parity-auditor, TDD, and validation-gate loop to port a broad Python agent runtime shape into a Go-native target implementation. The kit extracts the method, not the runtime dependency. A new port should replace the source implementation, target implementation, progress file, and validation commands with its own evidence.

## License

MIT License. See `LICENSE`.
