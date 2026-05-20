import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import { runSetupNpmTrustedPublishers } from './index.mjs';

const setupDir = dirname(fileURLToPath(import.meta.url));
const expectedProvision = join(setupDir, '../provision-npm-packages/index.mjs');

describe('runSetupNpmTrustedPublishers', () => {
  it('forwards argv to provision script path correctly', () => {
    const calls = [];
    const execPath = '/usr/bin/node';

    const status = runSetupNpmTrustedPublishers(['--dry-run', 'foo'], {
      execPath,
      spawnSync: (node, args, opts) => {
        calls.push({ args, node, opts });
        return { status: 0 };
      },
    });

    assert.equal(status, 0);
    assert.equal(calls.length, 1);
    assert.equal(calls[0].node, execPath);
    assert.deepEqual(calls[0].args, [expectedProvision, '--dry-run', 'foo']);
    assert.deepEqual(calls[0].opts, { stdio: 'inherit' });
  });

  it('returns 1 when spawnSync result has no status', () => {
    const status = runSetupNpmTrustedPublishers([], {
      execPath: '/usr/bin/node',
      spawnSync: () => ({}),
    });

    assert.equal(status, 1);
  });
});
