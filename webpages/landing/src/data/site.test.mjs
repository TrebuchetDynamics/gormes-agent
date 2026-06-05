import assert from 'node:assert/strict';
import { test } from 'node:test';

import { absoluteDocsUrl, absoluteSiteUrl, site } from './site.js';

test('landing site metadata owns canonical public URLs', () => {
  assert.equal(site.name, 'Gormes');
  assert.equal(site.url, 'https://gormes.ai/');
  assert.equal(site.githubUrl, 'https://github.com/TrebuchetDynamics/gormes-agent');
  assert.equal(site.installScriptUrl, 'https://gormes.ai/install.sh');
  assert.equal(site.socialImage, 'https://gormes.ai/static/social-card.png');

  assert.equal(absoluteSiteUrl('/'), 'https://gormes.ai/');
  assert.equal(absoluteSiteUrl('/built-with'), 'https://gormes.ai/built-with');
  assert.equal(
    absoluteDocsUrl('/why-gormes/#public-comparison-matrix'),
    'https://docs.gormes.ai/why-gormes/#public-comparison-matrix',
  );
});
