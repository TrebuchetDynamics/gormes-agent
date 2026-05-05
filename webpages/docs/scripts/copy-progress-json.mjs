import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const docsDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const outDir = path.resolve(docsDir, process.env.ASTRO_OUT_DIR || 'dist');
const source = path.join(docsDir, 'content', 'building-gormes', 'architecture_plan', 'progress.json');
const target = path.join(outDir, 'building-gormes', 'architecture_plan', 'progress.json');

await fs.mkdir(path.dirname(target), { recursive: true });
await fs.copyFile(source, target);
