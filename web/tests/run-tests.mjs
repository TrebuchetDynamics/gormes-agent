import { readdir, readFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

const expectedRoutes = [
  ['dashboard', '/'],
  ['chat', '/chat'],
  ['config', '/config'],
  ['env', '/env'],
  ['sessions', '/sessions'],
  ['logs', '/logs'],
  ['cron', '/cron'],
  ['skills', '/skills'],
  ['docs', '/docs'],
  ['analytics', '/analytics'],
];

async function read(rel) {
  return readFile(path.join(root, rel), 'utf8');
}

async function DashboardFrontendScaffoldDefinesRoutes() {
  const app = await read('src/App.tsx');
  for (const [id, route] of expectedRoutes) {
    assert.match(app, new RegExp(`id:\\s*['\"]${id}['\"]`), `route id ${id} is present`);
    assert.match(app, new RegExp(`path:\\s*['\"]${route.replace('/', '\\/')}['\"]`), `route path ${route} is present`);
  }
  const pages = await readdir(path.join(root, 'src/pages'));
  for (const [id] of expectedRoutes) {
    const pageName = id === 'dashboard' ? 'DashboardPage.tsx' : `${id[0].toUpperCase()}${id.slice(1)}Page.tsx`;
    assert.ok(pages.includes(pageName), `page ${pageName} exists`);
  }
}

async function DashboardFrontendScaffoldBuildManifest() {
  const pkg = JSON.parse(await read('package.json'));
  assert.equal(pkg.type, 'module');
  assert.equal(pkg.scripts.build, 'vite build');
  assert.ok(pkg.dependencies.react, 'react dependency is declared');
  assert.ok(pkg.dependencies['react-dom'], 'react-dom dependency is declared');
  assert.ok(pkg.dependencies.vite, 'vite dependency is declared');
  assert.ok(existsSync(path.join(root, 'vite.config.ts')), 'vite.config.ts exists');
  assert.ok(existsSync(path.join(root, 'index.html')), 'index.html exists');
}

async function DashboardFrontendScaffoldPageFallbacks() {
  for (const [id] of expectedRoutes) {
    if (id === 'cron') continue;
    const pageName = id === 'dashboard' ? 'DashboardPage.tsx' : `${id[0].toUpperCase()}${id.slice(1)}Page.tsx`;
    const source = await read(`src/pages/${pageName}`);
    assert.match(source, /UnavailablePanel/, `${pageName} renders an explicit unavailable panel`);
    assert.match(source, /endpoint=/, `${pageName} names the backing endpoint`);
  }
}

async function CronDashboardPageRendersPartialJobsAndActions() {
  const source = await read('src/pages/CronPage.tsx');
  assert.doesNotMatch(source, /UnavailablePanel/, 'CronPage is now a concrete dashboard page');

  for (const helper of ['getJobTitle', 'getJobScheduleDisplay', 'getJobState']) {
    assert.match(source, new RegExp(`function\\s+${helper}\\b`), `${helper} helper is present`);
  }

  assert.match(source, /fetch\(['"]\/v1\/admin\/cron\/jobs['"]\)/, 'CronPage loads native cron admin jobs');
  assert.match(source, /\/pause[`'"]/, 'CronPage can pause jobs');
  assert.match(source, /\/resume[`'"]/, 'CronPage can resume jobs');
  assert.match(source, /\/trigger[`'"]/, 'CronPage can trigger jobs');
  assert.match(source, /getJobState\(job\)\s*===\s*['"]paused['"]/, 'pause/resume uses normalized state helper');

  for (const fallbackToken of ['schedule_display', 'schedule?.display', 'schedule?.expr', 'job.paused === true', 'job.enabled === false', 'scheduled']) {
    assert.match(source, new RegExp(fallbackToken.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')), `partial-record fallback ${fallbackToken} is present`);
  }

  for (const visibleState of ['Loading cron jobs', 'No cron jobs scheduled', 'Failed to load cron jobs', 'Retry']) {
    assert.match(source, new RegExp(visibleState), `visible state ${visibleState} is rendered`);
  }
}

const tests = {
  DashboardFrontendScaffoldDefinesRoutes,
  DashboardFrontendScaffoldBuildManifest,
  DashboardFrontendScaffoldPageFallbacks,
  CronDashboardPageRendersPartialJobsAndActions,
};

const runArgIndex = process.argv.indexOf('--run');
const filter = runArgIndex >= 0 ? process.argv[runArgIndex + 1] : '';
let failures = 0;
for (const [name, test] of Object.entries(tests)) {
  if (filter && !name.includes(filter)) continue;
  try {
    await test();
    console.log(`ok ${name}`);
  } catch (err) {
    failures += 1;
    console.error(`not ok ${name}`);
    console.error(err && err.stack ? err.stack : err);
  }
}
if (failures > 0) process.exit(1);
