import assert from 'node:assert/strict';
import { describe, it, mock, beforeEach, afterEach } from 'node:test';
import { configureTrust, parseMode, testHooks } from './index.mjs';

let trustListResult = () => ({ status: 1, stderr: '', stdout: '' });
/** @type {Error | null} */
let nextTrustApiError = null;
let trustApiCalls = 0;

function spawnSyncImpl(_cmd, args) {
  if (args?.[0] === 'trust' && args?.[1] === 'list') return trustListResult();
  return { status: 0, stderr: '', stdout: '11.14.1' };
}

const trustMock = async () => {
  trustApiCalls += 1;
  if (nextTrustApiError) {
    const err = nextTrustApiError;
    nextTrustApiError = null;
    throw err;
  }
};

describe('parseMode', () => {
  it('defaults to all', () => {
    assert.equal(parseMode(), 'all');
    assert.equal(parseMode(undefined), 'all');
  });

  it('accepts modes with or without -- prefix', () => {
    assert.equal(parseMode('seed'), 'seed');
    assert.equal(parseMode('--trust'), 'trust');
    assert.equal(parseMode('check'), 'check');
  });

  it('exits with code 2 for invalid mode', () => {
    mock.method(process, 'exit', (code) => {
      throw new Error(`exit:${code}`);
    });
    try {
      assert.throws(() => parseMode('bogus'), /exit:2/);
    } finally {
      mock.restoreAll();
    }
  });
});

describe('configureTrust', () => {
  const envSnapshot = {
    DRY_RUN: process.env.DRY_RUN,
    NODE_AUTH_TOKEN: process.env.NODE_AUTH_TOKEN,
  };

  beforeEach(() => {
    trustApiCalls = 0;
    nextTrustApiError = null;
    trustListResult = () => ({ status: 1, stderr: '', stdout: '' });
    testHooks.spawnSync = spawnSyncImpl;
    testHooks.configureTrustWithPermissions = trustMock;
    delete process.env.DRY_RUN;
    delete process.env.NODE_AUTH_TOKEN;
  });

  afterEach(() => {
    testHooks.spawnSync = undefined;
    testHooks.configureTrustWithPermissions = undefined;
    if (envSnapshot.DRY_RUN === undefined) delete process.env.DRY_RUN;
    else process.env.DRY_RUN = envSnapshot.DRY_RUN;
    if (envSnapshot.NODE_AUTH_TOKEN === undefined) delete process.env.NODE_AUTH_TOKEN;
    else process.env.NODE_AUTH_TOKEN = envSnapshot.NODE_AUTH_TOKEN;
  });

  it('dry-run skips trust API call', async () => {
    process.env.DRY_RUN = '1';

    assert.equal(await configureTrust('@mayjournal/fitness-shared'), true);
    assert.equal(trustApiCalls, 0);
  });

  it('skips trust setup when NODE_AUTH_TOKEN is set (CI token)', async () => {
    process.env.NODE_AUTH_TOKEN = 'granular-token';

    assert.equal(await configureTrust('@mayjournal/fitness-shared'), false);
    assert.equal(trustApiCalls, 0);
  });

  it('treats HTTP 409 from trust API as already configured', async () => {
    nextTrustApiError = Object.assign(new Error('Conflict'), { statusCode: 409 });

    assert.equal(await configureTrust('@mayjournal/fitness-shared'), true);
    assert.equal(trustApiCalls, 1);
  });

  it('returns true when npm trust list already has entries', async () => {
    trustListResult = () => ({ status: 0, stderr: '', stdout: '[{"type":"github"}]' });

    assert.equal(await configureTrust('@mayjournal/fitness-shared'), true);
    assert.equal(trustApiCalls, 0);
  });
});
