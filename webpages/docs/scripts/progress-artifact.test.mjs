import assert from 'node:assert/strict';
import path from 'node:path';
import { test } from 'node:test';

import { createProgressArtifactPlan } from './progress-artifact.mjs';

const repoRoot = path.join('/repo', 'gormes-agent');
const docsDir = path.join(repoRoot, 'webpages', 'docs');

test('createProgressArtifactPlan emits the split-safe progress artifact contract', () => {
  const plan = createProgressArtifactPlan({ docsDir, repoRoot, env: {} });

  assert.equal(plan.command, 'go');
  assert.deepEqual(plan.args, ['run', './cmd/progress', 'emit']);
  assert.equal(plan.cwd, repoRoot);
  assert.equal(plan.maxBuffer, 16 * 1024 * 1024);
  assert.equal(
    plan.target,
    path.join(docsDir, 'dist', 'building-gormes', 'architecture_plan', 'progress.json'),
  );
});

test('createProgressArtifactPlan respects relative and absolute Astro output dirs', () => {
  assert.equal(
    createProgressArtifactPlan({
      docsDir,
      repoRoot,
      env: { ASTRO_OUT_DIR: 'build-output' },
    }).target,
    path.join(docsDir, 'build-output', 'building-gormes', 'architecture_plan', 'progress.json'),
  );

  assert.equal(
    createProgressArtifactPlan({
      docsDir,
      repoRoot,
      env: { ASTRO_OUT_DIR: '/tmp/gormes-docs-dist' },
    }).target,
    path.join('/tmp/gormes-docs-dist', 'building-gormes', 'architecture_plan', 'progress.json'),
  );
});
