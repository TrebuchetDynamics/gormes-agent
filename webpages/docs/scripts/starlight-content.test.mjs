import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { test } from 'node:test';

import {
  aliasesFor,
  frontmatterFor,
  redirectsForContentDir,
  routeForContentFile,
  targetPathForContentFile,
  transformMarkdown,
} from './starlight-content.mjs';

test('content routes preserve Starlight public URL contracts', () => {
  const contentDir = path.join('repo', 'webpages', 'docs', 'content');

  assert.equal(routeForContentFile(contentDir, path.join(contentDir, '_index.md')), '/');
  assert.equal(routeForContentFile(contentDir, path.join(contentDir, 'install', '_index.md')), '/install/');
  assert.equal(routeForContentFile(contentDir, path.join(contentDir, 'install', 'linux-macos.md')), '/install/linux-macos/');
});

test('content targets map Hugo-style indexes into Starlight indexes', () => {
  const sourceDir = path.join('repo', 'webpages', 'docs', 'content');
  const targetDir = path.join('repo', 'webpages', 'docs', 'src', 'content', 'docs');

  assert.equal(
    targetPathForContentFile(sourceDir, targetDir, path.join(sourceDir, '_index.md')),
    path.join(targetDir, 'index.md'),
  );
  assert.equal(
    targetPathForContentFile(sourceDir, targetDir, path.join(sourceDir, 'install', '_index.md')),
    path.join(targetDir, 'install', 'index.md'),
  );
  assert.equal(
    targetPathForContentFile(sourceDir, targetDir, path.join(sourceDir, 'install', 'linux-macos.md')),
    path.join(targetDir, 'install', 'linux-macos.md'),
  );
});

// The upstream mirror stays path-compatible with Hermes for coverage checks,
// while the generated Starlight tree groups the noisy features folder by
// responsibility. Category landing pages keep their public route as index.md.
// Leaves that move deeper get automatic redirects from their old URLs.

test('upstream Hermes feature docs target responsibility subfolders', () => {
  const sourceDir = path.join('repo', 'webpages', 'docs', 'content');
  const targetDir = path.join('repo', 'webpages', 'docs', 'src', 'content', 'docs');

  assert.equal(
    targetPathForContentFile(sourceDir, targetDir, path.join(sourceDir, 'upstream-hermes', 'user-guide', 'features', 'tools', '_index.md')),
    path.join(targetDir, 'upstream-hermes', 'user-guide', 'features', 'tools', 'index.md'),
  );
  assert.equal(
    targetPathForContentFile(sourceDir, targetDir, path.join(sourceDir, 'upstream-hermes', 'user-guide', 'features', 'platform', 'api-server.md')),
    path.join(targetDir, 'upstream-hermes', 'user-guide', 'features', 'platform', 'api-server.md'),
  );
});

test('CLI command docs target responsibility subfolders while preserving the CLI index', () => {
  const sourceDir = path.join('repo', 'webpages', 'docs', 'content');
  const targetDir = path.join('repo', 'webpages', 'docs', 'src', 'content', 'docs');

  assert.equal(
    targetPathForContentFile(sourceDir, targetDir, path.join(sourceDir, 'cli', '_index.md')),
    path.join(targetDir, 'cli', 'index.md'),
  );
  assert.equal(
    targetPathForContentFile(sourceDir, targetDir, path.join(sourceDir, 'cli', 'auth.md')),
    path.join(targetDir, 'cli', 'setup', 'auth.md'),
  );
  assert.equal(
    targetPathForContentFile(sourceDir, targetDir, path.join(sourceDir, 'cli', 'setup', 'auth.md')),
    path.join(targetDir, 'cli', 'setup', 'auth.md'),
  );
  assert.equal(
    targetPathForContentFile(sourceDir, targetDir, path.join(sourceDir, 'cli', 'telegram.md')),
    path.join(targetDir, 'cli', 'channels', 'telegram.md'),
  );
  assert.equal(
    targetPathForContentFile(sourceDir, targetDir, path.join(sourceDir, 'cli', 'mcp.md')),
    path.join(targetDir, 'cli', 'extensions', 'mcp.md'),
  );
});

test('frontmatter aliases feed redirects without quotes', () => {
  const raw = `---\ntitle: Troubleshoot\naliases:\n  - /diagnose/\n  - '/doctor/'\n  - \"/logs/\"\n---\n\n# Troubleshoot\n`;

  assert.equal(frontmatterFor(raw).includes('aliases:'), true);
  assert.deepEqual(aliasesFor(frontmatterFor(raw)), ['/diagnose/', '/doctor/', '/logs/']);
});

test('content aliases produce Starlight redirect map', async () => {
  const contentDir = await fs.mkdtemp(path.join(os.tmpdir(), 'gormes-doc-content-'));
  await fs.mkdir(path.join(contentDir, 'install'), { recursive: true });
  await fs.writeFile(
    path.join(contentDir, 'install', 'linux-macos.md'),
    `---\ntitle: Linux and macOS\naliases:\n  - /setup/\n  - '/install.sh/'\n---\n\n# Linux and macOS\n`,
  );
  await fs.writeFile(path.join(contentDir, '_index.md'), `---\ntitle: Home\n---\n\n# Home\n`);

  assert.deepEqual(redirectsForContentDir(contentDir), {
    '/setup/': '/install/linux-macos/',
    '/install.sh/': '/install/linux-macos/',
  });
});

test('organized upstream Hermes feature docs redirect old public routes', async () => {
  const contentDir = await fs.mkdtemp(path.join(os.tmpdir(), 'gormes-doc-content-'));
  const featuresDir = path.join(contentDir, 'upstream-hermes', 'user-guide', 'features');
  await fs.mkdir(featuresDir, { recursive: true });
  await fs.mkdir(path.join(featuresDir, 'platform'), { recursive: true });
  await fs.mkdir(path.join(featuresDir, 'tools'), { recursive: true });
  await fs.writeFile(path.join(featuresDir, 'platform', 'api-server.md'), `---\ntitle: API Server\n---\n\n# API Server\n`);
  await fs.writeFile(path.join(featuresDir, 'tools', '_index.md'), `---\ntitle: Tools\n---\n\n# Tools\n`);

  assert.deepEqual(redirectsForContentDir(contentDir), {
    '/upstream-hermes/user-guide/features/api-server/': '/upstream-hermes/user-guide/features/platform/api-server/',
  });
});

test('organized CLI command docs redirect old public routes', async () => {
  const contentDir = await fs.mkdtemp(path.join(os.tmpdir(), 'gormes-doc-content-'));
  const cliDir = path.join(contentDir, 'cli');
  await fs.mkdir(cliDir, { recursive: true });
  await fs.mkdir(path.join(cliDir, 'setup'), { recursive: true });
  await fs.mkdir(path.join(cliDir, 'channels'), { recursive: true });
  await fs.writeFile(path.join(cliDir, 'setup', 'auth.md'), `---\ntitle: Auth\n---\n\n# Auth\n`);
  await fs.writeFile(path.join(cliDir, 'channels', 'telegram.md'), `---\ntitle: Telegram\n---\n\n# Telegram\n`);

  assert.deepEqual(redirectsForContentDir(contentDir), {
    '/cli/auth/': '/cli/setup/auth/',
    '/cli/telegram/': '/cli/channels/telegram/',
  });
});

test('markdown transform adds Starlight edit URLs and sidebar order', () => {
  const transformed = transformMarkdown(`---\ntitle: Install\nweight: 20\n---\n\n# Install\n`, 'install/_index.md');

  assert.equal(transformed.includes('weight: 20'), false);
  assert.equal(transformed.includes('sidebar:\n  order: 20'), true);
  assert.equal(
    transformed.includes('editUrl: https://github.com/TrebuchetDynamics/gormes-agent/edit/main/webpages/docs/content/install/_index.md'),
    true,
  );
});

test('markdown transform creates frontmatter for plain markdown', () => {
  const transformed = transformMarkdown('# First chat\n\nStart here.\n', 'operate/first-chat.md');

  assert.equal(transformed.startsWith('---\ntitle: "First chat"\n'), true);
  assert.equal(
    transformed.includes('editUrl: https://github.com/TrebuchetDynamics/gormes-agent/edit/main/webpages/docs/content/operate/first-chat.md'),
    true,
  );
});

test('markdown transform rewrites relative links when CLI docs move into subfolders', () => {
  const raw = `---\ntitle: Channels\n---\n\n[CLI](../) [gateway](../gateway/) [telegram guide](../../configure/telegram/)\n`;
  const transformed = transformMarkdown(raw, 'cli/channels.md');

  assert.equal(transformed.includes('[CLI](../../)'), true);
  assert.equal(transformed.includes('[gateway](../../runtime/gateway/)'), true);
  assert.equal(transformed.includes('[telegram guide](../../../configure/telegram/)'), true);
});

test('markdown transform keeps legacy CLI link intent after source docs are organized', () => {
  const raw = `---\ntitle: Auth\n---\n\n[CLI](../) [setup](../setup/) [environment](../../configure/environment/)\n`;
  const transformed = transformMarkdown(raw, 'cli/setup/auth.md');

  assert.equal(transformed.includes('[CLI](../../)'), true);
  assert.equal(transformed.includes('[setup](../setup/)'), true);
  assert.equal(transformed.includes('[environment](../../../configure/environment/)'), true);
});
