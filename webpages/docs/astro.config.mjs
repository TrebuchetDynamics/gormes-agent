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
        { label: 'Home', link: '/' },
        { label: 'Getting Started', autogenerate: { directory: 'getting-started' } },
        { label: 'Operate', autogenerate: { directory: 'guides' } },
        { label: 'Using Gormes', autogenerate: { directory: 'using-gormes' } },
        { label: 'Reference', autogenerate: { directory: 'reference' } },
        { label: 'Architecture', autogenerate: { directory: 'architecture' } },
        { label: 'Development', autogenerate: { directory: 'development' } },
        { label: 'Parity', collapsed: true, autogenerate: { directory: 'parity' } },
        {
          label: 'Building Gormes',
          collapsed: true,
          autogenerate: { directory: 'building-gormes', collapsed: true },
        },
        {
          label: 'Upstream Hermes',
          collapsed: true,
          autogenerate: { directory: 'upstream-hermes', collapsed: true },
        },
        { label: 'Papers', collapsed: true, autogenerate: { directory: 'papers' } },
        { label: 'Why Gormes', link: '/why-gormes/' },
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
