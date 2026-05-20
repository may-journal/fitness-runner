import { mkdtempSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { stdin as input, stdout as output } from 'node:process';
import assert from 'node:assert/strict';
import { describe, it, mock, beforeEach, afterEach } from 'node:test';
import {
  configureTrustWithPermissions,
  needsWebSignIn,
  isWebOtpChallenge,
  loadAuthFromNpmrc,
  otplease,
  trustPostBody,
} from './trust-github-api.mjs';

describe('trustPostBody', () => {
  it('includes permissions createPackage for registry trust API', () => {
    const body = trustPostBody('may-journal/fitness-runner', 'publish.yml');

    assert.equal(body.length, 1);
    assert.equal(body[0].type, 'github');
    assert.deepEqual(body[0].permissions, ['createPackage']);
    assert.equal(body[0].claims.repository, 'may-journal/fitness-runner');
    assert.deepEqual(body[0].claims.workflow_ref, { file: 'publish.yml' });
  });
});

describe('loadAuthFromNpmrc', () => {
  let tmpDir;

  afterEach(() => {
    if (tmpDir) {
      rmSync(tmpDir, { force: true, recursive: true });
      tmpDir = undefined;
    }
    delete process.env.NODE_AUTH_TOKEN;
  });

  it('reads _authToken from the first matching path', () => {
    tmpDir = mkdtempSync(join(tmpdir(), 'provision-trust-auth-'));
    const npmrc = join(tmpDir, '.npmrc');
    writeFileSync(npmrc, '//registry.npmjs.org/:_authToken=npm_test_token_abc\n', 'utf8');

    assert.equal(loadAuthFromNpmrc([npmrc, join(tmpDir, 'missing')]), 'npm_test_token_abc');
  });

  it('prefers home npmrc before repo when both exist', () => {
    tmpDir = mkdtempSync(join(tmpdir(), 'provision-trust-auth-'));
    const home = join(tmpDir, 'home.npmrc');
    const repo = join(tmpDir, 'repo.npmrc');
    writeFileSync(home, '//registry.npmjs.org/:_authToken=from-home\n', 'utf8');
    writeFileSync(repo, '//registry.npmjs.org/:_authToken=from-repo\n', 'utf8');

    assert.equal(loadAuthFromNpmrc([home, repo]), 'from-home');
  });

  it('falls back to NODE_AUTH_TOKEN when no npmrc token', () => {
    tmpDir = mkdtempSync(join(tmpdir(), 'provision-trust-auth-'));
    process.env.NODE_AUTH_TOKEN = 'env-token';

    assert.equal(loadAuthFromNpmrc([join(tmpDir, 'missing')]), 'env-token');
  });
});

describe('isWebOtpChallenge', () => {
  it('detects authUrl and doneUrl on E401 trust responses', () => {
    const err = {
      body: { authUrl: 'https://auth.example', doneUrl: 'https://done.example' },
      code: 'E401',
      statusCode: 401,
    };

    assert.equal(isWebOtpChallenge(err), true);
    assert.equal(needsWebSignIn(err), true);
  });
});

describe('needsWebSignIn', () => {
  it('detects bare EOTP without web URLs (registry www-authenticate: otp)', () => {
    const err = { body: { error: 'OTP required for authentication' }, code: 'EOTP' };
    assert.equal(isWebOtpChallenge(err), false);
    assert.equal(needsWebSignIn(err), true);
  });

  it('detects plain 401 on trust POST (E401 without OTP body)', () => {
    const err = {
      code: 'E401',
      message:
        '401 Unauthorized - POST https://registry.npmjs.org/-/package/@mayjournal%2ffitness-shared/trust',
      statusCode: 401,
    };
    assert.equal(needsWebSignIn(err), true);
  });
});

describe('otplease', () => {
  const ttySnapshot = { stdin: input.isTTY, stdout: output.isTTY };

  beforeEach(() => {
    Object.defineProperty(input, 'isTTY', { configurable: true, value: true });
    Object.defineProperty(output, 'isTTY', { configurable: true, value: true });
  });

  afterEach(() => {
    Object.defineProperty(input, 'isTTY', { configurable: true, value: ttySnapshot.stdin });
    Object.defineProperty(output, 'isTTY', { configurable: true, value: ttySnapshot.stdout });
  });

  it('retries fn with otp after E401 web-auth challenge', async () => {
    let attempts = 0;
    const fn = mock.fn(async (opts) => {
      attempts += 1;
      if (attempts === 1) {
        const err = new Error('401 Unauthorized');
        err.code = 'E401';
        err.statusCode = 401;
        err.body = { authUrl: 'https://auth.example', doneUrl: 'https://done.example' };
        err.uri = 'https://registry.npmjs.org/-/package/@scope%2fpkg/trust';
        throw err;
      }
      assert.equal(opts.otp, 'web-otp-token');
      return 'ok';
    });

    const depsLoader = () => ({
      webAuthOpener: mock.fn(async () => ({ token: 'web-otp-token' })),
    });

    const result = await otplease({ registry: 'https://registry.npmjs.org/' }, fn, depsLoader);

    assert.equal(result, 'ok');
    assert.equal(attempts, 2);
  });

  it('retries fn with otp after EOTP and webAuthOpener', async () => {
    let attempts = 0;
    const fn = mock.fn(async (opts) => {
      attempts += 1;
      if (attempts === 1) {
        const err = new Error('EOTP required');
        err.code = 'EOTP';
        err.body = { authUrl: 'https://auth.example', doneUrl: 'https://done.example' };
        err.uri = 'https://registry.npmjs.org/-/package/@scope%2fpkg/trust';
        throw err;
      }
      assert.equal(opts.otp, '999999');
      return 'ok';
    });

    const depsLoader = () => ({
      webAuthOpener: mock.fn(async () => ({ token: '999999' })),
    });

    const result = await otplease({ registry: 'https://registry.npmjs.org/' }, fn, depsLoader);

    assert.equal(result, 'ok');
    assert.equal(attempts, 2);
    assert.equal(fn.mock.callCount(), 2);
  });

  it('refreshes Bearer token via loginWeb then retries (not npm-otp)', async () => {
    let attempts = 0;
    const fn = mock.fn(async (opts) => {
      attempts += 1;
      if (attempts === 1) {
        const err = new Error('401 Unauthorized');
        err.code = 'E401';
        err.statusCode = 401;
        err.message =
          '401 Unauthorized - POST https://registry.npmjs.org/-/package/@scope%2fpkg/trust';
        throw err;
      }
      assert.equal(opts.token, 'session-from-login-web');
      assert.equal(opts.otp, undefined);
      return 'ok';
    });

    const depsLoader = () => ({
      loginWeb: mock.fn(async () => ({ token: 'session-from-login-web' })),
      webAuthOpener: mock.fn(),
    });

    const result = await otplease(
      { registry: 'https://registry.npmjs.org/', token: 'stale-token' },
      fn,
      depsLoader
    );

    assert.equal(result, 'ok');
    assert.equal(attempts, 2);
  });
});

describe('configureTrustWithPermissions', () => {
  const savedToken = process.env.NODE_AUTH_TOKEN;

  afterEach(() => {
    if (savedToken === undefined) delete process.env.NODE_AUTH_TOKEN;
    else process.env.NODE_AUTH_TOKEN = savedToken;
  });

  it('POSTs trust body with createPackage via mocked npmFetch', async () => {
    process.env.NODE_AUTH_TOKEN = 'test-token';
    const fetchCalls = [];
    const npmFetch = mock.fn(async (uri, opts) => {
      fetchCalls.push({
        authType: opts.authType,
        body: opts.body,
        method: opts.method,
        npmCommand: opts.npmCommand,
        otp: opts.otp,
        uri,
      });
    });
    const npa = (name) => ({ escapedName: name.replace(/^@mayjournal\//, 'mayjournal%2f') });

    await configureTrustWithPermissions(
      {
        packageName: '@mayjournal/fitness-shared',
        repo: 'may-journal/fitness-runner',
        workflow: 'publish.yml',
      },
      { npa, npmFetch }
    );

    assert.equal(fetchCalls.length, 1);
    assert.match(fetchCalls[0].uri, /trust$/);
    assert.equal(fetchCalls[0].method, 'POST');
    assert.deepEqual(
      fetchCalls[0].body,
      trustPostBody('may-journal/fitness-runner', 'publish.yml')
    );
    assert.equal(fetchCalls[0].otp, undefined);
    assert.equal(fetchCalls[0].authType, 'web');
    assert.equal(fetchCalls[0].npmCommand, 'trust');
  });
});
