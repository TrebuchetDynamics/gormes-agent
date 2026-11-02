"""Offline gate contract tests: no Docker, host Hermes, Go, cron, or model calls."""
import contextlib
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
from types import ModuleType
import unittest
from unittest.mock import Mock, patch

SCRIPT = Path(__file__).with_name('development_preflight.py')


class GateTests(unittest.TestCase):
    def setUp(self):
        self.assertTrue(SCRIPT.is_file(), 'Track the development preflight alongside its tests')
        spec = importlib.util.spec_from_file_location('development_preflight', SCRIPT)
        self.gate = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.gate)
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)
        self.gate.ROOT = self.root
        (self.root / 'codemap.md').write_text('# Fixture map\n')
        (self.root / 'go.mod').write_text('module example.invalid/fixture\n\ngo 1.26.5\n')
        for name in self.gate.SKILLS:
            directory = self.root / 'development-skills' / name
            directory.mkdir(parents=True)
            (directory / 'SKILL.md').write_text('# Fixture skill\n')

        # Hermes imports are container-only. Replace their external boundaries,
        # not the gate's decisions or its workspace/config-file checks.
        self.denied = Mock(return_value=None)
        self.skill_view = Mock(return_value=json.dumps({'name': 'fixture', 'content': '# Skill'}))
        self.terminal = Mock(return_value=json.dumps({'exit_code': 0, 'output': 'go version go1.26.8 linux/amd64'}))
        modules = {}
        for name, functions in {
            'tools.lazy_deps': {'activate_durable_lazy_target': lambda: None},
            'agent.file_safety': {'get_write_denied_error': self.denied},
            'tools.skills_tool': {'skill_view': self.skill_view},
            'tools.terminal_tool': {'terminal_tool': self.terminal, 'cleanup_all_environments': lambda: None},
        }.items():
            module = ModuleType(name)
            module.__dict__.update(functions)
            modules[name] = module
        for name in ('tools', 'agent'):
            modules[name] = ModuleType(name)
            modules[name].__path__ = []
        self.enterContext(patch.dict(sys.modules, modules))
        self.enterContext(patch.object(sys, 'path', list(sys.path)))
        self.enterContext(patch.dict(os.environ, {'HERMES_WRITE_SAFE_ROOT': '/opt/data:/workspace'}))
        self.which = self.enterContext(patch.object(self.gate.shutil, 'which', return_value='/fixture/bin/go'))
        self.branch = 'development'
        self.version = 'go version go1.26.8 linux/amd64'
        self.validation_status = 0
        self.enterContext(patch.object(self.gate.subprocess, 'run', side_effect=self.run_command))

    def run_command(self, args, **kwargs):
        self.assertEqual(kwargs['cwd'], self.root)
        self.assertGreater(kwargs['timeout'], 0)
        if args == ('git', 'branch', '--show-current'):
            return subprocess.CompletedProcess(args, 0, self.branch + '\n', '')
        if args == ('/fixture/bin/go', 'version'):
            return subprocess.CompletedProcess(args, 0, self.version, '')
        if args == ('/fixture/bin/go', 'run', './cmd/progress', 'validate'):
            return subprocess.CompletedProcess(args, self.validation_status, '', '')
        raise AssertionError('Unexpected external command: ' + repr(args))

    def result(self, *args):
        output = io.StringIO()
        with patch.object(sys, 'argv', [str(SCRIPT), *args]), contextlib.redirect_stdout(output):
            code = self.gate.main()
        return code, json.loads(output.getvalue().strip().splitlines()[-1])

    def assert_blocked(self, blocker):
        code, result = self.result()
        self.assertEqual(code, 0, 'A cron skip must not become a script failure')
        self.assertFalse(result['wakeAgent'])
        self.assertIn(blocker, result['blockers'])

    def lock(self):
        (self.root / '.hermes' / 'setup.lock').mkdir(parents=True)

    def test_ready_workspace_wakes_agent(self):
        code, result = self.result('--check')
        self.assertEqual(code, 0)
        self.assertEqual(result, {'wakeAgent': True, 'blockers': [], 'evidence': {
            'go': '1.26.8', 'terminal_tool_go': 'passed', 'progress_validation': 'passed'}})

    def test_maintenance_blocks_before_terminal_execution(self):
        self.lock()
        self.terminal.side_effect = AssertionError('Do not execute terminal during maintenance')
        self.assert_blocked('maintenance-in-progress')

    def test_manual_check_returns_failure_when_blocked(self):
        self.lock()
        code, result = self.result('--check')
        self.assertEqual(code, 1)
        self.assertFalse(result['wakeAgent'])

    def test_missing_workspace_blocks_agent(self):
        self.gate.ROOT = self.root / 'missing'
        self.assert_blocked('workspace-missing')

    def test_non_development_branch_blocks_agent(self):
        self.branch = 'main'
        self.assert_blocked('development-branch-required')

    def test_missing_codemap_blocks_agent(self):
        (self.root / 'codemap.md').unlink()
        self.assert_blocked('codemap-missing')

    def test_missing_canonical_skill_blocks_agent(self):
        (self.root / 'development-skills/gormes-builder/SKILL.md').unlink()
        self.assert_blocked('skill-file-missing:gormes-builder')

    def test_unsafe_or_incomplete_write_scopes_block_agent(self):
        for roots in ('', '/opt/data', '/', '/opt/data:/workspace:/'):
            with self.subTest(roots=roots), patch.dict(os.environ, {'HERMES_WRITE_SAFE_ROOT': roots}):
                self.assert_blocked('write-scope-invalid')

    def test_actual_hermes_write_denial_blocks_agent(self):
        self.denied.return_value = 'Write denied'
        self.assert_blocked('workspace-write-denied')

    def test_skill_loader_failure_blocks_agent(self):
        self.skill_view.return_value = json.dumps({'error': 'Skill unavailable'})
        self.assert_blocked('skill-loader-unavailable:gormes-builder')

    def test_missing_go_blocks_agent(self):
        self.which.return_value = None
        self.assert_blocked('go-unavailable')

    def test_incompatible_or_unparseable_go_blocks_agent(self):
        for version in ('go version go1.26.4 linux/amd64', 'unknown'):
            with self.subTest(version=version):
                self.version = version
                self.assert_blocked('go-version-incompatible')

    def test_login_shell_missing_or_different_go_blocks_agent(self):
        for code, output in ((127, 'go: not found'), (0, 'go version go1.26.9 linux/amd64')):
            with self.subTest(output=output):
                self.terminal.return_value = json.dumps({'exit_code': code, 'output': output})
                self.assert_blocked('terminal-tool-go-unavailable-or-mismatched')

    def test_invalid_progress_blocks_agent(self):
        self.validation_status = 1
        self.assert_blocked('progress-validation-failed')

    def test_unexpected_error_fails_closed_without_leaking_details(self):
        self.skill_view.side_effect = RuntimeError('PRIVATE_SENTINEL')
        code, result = self.result()
        self.assertEqual(code, 0)
        self.assertEqual(result, {'wakeAgent': False, 'blockers': ['preflight-error:RuntimeError'], 'evidence': {}})
        self.assertNotIn('PRIVATE_SENTINEL', json.dumps(result))


if __name__ == '__main__':
    unittest.main()
