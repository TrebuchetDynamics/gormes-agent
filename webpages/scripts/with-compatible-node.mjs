import { spawnSync } from 'node:child_process';

import { resolveExecutionEnv } from './compatible-node.mjs';

const command = process.argv[2];
const args = process.argv.slice(3);

if (!command) {
  console.error('usage: node ../scripts/with-compatible-node.mjs <command> [...args]');
  process.exit(64);
}

const result = spawnSync(command, args, {
  env: resolveExecutionEnv(),
  stdio: 'inherit',
  shell: process.platform === 'win32',
});

if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}

process.exit(result.status ?? 1);
