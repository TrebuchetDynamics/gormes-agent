import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import starlight from '@astrojs/starlight';
import { defineConfig } from 'astro/config';

const docsDir = path.dirname(fileURLToPath(import.meta.url));
const canonicalContentDir = path.join(docsDir, 'content');

function walkMarkdown(dir, files = []) {
  if (!fs.existsSync(dir)) return files;
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      walkMarkdown(full, files);
      continue;
    }
    if (entry.isFile() && entry.name.endsWith('.md')) {
      files.push(full);
    }
  }
  return files;
}

function routeForContentFile(file) {
  const rel = path.relative(canonicalContentDir, file).split(path.sep).join('/');
  if (rel === '_index.md') return '/';
  if (rel.endsWith('/_index.md')) {
    return `/${rel.slice(0, -'_index.md'.length)}`;
  }
  return `/${rel.replace(/\.md$/, '/')}`;
}

function frontmatterFor(raw) {
  const match = raw.match(/^---\r?\n([\s\S]*?)\r?\n---/);
  return match ? match[1] : '';
}

function aliasesFor(frontmatter) {
  const match = frontmatter.match(/^aliases:\s*\r?\n((?:\s+-\s+.+\r?\n?)+)/m);
  if (!match) return [];
  return match[1]
    .split(/\r?\n/)
    .map((line) => line.trim().replace(/^-\s+/, '').trim())
    .filter(Boolean)
    .map((alias) => alias.replace(/^['"]|['"]$/g, ''));
}

function collectRedirects() {
  const redirects = {};
  for (const file of walkMarkdown(canonicalContentDir)) {
    const raw = fs.readFileSync(file, 'utf8');
    const destination = routeForContentFile(file);
    for (const alias of aliasesFor(frontmatterFor(raw))) {
      redirects[alias] = destination;
    }
  }
  return redirects;
}

export default defineConfig({
  site: 'https://docs.gormes.ai',
  outDir: process.env.ASTRO_OUT_DIR || './dist',
  publicDir: './static',
  redirects: collectRedirects(),
  integrations: [
    starlight({
      title: 'Gormes Docs',
      description: 'Install, configure, operate, and extend the Go-native Gormes runtime.',
      favicon: '/favicon.ico',
      customCss: ['./src/styles/starlight.css'],
      lastUpdated: false,
      pagefind: true,
      tableOfContents: {
        minHeadingLevel: 2,
        maxHeadingLevel: 4,
      },
      social: [
        {
          icon: 'github',
          label: 'GitHub',
          href: 'https://github.com/TrebuchetDynamics/gormes-agent',
        },
      ],
      sidebar: [
        { label: 'Start here', slug: 'start-here' },
        {
          label: 'Install',
          items: [
            { label: 'Overview', slug: 'install' },
            { label: 'Linux & macOS', slug: 'install/linux-macos' },
            { label: 'Windows', slug: 'install/windows' },
            { label: 'From source', slug: 'install/from-source' },
          ],
        },
        {
          label: 'Configure',
          items: [
            { label: 'Overview', slug: 'configure' },
            { label: 'Config file', slug: 'configure/config-file' },
            { label: 'Environment', slug: 'configure/environment' },
            { label: 'Providers', slug: 'configure/providers' },
            { label: 'Telegram', slug: 'configure/telegram' },
            { label: 'Paths & logs', slug: 'configure/paths' },
          ],
        },
        {
          label: 'CLI reference',
          autogenerate: { directory: 'cli' },
        },
        {
          label: 'Recipes',
          items: [
            { label: 'Overview', slug: 'recipes' },
            { label: 'Smoke-test offline', slug: 'recipes/doctor-offline' },
            { label: 'First turn', slug: 'recipes/first-turn' },
            { label: 'Telegram bot', slug: 'recipes/telegram-bot' },
            { label: 'Diagnose a broken install', slug: 'recipes/diagnose' },
            { label: 'Migrate from Hermes', slug: 'recipes/migrate-hermes' },
            { label: 'Profiles', slug: 'recipes/profiles' },
            { label: 'Multi-channel gateway', slug: 'recipes/multi-channel' },
            { label: 'Channel bindings', slug: 'recipes/bindings' },
            { label: 'Fallback provider chain', slug: 'recipes/fallback' },
            { label: 'Local Ollama', slug: 'recipes/local-ollama' },
          ],
        },
        {
          label: 'Troubleshooting',
          items: [
            { label: 'Overview', slug: 'troubleshooting' },
            { label: 'Doctor', slug: 'troubleshooting/doctor' },
            { label: 'Common errors', slug: 'troubleshooting/common-errors' },
            { label: 'Logs', slug: 'troubleshooting/logs' },
          ],
        },
        { label: 'Why Gormes', slug: 'why-gormes' },
      ],
      head: [
        { tag: 'link', attrs: { rel: 'icon', href: '/favicon.ico' } },
        { tag: 'link', attrs: { rel: 'icon', type: 'image/png', sizes: '16x16', href: '/favicon-16x16.png' } },
        { tag: 'link', attrs: { rel: 'icon', type: 'image/png', sizes: '32x32', href: '/favicon-32x32.png' } },
        { tag: 'link', attrs: { rel: 'apple-touch-icon', href: '/apple-touch-icon.png' } },
        { tag: 'meta', attrs: { property: 'og:site_name', content: 'Gormes Docs' } },
        { tag: 'meta', attrs: { property: 'og:image', content: 'https://docs.gormes.ai/social-card.png' } },
        { tag: 'meta', attrs: { property: 'og:image:width', content: '1200' } },
        { tag: 'meta', attrs: { property: 'og:image:height', content: '630' } },
        { tag: 'meta', attrs: { name: 'twitter:card', content: 'summary_large_image' } },
        { tag: 'meta', attrs: { name: 'twitter:image', content: 'https://docs.gormes.ai/social-card.png' } },
      ],
    }),
  ],
});
