# Hermes development preflight

Repository-owned source for the Docker Hermes parity-job gate, extracted from
local setup tooling. These files are safe to version; `.hermes/` remains ignored
for credentials, identity, receipts, locks, backups, and other live state.
Copying this source into Git does **not** update an installed script or cron job.

## Offline tests

From the repository root, with Python 3.11 or newer:

```sh
python3 -B -m unittest discover -s scripts/hermes -p 'test_*.py' -v
```

Tests use temporary workspace fixtures and replace external command/Hermes
boundaries. They require no Docker, Go installation, host Hermes, credentials,
network, maintenance lock, or live job. They verify gate decisions and JSON/exit
semantics, not the installed image's actual tool behavior.

## Runtime contract

`development_preflight.py` runs **inside the owned Hermes container**, using its
Python interpreter and effective runtime configuration:

- repository mounted at `/workspace`, on `development`;
- Hermes application at `/opt/hermes`;
- the Hermes home is a host bind of the repo's `.hermes/home` onto the
  image-required `/opt/data` path, so all persistent state lives under `.hermes`;
- `HOME` and `HERMES_HOME` set to `/opt/data` (the bind target);
- `HERMES_WRITE_SAFE_ROOT` containing exactly `/opt/data` and `/workspace`;
- Go on PATH, satisfying the repository's `go.mod` version;
- the four required repo skills discoverable through Hermes's skill loader.

These are container-layout conventions, not host-specific paths. The script has
no container name, project ID, job ID, credentials, or fixed Go install path.
It checks the workspace, maintenance lock, branch, codemap, canonical skill files,
write policy, Go version, skill loader, real terminal-tool Go visibility, and
`go run ./cmd/progress validate`. It activates Hermes's persistent lazy-package
path before using runtime imports; it does not install dependencies.

The final stdout line is JSON with `wakeAgent`, `blockers`, and `evidence`:

- **Scheduler mode:** exit 0 even when blocked, with `wakeAgent: false`. Configure
  the scheduler's JSON wake gate; exit status alone is not a readiness signal.
- **Manual mode (`--check`):** exit 1 when blocked, otherwise 0.
- Unexpected exceptions produce a blocker containing only the exception type,
  not potentially sensitive exception text.

Terminal/progress execution is skipped when earlier checks find blockers. A Go
validation can build caches or fetch modules; this is not a zero-write probe.
The gate never creates jobs, invokes a model, edits source, reindexes memory,
changes trust/write policy, or bypasses approvals. It is not a full test suite or
an exclusive builder lock. Policy checks cannot prove actual filesystem writes.

## Native repository change monitor

`repository_monitor.py` can be installed as a job's `monitor_script`, alongside
its existing prerequisite `script`. It emits only a stable SHA-256 digest of
HEAD, tracked/staged diffs, non-ignored untracked file contents (32 MiB total
limit), and selected runtime metadata: profile config/SOUL/operating skill, shell
initialization, Go executable, and the docs Astro executable. Runtime config
contents are not emitted or read; untracked symlink targets are not read.
`.hermes/` receipts/locks are excluded. Other ignored dependencies and changes
inside an already-dirty submodule are not comprehensively monitored.

Hermes owns the monitor hash/snapshot, not a new backlog. Unchanged ticks skip
inference and delivery; the first/changed input can run the existing job. Monitor
errors exit nonzero with only the exception type. Polling cadence does not change.
This is change detection, not writer exclusion, test success, or a hard budget.

**Retry limitation:** the inspected runtime saves the hash before the agent runs.
An unchanged failed/blocked run is therefore suppressed too. After investigating,
an operator can approve re-arming monitor state; a manual trigger or resume alone
may still be suppressed. Do not grant cron scheduling tools merely so a worker
can bypass this rule. Profile/config metadata changes can also wake a fresh check.

## Operator validation and deployment

Deployment is a separate, approved operation. Do not replace the live cron script
merely because this tracked copy changed. First verify container ownership, the
pinned image, and the existing job's identity/configuration; preserve credentials
and unrelated work. Under an approved maintenance window:

1. Run the offline tests. Review image compatibility for `tools.lazy_deps`,
   `agent.file_safety`, `tools.skills_tool`, `tools.terminal_tool`, and the
   scheduler's script/JSON wake-gate support. These upstream interfaces can change.
2. With the owned job paused, verify actual temporary file-tool write/read/edit/
   delete and outside-root denial, terminal-tool Go and both progress selectors,
   and skill loading. Clean up only the probe files you created.
3. Deploy the reviewed copies into the scheduler-allowed home scripts directory
   (`$HERMES_HOME/scripts`, backed here by the host `.hermes/home/scripts` bind)
   as `gormes-development-preflight.py` (`script`) and
   `gormes-repository-monitor.py` (`monitor_script`), workdir `/workspace`; do
   not use `--check` for the scheduled script. The scheduler refuses job script
   paths outside `$HERMES_HOME/scripts`, so the job keeps the
   `/opt/data/scripts/...` aliases while the bytes live in `.hermes/home/scripts`.
4. Verify blocked behavior during maintenance and successful JSON output after
   maintenance, through the actual scheduler script runner without invoking a
   model. Restore only the job's previously approved schedule/state.
5. Recheck terminal PATH, skills, and file tools after an approved container
   recreation. Keep local deployment receipts and backups under ignored state,
   not in this directory.

Keep future maintained changes here; treat the installed persistent copy as a
separately deployed artifact. Compose templates, environment scripts, memory
helpers, and live configuration are intentionally outside this first extraction.
