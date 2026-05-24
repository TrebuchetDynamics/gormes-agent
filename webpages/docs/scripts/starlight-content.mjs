import fs from 'node:fs';
import fsp from 'node:fs/promises';
import path from 'node:path';

const editBaseUrl = 'https://github.com/TrebuchetDynamics/gormes-agent/edit/main/webpages/docs/content/';

export async function walkFiles(dir, files = []) {
  for (const entry of await fsp.readdir(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      await walkFiles(full, files);
      continue;
    }
    if (entry.isFile()) files.push(full);
  }
  return files;
}

export function walkMarkdownFiles(dir, files = []) {
  if (!fs.existsSync(dir)) return files;
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      walkMarkdownFiles(full, files);
      continue;
    }
    if (entry.isFile() && entry.name.endsWith('.md')) {
      files.push(full);
    }
  }
  return files;
}

function toPosixPath(value) {
  return value.split(path.sep).join('/');
}

export function sourceRel(sourceDir, sourceFile) {
  return toPosixPath(path.relative(sourceDir, sourceFile));
}

export function targetPathForContentFile(sourceDir, targetDir, sourceFile) {
  const rel = sourceRel(sourceDir, sourceFile);
  const parts = rel.split('/');
  const last = parts.at(-1);
  if (last === '_index.md') {
    parts[parts.length - 1] = 'index.md';
  }
  return path.join(targetDir, ...parts);
}

export function routeForContentFile(contentDir, file) {
  const rel = sourceRel(contentDir, file);
  if (rel === '_index.md') return '/';
  if (rel.endsWith('/_index.md')) {
    return `/${rel.slice(0, -'_index.md'.length)}`;
  }
  return `/${rel.replace(/\.md$/, '/')}`;
}

export function frontmatterFor(raw) {
  const match = raw.match(/^---\r?\n([\s\S]*?)\r?\n---/);
  return match ? match[1] : '';
}

export function aliasesFor(frontmatter) {
  const match = frontmatter.match(/^aliases:\s*\r?\n((?:\s+-\s+.+\r?\n?)+)/m);
  if (!match) return [];
  return match[1]
    .split(/\r?\n/)
    .map((line) => line.trim().replace(/^-\s+/, '').trim())
    .filter(Boolean)
    .map((alias) => alias.replace(/^['"]|['"]$/g, ''));
}

export function redirectsForContentDir(contentDir) {
  const redirects = {};
  for (const file of walkMarkdownFiles(contentDir)) {
    const raw = fs.readFileSync(file, 'utf8');
    const destination = routeForContentFile(contentDir, file);
    for (const alias of aliasesFor(frontmatterFor(raw))) {
      redirects[alias] = destination;
    }
  }
  return redirects;
}

function editUrlForSourceRel(sourceRel) {
  return `${editBaseUrl}${toPosixPath(sourceRel)}`;
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
    out.push(`editUrl: ${editUrlForSourceRel(sourceRel)}`);
  }

  return out.join('\n').trim();
}

export function transformMarkdown(raw, sourceRel) {
  const match = raw.match(/^---\r?\n([\s\S]*?)\r?\n---(\r?\n[\s\S]*)$/);
  if (!match) {
    const title = titleFromMarkdown(raw, sourceRel);
    return `---\ntitle: ${JSON.stringify(title)}\neditUrl: ${editUrlForSourceRel(sourceRel)}\n---\n\n${raw}`;
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
