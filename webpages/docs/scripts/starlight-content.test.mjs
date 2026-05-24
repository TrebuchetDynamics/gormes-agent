import assert from 'node:assert/strict';
import path from 'node:path';
import { test } from 'node:test';

import {
  aliasesFor,
  frontmatterFor,
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

test('frontmatter aliases feed redirects without quotes', () => {
  const raw = `---\ntitle: Troubleshoot\naliases:\n  - /diagnose/\n  - '/doctor/'\n  - \"/logs/\"\n---\n\n# Troubleshoot\n`;

  assert.equal(frontmatterFor(raw).includes('aliases:'), true);
  assert.deepEqual(aliasesFor(frontmatterFor(raw)), ['/diagnose/', '/doctor/', '/logs/']);
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
