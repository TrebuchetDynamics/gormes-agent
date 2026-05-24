import fs from 'node:fs/promises';
import path from 'node:path';
import { execFile } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { promisify } from 'node:util';

import { createProgressArtifactPlan } from './progress-artifact.mjs';

const docsDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const repoRoot = path.resolve(docsDir, '..', '..');
const { command, args, cwd, maxBuffer, target } = createProgressArtifactPlan({
  docsDir,
  repoRoot,
  env: process.env,
});
const execFileAsync = promisify(execFile);

const { stdout } = await execFileAsync(command, args, { cwd, maxBuffer });

await fs.mkdir(path.dirname(target), { recursive: true });
await fs.writeFile(target, stdout);
