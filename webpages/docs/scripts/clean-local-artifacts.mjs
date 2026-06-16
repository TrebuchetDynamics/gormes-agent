#!/usr/bin/env node
import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const docsDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const args = new Set(process.argv.slice(2));
const includeDeps = args.has('--deps');
const dryRun = args.has('--dry-run');
const unknownArgs = [...args].filter((arg) => !['--deps', '--dry-run'].includes(arg));

if (unknownArgs.length > 0) {
  console.error(`Unknown argument(s): ${unknownArgs.join(', ')}`);
  console.error('Usage: node ./scripts/clean-local-artifacts.mjs [--deps] [--dry-run]');
  process.exit(2);
}

const generatedTargets = [
  '.astro',
  '.hugo_build.lock',
  'dist',
  'public',
  path.join('src', 'content'),
  path.join('www-tests', 'test-results'),
];

const dependencyTargets = [
  'node_modules',
  path.join('www-tests', 'node_modules'),
];

const targets = includeDeps ? [...generatedTargets, ...dependencyTargets] : generatedTargets;

for (const rel of targets) {
  const target = path.resolve(docsDir, rel);
  if (!target.startsWith(`${docsDir}${path.sep}`)) {
    throw new Error(`refusing to remove path outside docs dir: ${rel}`);
  }

  if (dryRun) {
    console.log(`would remove ${rel}`);
    continue;
  }

  await fs.rm(target, { recursive: true, force: true, maxRetries: 5, retryDelay: 100 });
  console.log(`removed ${rel}`);
}
