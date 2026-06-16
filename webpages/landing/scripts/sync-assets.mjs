import { spawnSync } from 'node:child_process';
import { access, copyFile, mkdir, readFile, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

import { createLandingAssetCopyPlan, parseReleaseData, planBenchmarkRefresh } from './asset-sync.mjs';

const siteRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const repoRoot = resolve(siteRoot, '../..');

async function pathExists(path) {
  try {
    await access(path);
    return true;
  } catch {
    return false;
  }
}

async function refreshBenchmarks() {
  const binaryExists = await pathExists(resolve(repoRoot, 'bin/gormes'));
  const forceRefresh = process.env.GORMES_WWW_REFRESH_BENCHMARKS === '1';
  const gitStatus = binaryExists && !forceRefresh
    ? spawnSync('git', ['status', '--porcelain'], {
        cwd: repoRoot,
        encoding: 'utf8',
      })
    : undefined;
  const plan = planBenchmarkRefresh({ binaryExists, forceRefresh, gitStatus });
  if (plan.action === 'skip') {
    console.log(plan.message);
    return;
  }

  const result = spawnSync('go', ['run', './cmd/gormes-repo', 'benchmark', 'record'], {
    cwd: repoRoot,
    env: process.env,
    stdio: 'inherit',
  });
  if (result.error?.code === 'ENOENT') {
    console.warn('benchmark refresh skipped: go is not available on PATH');
    return;
  }
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    throw new Error(`benchmark refresh failed with exit code ${result.status}`);
  }
}

async function writeReleaseData() {
  const versionFile = resolve(repoRoot, 'cmd/gormes/main.go');
  const raw = await readFile(versionFile, 'utf8');
  const release = parseReleaseData(raw, {
    source: 'cmd/gormes/main.go',
    errorSource: versionFile,
  });

  const target = resolve(siteRoot, 'src/data/release.json');
  await mkdir(dirname(target), { recursive: true });
  await writeFile(target, `${JSON.stringify(release, null, 2)}\n`, 'utf8');
}

const { copies } = createLandingAssetCopyPlan({ repoRoot, siteRoot });

await refreshBenchmarks();
await writeReleaseData();

for (const { source, target } of copies) {
  await mkdir(dirname(target), { recursive: true });
  await copyFile(source, target);
}

console.log(`synced ${copies.length} www.gormes.ai assets`);
