import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const docsDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const sourceDir = path.join(docsDir, 'content');
const targetDir = path.join(docsDir, 'src', 'content', 'docs');
const editBaseUrl = 'https://github.com/TrebuchetDynamics/gormes-agent/edit/main/docs/content/';

async function walk(dir, files = []) {
  for (const entry of await fs.readdir(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      await walk(full, files);
      continue;
    }
    if (entry.isFile()) files.push(full);
  }
  return files;
}

function targetPathFor(sourceFile) {
  const rel = path.relative(sourceDir, sourceFile);
  const parts = rel.split(path.sep);
  const last = parts.at(-1);
  if (last === '_index.md') {
    parts[parts.length - 1] = 'index.md';
  }
  return path.join(targetDir, ...parts);
}

function transformFrontmatter(frontmatter, sourceRel) {
  const lines = frontmatter.split(/\r?\n/);
  const out = [];
  let weight = '';
  let hasSidebar = false;
  let hasEditUrl = false;

  for (const line of lines) {
    const weightMatch = line.match(/^weight:\s*(\d+)\s*$/);
    if (weightMatch) {
      weight = weightMatch[1];
      continue;
    }
    if (line.match(/^sidebar:\s*$/)) hasSidebar = true;
    if (line.match(/^editUrl:\s*/)) hasEditUrl = true;
    out.push(line);
  }

  if (weight && !hasSidebar) {
    out.push('sidebar:');
    out.push(`  order: ${weight}`);
  }

  if (!hasEditUrl) {
    out.push(`editUrl: ${editBaseUrl}${sourceRel.split(path.sep).join('/')}`);
  }

  return out.join('\n').trim();
}

function transformMarkdown(raw, sourceRel) {
  const match = raw.match(/^---\r?\n([\s\S]*?)\r?\n---(\r?\n[\s\S]*)$/);
  if (!match) {
    const title = titleFromMarkdown(raw, sourceRel);
    return `---\ntitle: ${JSON.stringify(title)}\neditUrl: ${editBaseUrl}${sourceRel.split(path.sep).join('/')}\n---\n\n${raw}`;
  }

  const frontmatter = transformFrontmatter(match[1], sourceRel);
  return `---\n${frontmatter}\n---${match[2]}`;
}

function titleFromMarkdown(raw, sourceRel) {
  const heading = raw.match(/^#\s+(.+?)\s*$/m);
  if (heading) return heading[1].replace(/\s+#*$/, '');

  const base = path.basename(sourceRel, path.extname(sourceRel));
  return base
    .split(/[-_]+/)
    .filter(Boolean)
    .map((word) => word.slice(0, 1).toUpperCase() + word.slice(1))
    .join(' ');
}

await fs.rm(targetDir, { recursive: true, force: true });
await fs.mkdir(targetDir, { recursive: true });

for (const sourceFile of await walk(sourceDir)) {
  const rel = path.relative(sourceDir, sourceFile);
  if (path.extname(sourceFile) !== '.md') continue;

  const targetFile = targetPathFor(sourceFile);
  await fs.mkdir(path.dirname(targetFile), { recursive: true });
  const raw = await fs.readFile(sourceFile, 'utf8');
  await fs.writeFile(targetFile, transformMarkdown(raw, rel), 'utf8');
}
