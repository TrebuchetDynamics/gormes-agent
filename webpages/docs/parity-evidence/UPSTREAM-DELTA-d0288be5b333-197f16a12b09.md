# Upstream Delta: d0288be5b333..197f16a12b09

Baseline SHA: `d0288be5b3330d2442e3907185b8e9d0958297bb`

Target SHA: `197f16a12b098f1ea541bbd0a5ddd7d66f2d2dee`

Commits: **17230**

Changed paths: **13572**

## Category Summary

| Category | Paths | Share |
|---|---:|---:|
| `runtime` | 3669 | 27.0% |
| `app-ui` | 1999 | 14.7% |
| `test` | 6552 | 48.3% |
| `docs` | 1098 | 8.1% |
| `tooling-exclusion` | 254 | 1.9% |

## Runtime Feature Families

| Family | Paths | Added | Modified | Deleted |
|---|---:|---:|---:|---:|
| `plugin-catalog` | 744 | 627 | 103 | 14 |
| `hermes_cli` | 712 | 224 | 476 | 12 |
| `agent` | 381 | 106 | 275 | 0 |
| `plugins` | 368 | 163 | 154 | 51 |
| `tools` | 365 | 53 | 289 | 23 |
| `gateway` | 344 | 190 | 133 | 21 |
| `optional-skills` | 254 | 94 | 108 | 52 |
| `skills` | 177 | 28 | 97 | 52 |
| `tui_gateway` | 108 | 27 | 79 | 2 |
| `cron` | 51 | 21 | 29 | 1 |
| `pm` | 47 | 7 | 40 | 0 |
| `acp_adapter` | 18 | 8 | 10 | 0 |
| `hermes_platform` | 11 | 4 | 7 | 0 |
| `providers` | 3 | 0 | 3 | 0 |
| `batch_runner.py` | 1 | 0 | 1 | 0 |
| `cli-config.yaml.example` | 1 | 0 | 1 | 0 |
| `cli.py` | 1 | 0 | 1 | 0 |
| `hermes_bootstrap.py` | 1 | 0 | 1 | 0 |
| `hermes_constants.py` | 1 | 0 | 1 | 0 |
| `hermes_constants_scratch.py` | 1 | 0 | 1 | 0 |
| `hermes_constants_scratch_detach.py` | 1 | 1 | 0 | 0 |
| `hermes_constants_scratch_stuck.py` | 1 | 1 | 0 | 0 |
| `hermes_dir_acl.py` | 1 | 1 | 0 | 0 |
| `hermes_logging.py` | 1 | 0 | 1 | 0 |
| `hermes_startup_watchdog.py` | 1 | 0 | 1 | 0 |
| `hermes_state.py` | 1 | 0 | 1 | 0 |
| `hermes_state_api_content.py` | 1 | 1 | 0 | 0 |
| `hermes_state_common.py` | 1 | 0 | 1 | 0 |
| `hermes_state_compaction_display.py` | 1 | 1 | 0 | 0 |
| `hermes_state_compression.py` | 1 | 0 | 1 | 0 |
| `hermes_state_coverage.py` | 1 | 1 | 0 | 0 |
| `hermes_state_dbfile.py` | 1 | 0 | 1 | 0 |
| `hermes_state_detached_results.py` | 1 | 1 | 0 | 0 |
| `hermes_state_errors.py` | 1 | 0 | 1 | 0 |
| `hermes_state_fts.py` | 1 | 0 | 1 | 0 |
| `hermes_state_gateway.py` | 1 | 0 | 1 | 0 |
| `hermes_state_guard.py` | 1 | 0 | 1 | 0 |
| `hermes_state_health.py` | 1 | 0 | 1 | 0 |
| `hermes_state_holders.py` | 1 | 0 | 1 | 0 |
| `hermes_state_identity.py` | 1 | 1 | 0 | 0 |
| `hermes_state_ids.py` | 1 | 0 | 1 | 0 |
| `hermes_state_keys.py` | 1 | 1 | 0 | 0 |
| `hermes_state_local.py` | 1 | 1 | 0 | 0 |
| `hermes_state_local_lineage.py` | 1 | 1 | 0 | 0 |
| `hermes_state_local_migration.py` | 1 | 1 | 0 | 0 |
| `hermes_state_lockguard.py` | 1 | 0 | 1 | 0 |
| `hermes_state_lockowners.py` | 1 | 0 | 1 | 0 |
| `hermes_state_maintenance.py` | 1 | 0 | 1 | 0 |
| `hermes_state_media.py` | 1 | 1 | 0 | 0 |
| `hermes_state_messages.py` | 1 | 0 | 1 | 0 |
| `hermes_state_mutation_binding.py` | 1 | 1 | 0 | 0 |
| `hermes_state_mutation_branch.py` | 1 | 1 | 0 | 0 |
| `hermes_state_mutation_chat_pick.py` | 1 | 1 | 0 | 0 |
| `hermes_state_mutation_compress.py` | 1 | 1 | 0 | 0 |
| `hermes_state_mutation_guards.py` | 1 | 1 | 0 | 0 |
| `hermes_state_mutation_prepared.py` | 1 | 1 | 0 | 0 |
| `hermes_state_mutation_reset.py` | 1 | 1 | 0 | 0 |
| `hermes_state_mutation_retirement.py` | 1 | 1 | 0 | 0 |
| `hermes_state_mutation_transcript.py` | 1 | 1 | 0 | 0 |
| `hermes_state_mutations.py` | 1 | 1 | 0 | 0 |
| `hermes_state_pidns.py` | 1 | 1 | 0 | 0 |
| `hermes_state_portability.py` | 1 | 0 | 1 | 0 |
| `hermes_state_profile_repair.py` | 1 | 0 | 1 | 0 |
| `hermes_state_prompt_refresh.py` | 1 | 1 | 0 | 0 |
| `hermes_state_prune_rewrite.py` | 1 | 1 | 0 | 0 |
| `hermes_state_readpool.py` | 1 | 0 | 1 | 0 |
| `hermes_state_reconcile.py` | 1 | 1 | 0 | 0 |
| `hermes_state_registry.py` | 1 | 0 | 1 | 0 |
| `hermes_state_repair.py` | 1 | 0 | 1 | 0 |
| `hermes_state_rewind.py` | 1 | 0 | 1 | 0 |
| `hermes_state_runtime.py` | 1 | 1 | 0 | 0 |
| `hermes_state_runtime_workers.py` | 1 | 1 | 0 | 0 |
| `hermes_state_schema.py` | 1 | 0 | 1 | 0 |
| `hermes_state_search.py` | 1 | 0 | 1 | 0 |
| `hermes_state_session_list.py` | 1 | 1 | 0 | 0 |
| `hermes_state_sessions.py` | 1 | 0 | 1 | 0 |
| `hermes_state_sessions_model_config.py` | 1 | 1 | 0 | 0 |
| `hermes_state_telegram.py` | 1 | 0 | 1 | 0 |
| `hermes_state_terminal.py` | 1 | 1 | 0 | 0 |
| `hermes_state_timeline.py` | 1 | 0 | 1 | 0 |
| `hermes_state_titles.py` | 1 | 0 | 1 | 0 |
| `hermes_state_todo.py` | 1 | 1 | 0 | 0 |
| `hermes_state_usage.py` | 1 | 0 | 1 | 0 |
| `hermes_state_user_copy.py` | 1 | 0 | 1 | 0 |
| `hermes_state_wal.py` | 1 | 0 | 1 | 0 |
| `hermes_state_worker_compression.py` | 1 | 1 | 0 | 0 |
| `hermes_state_worker_context.py` | 1 | 1 | 0 | 0 |
| `hermes_state_worker_lifecycle.py` | 1 | 1 | 0 | 0 |
| `hermes_time.py` | 1 | 0 | 1 | 0 |
| `hermes_yaml.py` | 1 | 0 | 1 | 0 |
| `mcp_serve.py` | 1 | 0 | 0 | 1 |
| `mini_swe_runner.py` | 1 | 0 | 1 | 0 |
| `model_tools.py` | 1 | 0 | 1 | 0 |
| `optional-mcps` | 1 | 1 | 0 | 0 |
| `registration_lifecycle.py` | 1 | 0 | 1 | 0 |
| `run_agent.py` | 1 | 0 | 1 | 0 |
| `toolset_distributions.py` | 1 | 0 | 1 | 0 |
| `toolsets.py` | 1 | 0 | 1 | 0 |
| `trajectory_compressor.py` | 1 | 0 | 1 | 0 |
| `utils.py` | 1 | 0 | 1 | 0 |

## App / UI Families

| Family | Paths | Added | Modified | Deleted |
|---|---:|---:|---:|---:|
| `apps` | 1778 | 825 | 863 | 90 |
| `ui-tui` | 144 | 64 | 80 | 0 |
| `web` | 74 | 26 | 48 | 0 |
| `assets` | 3 | 1 | 2 | 0 |

## Test, Docs, And Tooling Families

| Category | Family | Paths |
|---|---|---:|
| `test` | `tests` | 5097 |
| `test` | `apps` | 1180 |
| `test` | `ui-tui` | 110 |
| `test` | `evals` | 101 |
| `test` | `web` | 26 |
| `test` | `tests-js` | 24 |
| `test` | `skills` | 7 |
| `test` | `optional-skills` | 3 |
| `test` | `scripts` | 3 |
| `test` | `hermes_cli` | 1 |
| `docs` | `contributors` | 656 |
| `docs` | `website` | 387 |
| `docs` | `locales` | 35 |
| `docs` | `.gitignore` | 1 |
| `docs` | `AGENTS.md` | 1 |
| `docs` | `COMPAT_MANIFEST.md` | 1 |
| `docs` | `CONTRIBUTING.es.md` | 1 |
| `docs` | `CONTRIBUTING.md` | 1 |
| `docs` | `Dockerfile` | 1 |
| `docs` | `README.es.md` | 1 |
| `docs` | `README.md` | 1 |
| `docs` | `README.ur-pk.md` | 1 |
| `docs` | `README.zh-CN.md` | 1 |
| `docs` | `compat_manifest.json` | 1 |
| `docs` | `flake.lock` | 1 |
| `docs` | `lefthook.yml` | 1 |
| `docs` | `package-lock.json` | 1 |
| `docs` | `package.json` | 1 |
| `docs` | `pyproject.toml` | 1 |
| `docs` | `ruff.pre-commit.toml` | 1 |
| `tooling-exclusion` | `scripts` | 157 |
| `tooling-exclusion` | `.github` | 68 |
| `tooling-exclusion` | `nix` | 16 |
| `tooling-exclusion` | `docker` | 5 |
| `tooling-exclusion` | `.coderabbit.yaml` | 1 |
| `tooling-exclusion` | `.dockerignore` | 1 |
| `tooling-exclusion` | `.env.example` | 1 |
| `tooling-exclusion` | `.envrc` | 1 |
| `tooling-exclusion` | `.npmrc` | 1 |
| `tooling-exclusion` | `activate` | 1 |
| `tooling-exclusion` | `activate.fish` | 1 |
| `tooling-exclusion` | `justfile` | 1 |

## Method

`git -C hermes-agent diff --name-status --no-renames d0288be5b3330d2442e3907185b8e9d0958297bb..197f16a12b098f1ea541bbd0a5ddd7d66f2d2dee --`

The full path-by-path classification is in the sibling JSON file. Every changed path has exactly one category; `other` is not used.
