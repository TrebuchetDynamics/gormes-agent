import fs from 'node:fs/promises';
import path from 'node:path';
import { execFile } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { promisify } from 'node:util';

const docsDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const repoRoot = path.resolve(docsDir, '..', '..');
const outDir = path.resolve(docsDir, process.env.ASTRO_OUT_DIR || 'dist');
const target = path.join(outDir, 'building-gormes', 'architecture_plan', 'progress.json');
const execFileAsync = promisify(execFile);

const { stdout } = await execFileAsync('go', ['run', './cmd/progress', 'emit'], {
  cwd: repoRoot,
  maxBuffer: 16 * 1024 * 1024,
});

await fs.mkdir(path.dirname(target), { recursive: true });
await fs.writeFile(target, stdout);
