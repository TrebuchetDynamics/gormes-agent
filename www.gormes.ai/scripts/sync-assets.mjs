import { copyFile, mkdir } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const siteRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const repoRoot = resolve(siteRoot, '..');
const legacyStatic = resolve(siteRoot, 'legacy/go-renderer/internal/site/static');

const copies = [
  ['install.sh', resolve(repoRoot, 'install.sh'), resolve(siteRoot, 'public/install.sh')],
  ['install.ps1', resolve(repoRoot, 'scripts/install.ps1'), resolve(siteRoot, 'public/install.ps1')],
  ['install.cmd', resolve(repoRoot, 'scripts/install.cmd'), resolve(siteRoot, 'public/install.cmd')],
  [
    'benchmarks.json',
    resolve(repoRoot, 'benchmarks.json'),
    resolve(siteRoot, 'src/data/benchmarks.json'),
  ],
  [
    'progress.json',
    resolve(repoRoot, 'docs/content/building-gormes/architecture_plan/progress.json'),
    resolve(siteRoot, 'src/data/progress.json'),
  ],
  [
    'favicon.ico',
    resolve(legacyStatic, 'favicon.ico'),
    resolve(siteRoot, 'public/static/favicon.ico'),
  ],
  [
    'favicon-16x16.png',
    resolve(legacyStatic, 'favicon-16x16.png'),
    resolve(siteRoot, 'public/static/favicon-16x16.png'),
  ],
  [
    'favicon-32x32.png',
    resolve(legacyStatic, 'favicon-32x32.png'),
    resolve(siteRoot, 'public/static/favicon-32x32.png'),
  ],
  [
    'apple-touch-icon.png',
    resolve(legacyStatic, 'apple-touch-icon.png'),
    resolve(siteRoot, 'public/static/apple-touch-icon.png'),
  ],
  [
    'android-chrome-192x192.png',
    resolve(legacyStatic, 'android-chrome-192x192.png'),
    resolve(siteRoot, 'public/static/android-chrome-192x192.png'),
  ],
  [
    'android-chrome-512x512.png',
    resolve(legacyStatic, 'android-chrome-512x512.png'),
    resolve(siteRoot, 'public/static/android-chrome-512x512.png'),
  ],
  [
    'social-card.png',
    resolve(legacyStatic, 'social-card.png'),
    resolve(siteRoot, 'public/static/social-card.png'),
  ],
  [
    'go-gopher-bear-lowpoly.png',
    resolve(legacyStatic, 'go-gopher-bear-lowpoly.png'),
    resolve(siteRoot, 'public/static/go-gopher-bear-lowpoly.png'),
  ],
];

for (const [, source, target] of copies) {
  await mkdir(dirname(target), { recursive: true });
  await copyFile(source, target);
}

console.log(`synced ${copies.length} www.gormes.ai assets`);
