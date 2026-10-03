"""Stable, nonsecret input digest for Hermes's native monitor_script gate.

No job mutation, model call, or timestamp output. Scheduler owns comparison state.
See README.md for coverage, deployment, and retry limitations.
"""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess

MAX_UNTRACKED_BYTES = 32 * 1024 * 1024


def fingerprint(root=Path('/workspace'), home=Path('/workspace/.hermes/home')):
    digest = hashlib.sha256()

    def add(data):
        digest.update(len(data).to_bytes(8, 'big'))
        digest.update(data)

    def git(*args):
        result = subprocess.run(('git', *args), cwd=root, capture_output=True,
                                check=True, timeout=20,
                                env=dict(os.environ, GIT_OPTIONAL_LOCKS='0'))
        return result.stdout

    add(git('rev-parse', 'HEAD'))
    # Includes staged and unstaged edits, gitlink changes, and dirty submodules.
    add(git('diff', '--no-ext-diff', '--no-textconv', '--binary', 'HEAD'))
    names = git('ls-files', '--others', '--exclude-standard', '-z')
    total = 0
    for raw in sorted(filter(None, names.split(b'\0'))):
        relative = Path(os.fsdecode(raw))
        if relative.is_absolute() or '..' in relative.parts:
            raise ValueError('Unsafe repository path')
        # Operational receipts must never retrigger their own monitoring job.
        if relative.parts[0] == '.hermes':
            continue
        path = root / relative
        if not path.parent.resolve().is_relative_to(root.resolve()):
            raise ValueError('Untracked parent escapes repository')
        add(raw)
        if path.is_symlink():
            add(os.fsencode(os.readlink(path)))  # never read a symlink's target
        elif path.is_file():
            remaining = MAX_UNTRACKED_BYTES - total
            with path.open('rb') as stream:
                content = stream.read(remaining + 1)
            total += len(content)
            if total > MAX_UNTRACKED_BYTES:
                raise ValueError('Untracked payload exceeds monitor budget')
            add(content)
        else:
            # Missing/racing entries are errors, not a false unchanged result.
            raise ValueError('Untracked entry is not a regular file or symlink')

    # Metadata only: config contents/credentials never enter the output or prompt.
    runtime_paths = [home / 'config.yaml', home / 'SOUL.md',
                     root / '.hermes/skills/gormes-repo-operations/SKILL.md',
                     home / 'toolchains/gormes-development-env.sh',
                     root / 'webpages/docs/node_modules/.bin/astro']
    go = shutil.which('go')
    if go:
        runtime_paths.append(Path(go))
    for path in runtime_paths:
        add(os.fsencode(str(path)))
        try:
            stat = path.stat()
            add(f'{stat.st_size}:{stat.st_mtime_ns}:{stat.st_mode}'.encode())
        except FileNotFoundError:
            add(b'missing')
    for name in ('HERMES_WRITE_SAFE_ROOT', 'PATH'):
        add(os.fsencode(os.environ.get(name, '')))
    return digest.hexdigest()


def main():
    try:
        print(json.dumps({'repository_state_sha256': fingerprint()}, sort_keys=True))
        return 0
    except Exception as exc:
        print(json.dumps({'monitor_error': type(exc).__name__}))
        return 1


if __name__ == '__main__':
    raise SystemExit(main())
