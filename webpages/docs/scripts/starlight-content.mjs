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

const upstreamHermesFeatureGroups = new Map(Object.entries({
  tools: [
    'acp',
    'browser',
    'code-execution',
    'computer-use',
    'delegation',
    'image-generation',
    'lsp',
    'mcp',
    'spotify',
    'tool-gateway',
    'tools',
    'tts',
    'vision',
    'web-search',
    'x-search',
  ],
  providers: ['credential-pools', 'fallback-providers', 'provider-routing', 'subscription-proxy'],
  context: ['context-files', 'context-references', 'curator', 'honcho', 'memory', 'memory-providers', 'personality', 'skills'],
  automation: ['batch-processing', 'cron', 'goals', 'kanban', 'kanban-tutorial', 'kanban-worker-lanes'],
  platform: ['api-server', 'codex-app-server-runtime', 'extending-the-dashboard', 'skins', 'voice-mode', 'web-dashboard'],
  plugins: ['built-in-plugins', 'hooks', 'plugins'],
}));

const upstreamHermesFeatureIndexSlugs = new Set(['plugins', 'tools']);

const cliCommandGroups = new Map(Object.entries({
  core: ['chat', 'completion', 'doctor', 'logs', 'status', 'version'],
  setup: ['auth', 'config', 'fallback', 'logout', 'model', 'profile', 'providers', 'secrets', 'security', 'setup'],
  runtime: ['agent', 'checkpoints', 'cron', 'dashboard', 'gateway', 'goncho', 'kanban', 'memory', 'restore', 'session', 'system', 'uninstall', 'update', 'usage'],
  channels: ['channels', 'navivox', 'slack', 'telegram', 'whatsapp'],
  extensions: ['acp', 'claw', 'curator', 'mcp', 'migrate', 'plugins', 'skills'],
}));

const cliCommandToGroup = new Map(
  [...cliCommandGroups].flatMap(([group, commands]) => commands.map((command) => [command, group])),
);

function organizedCliRel(rel) {
  if (rel === 'cli/_index.md') return rel;
  const match = rel.match(/^cli\/([^/]+)\.md$/);
  if (!match) return rel;

  const group = cliCommandToGroup.get(match[1]);
  if (!group) return rel;
  return `cli/${group}/${match[1]}.md`;
}

function organizedUpstreamHermesFeatureRel(rel) {
  const organizedMatch = rel.match(/^upstream-hermes\/user-guide\/features\/([^/]+)\/_index\.md$/);
  if (organizedMatch && upstreamHermesFeatureIndexSlugs.has(organizedMatch[1])) {
    return rel.replace(/\/_index\.md$/, '/index.md');
  }
  if (rel.match(/^upstream-hermes\/user-guide\/features\/[^/]+\/[^/]+\.md$/)) return rel;

  const match = rel.match(/^upstream-hermes\/user-guide\/features\/([^/]+)\.md$/);
  if (!match) return rel;

  const slug = match[1];
  for (const [group, slugs] of upstreamHermesFeatureGroups) {
    if (!slugs.includes(slug)) continue;
    const fileName = upstreamHermesFeatureIndexSlugs.has(slug) ? 'index.md' : `${slug}.md`;
    return `upstream-hermes/user-guide/features/${group}/${fileName}`;
  }
  return rel;
}

function legacyUpstreamHermesFeatureRoute(rel) {
  const indexMatch = rel.match(/^upstream-hermes\/user-guide\/features\/([^/]+)\/_index\.md$/);
  if (indexMatch && upstreamHermesFeatureIndexSlugs.has(indexMatch[1])) {
    return `/upstream-hermes/user-guide/features/${indexMatch[1]}/`;
  }

  const leafMatch = rel.match(/^upstream-hermes\/user-guide\/features\/[^/]+\/([^/]+)\.md$/);
  if (!leafMatch) return '';
  return `/upstream-hermes/user-guide/features/${leafMatch[1]}/`;
}

function organizedRel(rel) {
  return organizedCliRel(organizedUpstreamHermesFeatureRel(rel));
}

function indexRelForTarget(rel) {
  const parts = rel.split('/');
  if (parts.at(-1) === '_index.md') parts[parts.length - 1] = 'index.md';
  return parts.join('/');
}

export function targetPathForContentFile(sourceDir, targetDir, sourceFile) {
  const rel = organizedRel(sourceRel(sourceDir, sourceFile));
  return path.join(targetDir, ...indexRelForTarget(rel).split('/'));
}

function routeForRel(rel) {
  if (rel === '_index.md') return '/';
  if (rel.endsWith('/_index.md')) {
    return `/${rel.slice(0, -'_index.md'.length)}`;
  }
  if (rel.endsWith('/index.md')) {
    return `/${rel.slice(0, -'index.md'.length)}`;
  }
  return `/${rel.replace(/\.md$/, '/')}`;
}

export function routeForContentFile(contentDir, file) {
  return routeForRel(organizedRel(sourceRel(contentDir, file)));
}

function sourceRouteForContentFile(contentDir, file) {
  return routeForRel(sourceRel(contentDir, file));
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
    const rel = sourceRel(contentDir, file);
    const destination = routeForContentFile(contentDir, file);
    const legacyFeatureRoute = legacyUpstreamHermesFeatureRoute(rel);
    if (legacyFeatureRoute && legacyFeatureRoute !== destination) redirects[legacyFeatureRoute] = destination;
    const sourceRoute = sourceRouteForContentFile(contentDir, file);
    if (sourceRoute !== destination) redirects[sourceRoute] = destination;
    for (const alias of aliasesFor(frontmatterFor(raw))) {
      redirects[alias] = destination;
    }
  }
  return redirects;
}

function editUrlForSourceRel(sourceRel) {
  return `${editBaseUrl}${toPosixPath(sourceRel)}`;
}

function routePathForRel(rel) {
  const route = routeForRel(rel);
  return route.endsWith('/') ? route : `${route}/`;
}

function organizedRouteForPath(routePath) {
  const cliMatch = routePath.match(/^\/cli\/([^/]+)\/?$/);
  if (cliMatch) {
    const group = cliCommandToGroup.get(cliMatch[1]);
    if (group) return `/cli/${group}/${cliMatch[1]}/`;
  }
  return routePath;
}

function relativeRoute(fromRoute, toRoute) {
  const from = fromRoute.replace(/^\//, '').replace(/\/$/, '');
  const to = toRoute.replace(/^\//, '').replace(/\/$/, '');
  const rel = path.posix.relative(from || '.', to || '.');
  if (!rel) return './';
  const normalized = rel.startsWith('.') ? rel : `./${rel}`;
  return `${normalized}/`;
}

function rewriteRelativeLinks(markdown, sourceRel) {
  const sourceRoute = routePathForRel(sourceRel);
  const organizedSourceRoute = routePathForRel(organizedRel(sourceRel));

  return markdown.replace(/\]\(([^)]+)\)/g, (match, href) => {
    if (
      href.startsWith('#') ||
      href.startsWith('/') ||
      href.includes('://') ||
      href.startsWith('mailto:') ||
      href.startsWith('tel:')
    ) {
      return match;
    }

    const [pathPart, suffix = ''] = href.split(/(?=[?#])/, 2);
    if (!pathPart || pathPart.startsWith('#')) return match;

    const resolved = new URL(pathPart, `https://docs.gormes.ai${sourceRoute}`).pathname;
    const organizedTarget = organizedRouteForPath(resolved);
    if (organizedTarget === resolved && organizedSourceRoute === sourceRoute) return match;

    return `](${relativeRoute(organizedSourceRoute, organizedTarget)}${suffix})`;
  });
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
    return `---\ntitle: ${JSON.stringify(title)}\neditUrl: ${editUrlForSourceRel(sourceRel)}\n---\n\n${rewriteRelativeLinks(raw, sourceRel)}`;
  }

  const frontmatter = transformFrontmatter(match[1], sourceRel);
  return `---\n${frontmatter}\n---${rewriteRelativeLinks(match[2], sourceRel)}`;
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
