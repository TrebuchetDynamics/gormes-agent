import { readdir, readFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';

const root = path.resolve(import.meta.dirname, '..');

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
    const pageName = id === 'dashboard' ? 'DashboardPage.tsx' : `${id[0].toUpperCase()}${id.slice(1)}Page.tsx`;
    const source = await read(`src/pages/${pageName}`);
    assert.match(source, /UnavailablePanel/, `${pageName} renders an explicit unavailable panel`);
    assert.match(source, /endpoint=/, `${pageName} names the backing endpoint`);
  }
}

const tests = {
  DashboardFrontendScaffoldDefinesRoutes,
  DashboardFrontendScaffoldBuildManifest,
  DashboardFrontendScaffoldPageFallbacks,
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
