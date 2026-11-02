"""Repository capability gate for a Hermes parity cron job; never runs an agent.

Extracted from the repo-local Docker setup helper. See README.md for the runtime
layout and deployment boundary. The scheduler consumes the final JSON stdout
line. Blockers return wakeAgent=false with exit 0; --check instead exits 1.
"""
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys

ROOT = Path('/workspace')
SKILLS = ('gormes-skill-manager', 'gormes-hermes-parity', 'gormes-builder', 'gormes-tdd-slice')


def run(*args, timeout=15):
    return subprocess.run(args, cwd=ROOT, capture_output=True, text=True, timeout=timeout)


def check():
    blockers = []
    evidence = {}
    if not ROOT.is_dir():
        return ['workspace-missing'], evidence
    if (ROOT / '.hermes/setup.lock').exists():
        blockers.append('maintenance-in-progress')
    branch = run('git', 'branch', '--show-current')
    if branch.returncode or branch.stdout.strip() != 'development':
        blockers.append('development-branch-required')
    if not (ROOT / 'codemap.md').is_file():
        blockers.append('codemap-missing')
    for name in SKILLS:
        if not (ROOT / 'development-skills' / name / 'SKILL.md').is_file():
            blockers.append('skill-file-missing:' + name)
    roots = {str(Path(p).resolve()) for p in os.environ.get('HERMES_WRITE_SAFE_ROOT', '').split(os.pathsep) if p}
    if roots != {'/opt/data', '/workspace'}:
        blockers.append('write-scope-invalid')
    go = shutil.which('go')
    if not go:
        blockers.append('go-unavailable')
    else:
        version = run(go, 'version')
        have = re.search(r'\bgo(\d+)\.(\d+)\.(\d+)\b', version.stdout)
        want = re.search(r'^go\s+(\d+)\.(\d+)\.(\d+)\s*$', (ROOT / 'go.mod').read_text(), re.M)
        if version.returncode or not have or not want or tuple(map(int, have.groups())) < tuple(map(int, want.groups())):
            blockers.append('go-version-incompatible')
        else:
            evidence['go'] = '.'.join(have.groups())
    sys.path.insert(0, '/opt/hermes')
    from tools.lazy_deps import activate_durable_lazy_target
    activate_durable_lazy_target()
    from agent.file_safety import get_write_denied_error
    if get_write_denied_error(str(ROOT / '.hermes-development-write-probe')):
        blockers.append('workspace-write-denied')
    from tools.skills_tool import skill_view
    for name in SKILLS:
        result = json.loads(skill_view(name, preprocess=False))
        if result.get('error'):
            blockers.append('skill-loader-unavailable:' + name)
    if not blockers:
        # Local terminal sessions run a login shell, which can reset Docker's PATH.
        from tools.terminal_tool import terminal_tool, cleanup_all_environments
        try:
            result = json.loads(terminal_tool('go version', timeout=10, task_id='development-preflight'))
            if result.get('exit_code') != 0 or 'go' + evidence['go'] not in result.get('output', ''):
                blockers.append('terminal-tool-go-unavailable-or-mismatched')
            else:
                evidence['terminal_tool_go'] = 'passed'
        finally:
            cleanup_all_environments()
    if not blockers:
        result = run(go, 'run', './cmd/progress', 'validate', timeout=90)
        if result.returncode:
            blockers.append('progress-validation-failed')
        else:
            evidence['progress_validation'] = 'passed'
    return blockers, evidence


def main():
    try:
        blockers, evidence = check()
    except Exception as exc:
        blockers, evidence = ['preflight-error:' + type(exc).__name__], {}
    print(json.dumps({'wakeAgent': not blockers, 'blockers': blockers, 'evidence': evidence}))
    return 1 if '--check' in sys.argv and blockers else 0


if __name__ == '__main__':
    sys.exit(main())
