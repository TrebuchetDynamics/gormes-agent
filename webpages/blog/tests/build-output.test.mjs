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
