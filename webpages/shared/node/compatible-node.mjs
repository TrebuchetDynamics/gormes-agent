import { existsSync, readdirSync } from 'node:fs';
import { homedir } from 'node:os';
import { delimiter, join } from 'node:path';

const realFs = { existsSync, readdirSync };

export function resolveExecutionEnv({
  currentVersion = process.versions.node,
  env = process.env,
  home = homedir(),
  fs = realFs,
} = {}) {
  const resolved = { ...env };
  if (nodeAtLeast(currentVersion, 22, 12)) {
    return resolved;
  }

  const bin = findCompatibleNodeBin({ home, fs });
  if (bin) {
    resolved.PATH = resolved.PATH ? `${bin}${delimiter}${resolved.PATH}` : bin;
  }
  return resolved;
}

export function findCompatibleNodeBin({ home = homedir(), fs = realFs } = {}) {
  const root = join(home, '.nvm', 'versions', 'node');
  let entries;
  try {
    entries = fs.readdirSync(root, { withFileTypes: true });
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
    if (!fs.existsSync(join(bin, 'node'))) {
      continue;
    }
    if (!best || compareVersions(version, bestVersion) > 0) {
      best = bin;
      bestVersion = version;
    }
  }
  return best;
}

export function nodeAtLeast(rawVersion, major, minor) {
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
