import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';

function readDist(path) {
  return readFileSync(new URL(`../dist/${path}`, import.meta.url), 'utf8');
}

test('blog build publishes homepage, about page, real post, and feed', () => {
  const home = readDist('index.html');
  const about = readDist('about/index.html');
  const post = readDist('posts/autonomous-hermes-porting-loop/index.html');
  const feed = readDist('feed.xml');

  assert.match(home, /TrebuchetDynamics Engineering/);
  assert.match(home, /Autonomous Hermes-porting loop/);
  assert.match(home, /href="\/feed\.xml"/);

  assert.match(about, /TrebuchetDynamics/);
  assert.match(about, /validation-gated agentic engineering/i);
  assert.match(about, /gormes-agent/);
  assert.match(about, /agentic-porting-kit/);

  assert.match(post, /How an autonomous loop ships Hermes-parity slices/);
  assert.match(post, /progress\.json/);
  assert.match(post, /go test \.\/\.\.\. -count=1/);

  assert.match(feed, /<rss version="2\.0"/);
  assert.match(feed, /<title>TrebuchetDynamics Engineering<\/title>/);
  assert.match(feed, /autonomous-hermes-porting-loop/);
});


test('autonomous Hermes-porting writeup has evidence-backed draft structure', () => {
  const post = readDist('posts/autonomous-hermes-porting-loop/index.html');
  const article = post.match(/<div class="article-content">([\s\S]*?)<\/div>/)?.[1] ?? '';
  const plain = article.replace(/<[^>]+>/g, ' ');

  assert.match(article, /Intent → Oracle → Surface → Work package → Proof/);
  assert.match(article, /Architecture diagram/);
  assert.match(article, /c2a267dad/);
  assert.match(article, /5cd4f870f/);
  assert.match(article, /2e3a56a21/);
  assert.match(article, /go run \.\/cmd\/progress validate/);
  assert.match(article, /git diff --check/);
  assert.match(article, /cost-per-feature/);
  assert.match(article, /operator review/);
  assert.ok(plain.split(/\s+/).filter(Boolean).length >= 1500, 'expected a full local draft, not a stub');
});

test('autonomous Hermes-porting review packet keeps publication blockers local', () => {
  const packet = readFileSync(
    new URL('../../docs/content/building-gormes/strategy/writeups/autonomous-hermes-porting-loop-review-packet.md', import.meta.url),
    'utf8',
  );

  assert.match(packet, /webpages\/blog\/src\/content\/posts\/autonomous-hermes-porting-loop\.md/);
  assert.match(packet, /https:\/\/engineering\.trebuchetdynamics\.com\/feed\.xml/);
  assert.match(packet, /node scripts\/blog-feed-social\/publish\.mjs --dry-run/);
  assert.match(packet, /2e3a56a21/);
  assert.match(packet, /5cd4f870f/);
  assert.match(packet, /c2a267dad/);
  assert.match(packet, /one week of measured cost telemetry/i);
  assert.match(packet, /publication date/i);
  assert.match(packet, /target platform/i);
  assert.match(packet, /no social tokens/i);
  assert.match(packet, /voice\/tone/i);
});

test('repository ships a markdown commit deploy pipeline for the blog', () => {
  const workflow = readFileSync(
    new URL('../../../.github/workflows/deploy-td-blog.yml', import.meta.url),
    'utf8',
  );

  assert.match(workflow, /Deploy TrebuchetDynamics Engineering Blog/);
  assert.match(workflow, /webpages\/blog/);
  assert.match(workflow, /npm ci/);
  assert.match(workflow, /npm run build/);
  assert.match(workflow, /pages deploy webpages\/blog\/dist/);
  assert.match(workflow, /engineering\.trebuchetdynamics\.com/);
});
