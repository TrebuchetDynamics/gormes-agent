import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { mkdtempSync, readFileSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { test } from 'node:test';

import { buildDryRunPost, parseFeed } from './publish.mjs';

const fixtureFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>TrebuchetDynamics Engineering</title>
    <link>https://engineering.trebuchetdynamics.com/</link>
    <description>Engineering notes from the validation-gated agentic porting loop behind Gormes.</description>
    <language>en-us</language>
    <item><title>Autonomous Hermes-porting loop</title><link>https://engineering.trebuchetdynamics.com/posts/autonomous-hermes-porting-loop/</link><guid>https://engineering.trebuchetdynamics.com/posts/autonomous-hermes-porting-loop/</guid><pubDate>Wed, 13 May 2026 00:00:00 GMT</pubDate><description>How TrebuchetDynamics turns Gormes progress rows into small, test-proven Hermes-parity slices.</description></item>
  </channel>
</rss>`;

test('parseFeed extracts the first blog item from the RSS feed', () => {
  const feed = parseFeed(fixtureFeed);

  assert.equal(feed.title, 'TrebuchetDynamics Engineering');
  assert.equal(feed.items.length, 1);
  assert.deepEqual(feed.items[0], {
    title: 'Autonomous Hermes-porting loop',
    link: 'https://engineering.trebuchetdynamics.com/posts/autonomous-hermes-porting-loop/',
    guid: 'https://engineering.trebuchetdynamics.com/posts/autonomous-hermes-porting-loop/',
    pubDate: 'Wed, 13 May 2026 00:00:00 GMT',
    description: 'How TrebuchetDynamics turns Gormes progress rows into small, test-proven Hermes-parity slices.',
  });
});

test('buildDryRunPost emits deterministic social copy and idempotency evidence', () => {
  const post = buildDryRunPost(parseFeed(fixtureFeed), { feedPath: 'webpages/blog/dist/feed.xml' });

  assert.equal(post.mode, 'dry-run');
  assert.equal(post.platform, 'operator-selected-social');
  assert.equal(post.source_feed, 'webpages/blog/dist/feed.xml');
  assert.equal(post.canonical_url, 'https://engineering.trebuchetdynamics.com/posts/autonomous-hermes-porting-loop/');
  assert.equal(post.idempotency_key, 'td-blog:5bb87e716f8d6d69');
  assert.equal(
    post.post_text,
    'New TrebuchetDynamics Engineering post: Autonomous Hermes-porting loop\nhttps://engineering.trebuchetdynamics.com/posts/autonomous-hermes-porting-loop/',
  );
  assert.equal(post.network_publish, false);
  assert.equal(post.secret_policy, 'no social tokens are read in dry-run mode');
});

test('CLI dry-run writes the preview JSON without publishing', () => {
  const dir = mkdtempSync(join(tmpdir(), 'gormes-social-dry-run-'));
  const feedPath = join(dir, 'feed.xml');
  const outPath = join(dir, 'post.json');
  writeFileSync(feedPath, fixtureFeed);

  const stdout = execFileSync(
    process.execPath,
    ['scripts/blog-feed-social/publish.mjs', '--dry-run', '--feed', feedPath, '--out', outPath],
    { cwd: new URL('../..', import.meta.url).pathname, encoding: 'utf8' },
  );
  const post = JSON.parse(readFileSync(outPath, 'utf8'));

  assert.match(stdout, /dry-run social preview written/);
  assert.equal(post.idempotency_key, 'td-blog:5bb87e716f8d6d69');
  assert.equal(post.network_publish, false);
});
