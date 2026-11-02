import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import { sourceRel, transformMarkdown, walkFiles } from './starlight-content.mjs';

const docsDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const repoDir = path.resolve(docsDir, '..', '..');
const sourceDir = path.join(repoDir, 'hermes-agent', 'website', 'docs');
const targetDir = path.join(docsDir, 'content', 'upstream-hermes');
const supplements = new Set([
  'good-and-bad.md',
  'gormes-takeaways.md',
  'guides/build-a-hermes-plugin.md',
  'source-study.md',
  'user-guide/features/automation/_index.md',
  'user-guide/features/context/_index.md',
  'user-guide/features/platform/_index.md',
  'user-guide/features/providers/_index.md',
  'user-stories.md',
]);

const featureGroups = new Map([
  ['tools', ['acp', 'browser', 'code-execution', 'computer-use', 'delegation', 'image-generation', 'lsp', 'mcp', 'spotify', 'tool-gateway', 'tools', 'tts', 'vision', 'web-search', 'x-search']],
  ['providers', ['credential-pools', 'fallback-providers', 'provider-routing', 'subscription-proxy']],
  ['context', ['context-files', 'context-references', 'curator', 'honcho', 'memory', 'memory-providers', 'personality', 'skills']],
  ['automation', ['batch-processing', 'cron', 'goals', 'kanban', 'kanban-tutorial', 'kanban-worker-lanes']],
  ['platform', ['api-server', 'codex-app-server-runtime', 'extending-the-dashboard', 'skins', 'voice-mode', 'web-dashboard']],
  ['plugins', ['built-in-plugins', 'hooks', 'plugins']],
]);
const featureIndexSlugs = new Set(['plugins', 'tools']);

try {
  await fs.access(sourceDir);
} catch {
  throw new Error(`Hermes website docs source not found: ${sourceDir}`);
}

function relativeTarget(sourceFile) {
  const rel = sourceRel(sourceDir, sourceFile);
  if (rel === 'index.md' || rel === 'index.mdx') return '_index.md';
  if (rel.endsWith('/index.md') || rel.endsWith('/index.mdx')) {
    return `${rel.slice(0, rel.lastIndexOf('/') + 1)}_index.md`;
  }
  if (rel.endsWith('/_category_.json')) {
    return `${rel.slice(0, rel.lastIndexOf('/') + 1)}_index.md`;
  }
  let mapped = rel.endsWith('.mdx') ? `${rel.slice(0, -4)}.md` : rel;
  const feature = mapped.match(/^user-guide\/features\/([^/]+)\.md$/);
  if (feature) {
    for (const [group, slugs] of featureGroups) {
      if (slugs.includes(feature[1])) {
        const file = featureIndexSlugs.has(feature[1]) ? '_index.md' : `${feature[1]}.md`;
        mapped = `user-guide/features/${group}/${file}`;
        break;
      }
    }
  }
  return mapped;
}

function sanitizeMarkdown(raw) {
  return raw
    .replace(/\{\/\*[\s\S]*?\*\/\}/g, '')
    .replace(/^\s*<\/?(?:Tabs|TabItem)\b[^>]*>\s*$/gm, '')
    .replace(/^\s*<\/?[A-Z][A-Za-z0-9]*(?:\s+[^>]*?)?\s*\/?\s*>\s*$/gm, '')
    .replace(/^\s*:::(tip|info|note|warning|danger|caution|important)(?:\[([^\]]+)\])?(.*)$/gm, '> **$1$2$3**')
    .replace(/^\s*:::\s*$/gm, '')
    .replace(/className=/g, 'class=')
    .replace(/src=\{useBaseUrl\('([^']+)'\)\}/g, 'src="$1"')
    .replace(/\]\(\/(?!\/)/g, '](https://hermes-agent.nousresearch.com/')
    .replace(/\r\n/g, '\n')
    .split('\n')
    .map((line) => line.replace(/[ \t\r]+$/, ''))
    .join('\n')
    .replace(/\n{3,}/g, '\n\n')
    .trimEnd() + '\n';
}

const expected = new Set();
for (const sourceFile of await walkFiles(sourceDir)) {
  const rel = sourceRel(sourceDir, sourceFile);
  if (!(rel.endsWith('.md') || rel.endsWith('.mdx') || rel.endsWith('/_category_.json'))) continue;
  if (rel === 'user-guide/skills/godmode.md' || rel === 'user-guide/skills/bundled/red-teaming/red-teaming-godmode.md') continue;

  const targetRel = relativeTarget(sourceFile);
  expected.add(targetRel);
  if (rel === 'user-stories.md' || rel === 'user-stories.mdx') continue;
  const targetFile = path.join(targetDir, ...targetRel.split('/'));
  await fs.mkdir(path.dirname(targetFile), { recursive: true });
  if (rel.endsWith('/_category_.json')) {
    const category = JSON.parse(await fs.readFile(sourceFile, 'utf8'));
    const title = category.label ?? path.basename(path.dirname(sourceFile));
    await fs.writeFile(targetFile, `---\ntitle: ${JSON.stringify(title)}\n---\n`, 'utf8');
    continue;
  }
  const raw = await fs.readFile(sourceFile, 'utf8');
  await fs.writeFile(targetFile, sanitizeMarkdown(transformMarkdown(raw, `upstream-hermes/${rel}`)), 'utf8');
}

for (const file of await walkFiles(targetDir)) {
  if (path.extname(file) !== '.md') continue;
  const rel = sourceRel(targetDir, file);
  if (expected.has(rel) || supplements.has(rel)) continue;
  await fs.rm(file);
}
