"""Offline change-monitor tests; no real Git commands or Hermes runtime."""
import contextlib
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

SCRIPT = Path(__file__).with_name('repository_monitor.py')


class MonitorTests(unittest.TestCase):
    def setUp(self):
        self.assertTrue(SCRIPT.is_file(), 'Repository change monitor is missing')
        spec = importlib.util.spec_from_file_location('repository_monitor', SCRIPT)
        self.monitor = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.monitor)
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name) / 'repo'
        self.home = Path(temp.name) / 'home'
        self.root.mkdir()
        self.home.mkdir()
        self.diff = b''
        self.untracked = b''
        self.head = b'abc123\n'
        self.enterContext(patch.object(self.monitor.subprocess, 'run', side_effect=self.git))
        self.enterContext(patch.object(self.monitor.shutil, 'which', return_value=None))
        self.enterContext(patch.dict(os.environ, {'HERMES_WRITE_SAFE_ROOT': '/opt/data:/workspace', 'PATH': '/fixture/bin'}))

    def git(self, args, **kwargs):
        self.assertEqual(kwargs['cwd'], self.root)
        self.assertGreater(kwargs['timeout'], 0)
        self.assertEqual(kwargs['env']['GIT_OPTIONAL_LOCKS'], '0')
        commands = {
            ('git', 'rev-parse', 'HEAD'): self.head,
            ('git', 'diff', '--no-ext-diff', '--no-textconv', '--binary', 'HEAD'): self.diff,
            ('git', 'ls-files', '--others', '--exclude-standard', '-z'): self.untracked,
        }
        return subprocess.CompletedProcess(args, 0, commands[tuple(args)], b'')

    def fingerprint(self):
        return self.monitor.fingerprint(self.root, self.home)

    def test_unchanged_inputs_are_stable(self):
        self.assertEqual(self.fingerprint(), self.fingerprint())

    def test_tracked_changes_and_commit_changes_are_detected(self):
        baseline = self.fingerprint()
        self.diff = b'diff: tracked source changed'
        self.assertNotEqual(baseline, self.fingerprint())
        self.diff = b''
        self.head = b'new-commit\n'
        self.assertNotEqual(baseline, self.fingerprint())

    def test_untracked_content_changes_are_detected(self):
        self.untracked = b'new.go\0'
        p = self.root / 'new.go'
        p.write_text('before')
        baseline = self.fingerprint()
        p.write_text('after')
        self.assertNotEqual(baseline, self.fingerprint())

    def test_untracked_symlink_does_not_read_external_content(self):
        target = self.home / 'private'
        target.write_text('private before')
        (self.root / 'link').symlink_to(target)
        self.untracked = b'link\0'
        baseline = self.fingerprint()
        target.write_text('private after')
        self.assertEqual(baseline, self.fingerprint())

    def test_runtime_configuration_and_ignored_dependency_changes_are_detected(self):
        baseline = self.fingerprint()
        (self.home / 'config.yaml').write_text('private: DO_NOT_PRINT')
        configured = self.fingerprint()
        self.assertNotEqual(baseline, configured)
        astro = self.root / 'webpages/docs/node_modules/.bin/astro'
        astro.parent.mkdir(parents=True)
        astro.write_text('fixture executable')
        self.assertNotEqual(configured, self.fingerprint())
        self.assertNotIn('DO_NOT_PRINT', self.fingerprint())

    def test_setup_lock_is_not_a_change_input(self):
        baseline = self.fingerprint()
        (self.root / '.hermes/setup.lock').mkdir(parents=True)
        self.assertEqual(baseline, self.fingerprint())

    def test_unsafe_untracked_path_is_rejected(self):
        self.untracked = b'../outside\0'
        with self.assertRaises(ValueError):
            self.fingerprint()

    def test_large_untracked_payload_fails_closed(self):
        self.untracked = b'large\0'
        (self.root / 'large').write_bytes(b'12345')
        with patch.object(self.monitor, 'MAX_UNTRACKED_BYTES', 4), self.assertRaises(ValueError):
            self.fingerprint()

    def test_monitor_failure_is_nonzero_and_redacts_exception(self):
        out = io.StringIO()
        with patch.object(self.monitor, 'fingerprint', side_effect=RuntimeError('PRIVATE_SENTINEL')), contextlib.redirect_stdout(out):
            code = self.monitor.main()
        self.assertEqual(code, 1)
        self.assertEqual(json.loads(out.getvalue()), {'monitor_error': 'RuntimeError'})
        self.assertNotIn('PRIVATE_SENTINEL', out.getvalue())


if __name__ == '__main__':
    unittest.main()
