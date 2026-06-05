import assert from 'node:assert/strict';
import { join } from 'node:path';
import { test } from 'node:test';

import { createLandingAssetCopyPlan, parseReleaseData, planBenchmarkRefresh } from './asset-sync.mjs';

const repoRoot = '/repo/gormes-agent';
const siteRoot = join(repoRoot, 'webpages/landing');

function byLabel(plan, label) {
  return plan.copies.find((copy) => copy.label === label);
}

test('createLandingAssetCopyPlan keeps the public installer and static mirror contract', () => {
  const plan = createLandingAssetCopyPlan({ repoRoot, siteRoot });
  const labels = plan.copies.map((copy) => copy.label);

  assert.equal(plan.copies.length, 5);
  assert.equal(new Set(labels).size, labels.length, 'copy labels must stay unique');
  assert.equal(new Set(plan.copies.map((copy) => copy.target)).size, plan.copies.length, 'copy targets must stay unique');

  assert.deepEqual(byLabel(plan, 'install.sh'), {
    label: 'install.sh',
    source: join(repoRoot, 'install.sh'),
    target: join(siteRoot, 'public/install.sh'),
  });
  assert.deepEqual(byLabel(plan, 'install.ps1'), {
    label: 'install.ps1',
    source: join(repoRoot, 'scripts/install.ps1'),
    target: join(siteRoot, 'public/install.ps1'),
  });
  assert.deepEqual(byLabel(plan, 'install.cmd'), {
    label: 'install.cmd',
    source: join(repoRoot, 'scripts/install.cmd'),
    target: join(siteRoot, 'public/install.cmd'),
  });
  assert.deepEqual(byLabel(plan, 'benchmarks.json'), {
    label: 'benchmarks.json',
    source: join(repoRoot, 'benchmarks.json'),
    target: join(siteRoot, 'src/data/benchmarks.json'),
  });
  assert.deepEqual(byLabel(plan, 'gormes-agent-logo-blue.svg'), {
    label: 'gormes-agent-logo-blue.svg',
    source: join(repoRoot, 'assets/gormes-agent-logo-blue.svg'),
    target: join(siteRoot, 'public/static/gormes-agent-logo-blue.svg'),
  });

  for (const copy of plan.copies) {
    assert.ok(!copy.source.includes('/legacy/go-renderer/'), `${copy.label} still reads from legacy renderer`);
    assert.ok(!copy.target.includes('/legacy/go-renderer/'), `${copy.label} still writes to legacy renderer`);
  }
});

test('planBenchmarkRefresh preserves benchmark refresh guardrails', () => {
  assert.deepEqual(planBenchmarkRefresh({ binaryExists: false }), {
    action: 'skip',
    message: 'benchmark refresh skipped: bin/gormes is not built',
  });
  assert.deepEqual(
    planBenchmarkRefresh({
      binaryExists: true,
      forceRefresh: false,
      gitStatus: { status: 0, stdout: ' M README.md\n' },
    }),
    {
      action: 'skip',
      message: 'benchmark refresh skipped: worktree has local changes',
    },
  );
  assert.deepEqual(
    planBenchmarkRefresh({
      binaryExists: true,
      forceRefresh: true,
      gitStatus: { status: 0, stdout: ' M README.md\n' },
    }),
    { action: 'record' },
  );
  assert.deepEqual(
    planBenchmarkRefresh({
      binaryExists: true,
      forceRefresh: false,
      gitStatus: { status: 128, stdout: ' M README.md\n' },
    }),
    { action: 'record' },
  );
});

test('parseReleaseData mirrors cmd/gormes version metadata for the landing page', () => {
  const release = parseReleaseData(
    'package main\nvar Version = "0.2.99"\nvar VersionDateAlias = "2026-05-24"\n',
    { source: 'cmd/gormes/main.go' },
  );

  assert.deepEqual(release, {
    version: '0.2.99',
    tag: 'v0.2.99',
    date_alias: '2026-05-24',
    url: 'https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/v0.2.99',
    source: 'cmd/gormes/main.go',
  });
});

test('parseReleaseData reports the missing version field by source file', () => {
  assert.throws(
    () => parseReleaseData('package main\nvar VersionDateAlias = "2026-05-24"\n', { source: 'cmd/gormes/main.go' }),
    /could not read Version from cmd\/gormes\/main.go/,
  );
});
