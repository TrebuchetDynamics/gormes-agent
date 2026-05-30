import assert from 'node:assert/strict';
import { delimiter, join } from 'node:path';
import { test } from 'node:test';

import {
  findCompatibleNodeBin,
  nodeAtLeast,
  resolveExecutionEnv,
} from './compatible-node.mjs';

function fakeFs(dirsWithNode) {
  return {
    readdirSync(root, options) {
      assert.equal(options.withFileTypes, true);
      return dirsWithNode.map((dir) => ({
        name: dir,
        isDirectory: () => true,
      }));
    },
    existsSync(candidate) {
      return dirsWithNode.some(
        (dir) => candidate === join('/home/gormes', '.nvm', 'versions', 'node', dir, 'bin', 'node'),
      );
    },
  };
}

test('nodeAtLeast accepts the required Node floor and newer majors', () => {
  assert.equal(nodeAtLeast('v22.11.0', 22, 12), false);
  assert.equal(nodeAtLeast('v22.12.0', 22, 12), true);
  assert.equal(nodeAtLeast('23.0.0', 22, 12), true);
});

test('findCompatibleNodeBin chooses the newest compatible nvm install', () => {
  const fs = fakeFs(['v20.19.0', 'v22.12.0', 'v22.21.1', 'v23.0.0']);

  assert.equal(
    findCompatibleNodeBin({ home: '/home/gormes', fs }),
    join('/home/gormes', '.nvm', 'versions', 'node', 'v23.0.0', 'bin'),
  );
});

test('resolveExecutionEnv prepends compatible Node only below the floor', () => {
  const fs = fakeFs(['v22.21.1']);
  const env = { PATH: '/usr/bin', GORMES_EXAMPLE: '1' };

  const resolved = resolveExecutionEnv({
    currentVersion: 'v20.11.1',
    env,
    home: '/home/gormes',
    fs,
  });

  assert.equal(
    resolved.PATH,
    `${join('/home/gormes', '.nvm', 'versions', 'node', 'v22.21.1', 'bin')}${delimiter}${env.PATH}`,
  );
  assert.equal(resolved.GORMES_EXAMPLE, '1');
  assert.equal(env.PATH, '/usr/bin', 'input env must not be mutated');
});

test('resolveExecutionEnv leaves PATH unchanged when current Node is compatible or no nvm match exists', () => {
  assert.deepEqual(
    resolveExecutionEnv({
      currentVersion: 'v22.21.1',
      env: { PATH: '/usr/bin' },
      home: '/home/gormes',
      fs: fakeFs(['v23.0.0']),
    }),
    { PATH: '/usr/bin' },
  );

  assert.deepEqual(
    resolveExecutionEnv({
      currentVersion: 'v20.11.1',
      env: { PATH: '/usr/bin' },
      home: '/home/gormes',
      fs: fakeFs(['v20.19.0']),
    }),
    { PATH: '/usr/bin' },
  );
});
