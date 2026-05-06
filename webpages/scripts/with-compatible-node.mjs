import { spawnSync } from 'node:child_process';
import { existsSync, readdirSync } from 'node:fs';
import { homedir } from 'node:os';
import { delimiter, join } from 'node:path';

const command = process.argv[2];
const args = process.argv.slice(3);

if (!command) {
  console.error('usage: node ../scripts/with-compatible-node.mjs <command> [...args]');
  process.exit(64);
}

const env = { ...process.env };
if (!nodeAtLeast(process.versions.node, 22, 12)) {
  const bin = findCompatibleNodeBin();
  if (bin) {
    env.PATH = env.PATH ? `${bin}${delimiter}${env.PATH}` : bin;
  }
}

const result = spawnSync(command, args, {
  env,
  stdio: 'inherit',
  shell: process.platform === 'win32',
});

if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}

process.exit(result.status ?? 1);

function findCompatibleNodeBin() {
  const root = join(homedir(), '.nvm', 'versions', 'node');
  let entries;
  try {
    entries = readdirSync(root, { withFileTypes: true });
  } catch {
    return '';
  }

  let best = '';
  let bestVersion = [0, 0, 0];
  for (const entry of entries) {
    if (!entry.isDirectory()) {
      continue;
    }
    const version = nodeVersionParts(entry.name);
    if (!versionAtLeast(version, 22, 12)) {
      continue;
    }
    const bin = join(root, entry.name, 'bin');
    if (!existsSync(join(bin, 'node'))) {
      continue;
    }
    if (!best || compareVersions(version, bestVersion) > 0) {
      best = bin;
      bestVersion = version;
    }
  }
  return best;
}

function nodeAtLeast(rawVersion, major, minor) {
  return versionAtLeast(nodeVersionParts(rawVersion), major, minor);
}

function versionAtLeast(version, major, minor) {
  if (version[0] !== major) {
    return version[0] > major;
  }
  return version[1] >= minor;
}

function nodeVersionParts(rawVersion) {
  const parts = rawVersion
    .trim()
    .replace(/^v/, '')
    .split('.')
    .slice(0, 3)
    .map((part) => Number.parseInt(part, 10) || 0);
  while (parts.length < 3) {
    parts.push(0);
  }
  return parts;
}

function compareVersions(left, right) {
  for (let i = 0; i < 3; i += 1) {
    if (left[i] > right[i]) {
      return 1;
    }
    if (left[i] < right[i]) {
      return -1;
    }
  }
  return 0;
}
