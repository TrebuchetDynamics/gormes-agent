import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import { sourceRel, targetPathForContentFile, transformMarkdown, walkFiles } from './starlight-content.mjs';

const docsDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const sourceDir = path.join(docsDir, 'content');
const targetDir = path.join(docsDir, 'src', 'content', 'docs');

await fs.rm(targetDir, { recursive: true, force: true, maxRetries: 5, retryDelay: 100 });
await fs.mkdir(targetDir, { recursive: true });

for (const sourceFile of await walkFiles(sourceDir)) {
  const rel = sourceRel(sourceDir, sourceFile);
  if (path.extname(sourceFile) !== '.md') continue;

  const targetFile = targetPathForContentFile(sourceDir, targetDir, sourceFile);
  await fs.mkdir(path.dirname(targetFile), { recursive: true });
  const raw = await fs.readFile(sourceFile, 'utf8');
  await fs.writeFile(targetFile, transformMarkdown(raw, rel), 'utf8');
}
