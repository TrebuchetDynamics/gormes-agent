import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import starlight from '@astrojs/starlight';
import { defineConfig } from 'astro/config';

import { aliasesFor, frontmatterFor, routeForContentFile, walkMarkdownFiles } from './scripts/starlight-content.mjs';

const docsDir = path.dirname(fileURLToPath(import.meta.url));
const canonicalContentDir = path.join(docsDir, 'content');

function collectRedirects() {
  const redirects = {};
  for (const file of walkMarkdownFiles(canonicalContentDir)) {
    const raw = fs.readFileSync(file, 'utf8');
    const destination = routeForContentFile(canonicalContentDir, file);
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
      // Keep the public IA curated. The content tree contains generated
      // progress, upstream archives, papers, and agent-control-plane material;
      // those stay reachable through hubs instead of becoming the sidebar.
      sidebar: [
        { label: 'Quickstart', slug: 'start-here' },
        {
          label: 'Install',
          items: [
            { label: 'Overview', slug: 'install' },
            { label: 'Linux & macOS', slug: 'install/linux-macos' },
            { label: 'Windows', slug: 'install/windows' },
            { label: 'Termux', slug: 'install/termux' },
            { label: 'From source', slug: 'install/from-source' },
            { label: 'Update and uninstall', slug: 'install/update-uninstall' },
          ],
        },
        {
          label: 'Configure',
          items: [
            { label: 'Overview', slug: 'configure' },
            { label: 'Setup wizard', slug: 'configure/setup-wizard' },
            { label: 'Providers', slug: 'configure/providers' },
            { label: 'Models and routing', slug: 'configure/models-routing' },
            { label: 'Profiles and workspaces', slug: 'configure/profiles-workspaces' },
            { label: 'Channel credentials', slug: 'configure/telegram' },
            { label: 'Secrets and local state', slug: 'configure/secrets-local-state' },
          ],
        },
        {
          label: 'Operate',
          items: [
            { label: 'Overview', slug: 'operate' },
            { label: 'First chat', slug: 'operate/first-chat' },
            { label: 'Local Ollama', slug: 'operate/local-ollama' },
            { label: 'Profiles for client work', slug: 'operate/profiles-client-work' },
            { label: 'Memory and sessions', slug: 'operate/memory-sessions' },
            { label: 'Telegram bot', slug: 'operate/telegram-bot' },
            { label: 'Multi-channel gateway', slug: 'operate/multi-channel-gateway' },
            { label: 'Channel bindings', slug: 'operate/channel-bindings' },
            { label: 'Fallback providers', slug: 'operate/fallback-providers' },
            { label: 'Dashboard, status, logs', slug: 'operate/dashboard-status-logs' },
          ],
        },
        {
          label: 'Troubleshoot',
          items: [
            { label: 'Overview', slug: 'troubleshooting' },
            { label: 'Doctor', slug: 'troubleshooting/doctor' },
            { label: 'Diagnose broken install', slug: 'troubleshooting/diagnose' },
            { label: 'Common errors', slug: 'troubleshooting/common-errors' },
            { label: 'Logs', slug: 'troubleshooting/logs' },
          ],
        },
        {
          label: 'Reference',
          items: [
            { label: 'Overview', slug: 'reference' },
            { label: 'Status & Roadmap', slug: 'reference/status-readiness' },
            { label: 'CLI commands', slug: 'cli' },
            { label: 'Config file', slug: 'configure/config-file' },
            { label: 'Environment variables', slug: 'configure/environment' },
            { label: 'Paths and logs', slug: 'configure/paths' },
            { label: 'Glossary', slug: 'reference/glossary' },
          ],
        },
        {
          label: 'Concepts',
          items: [
            { label: 'Overview', slug: 'concepts' },
            { label: 'Why Gormes', slug: 'why-gormes' },
            { label: 'Runtime model', slug: 'architecture/runtime-model' },
            { label: 'SQLite memory and sessions', slug: 'architecture/memory-and-sessions' },
            { label: 'Gateway pipeline', slug: 'architecture/gateway-pipeline' },
            { label: 'Tool execution', slug: 'architecture/tool-execution' },
            { label: 'Tool output compaction', slug: 'architecture/tool-output-compaction' },
            { label: 'Hermes compatibility', slug: 'architecture/hermes-parity' },
            { label: 'TOON context encoding', slug: 'architecture/toon-context-encoding' },
          ],
        },
        {
          label: 'Build Gormes',
          items: [
            { label: 'Overview', slug: 'building-gormes' },
            { label: 'Repo layout', slug: 'development/repo-layout' },
            { label: 'Testing', slug: 'development/testing' },
            { label: 'Parity workflow', slug: 'development/parity-workflow' },
            { label: 'Implementation roadmap', slug: 'building-gormes/implementation-roadmap' },
            { label: 'Builder queue', slug: 'building-gormes/builder-loop/agent-queue' },
            { label: 'Progress schema', slug: 'building-gormes/builder-loop/progress-schema' },
            { label: 'Architecture archive', slug: 'building-gormes/architecture_plan' },
          ],
        },
        {
          label: 'Archive & Research',
          items: [
            { label: 'Overview', slug: 'archive' },
            { label: 'Upstream Hermes archive', slug: 'upstream-hermes' },
            { label: 'Agent research survey', link: '/papers/' },
            { label: 'Reading list', slug: 'papers/reading-list' },
          ],
        },
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
