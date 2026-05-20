import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import { parsePackDryRun, summarizePublint } from './index.mjs';
import { checkRunnerTarballGate, formatBytes } from './format-report.mjs';

describe('formatBytes', () => {
  it('formats B, KiB, and MiB', () => {
    assert.equal(formatBytes(512), '512 B');
    assert.equal(formatBytes(2048), '2.0 KiB');
    assert.equal(formatBytes(2 * 1024 * 1024), '2.00 MiB');
  });
});

describe('parsePackDryRun', () => {
  it('reads tarball and unpacked sizes from npm pack JSON array', () => {
    const json = JSON.stringify([{ size: 1000, unpackedSize: 5000 }]);
    assert.deepEqual(parsePackDryRun(json), { tarball: 1000, unpacked: 5000 });
  });

  it('throws on unexpected output', () => {
    assert.throws(() => parsePackDryRun('[]'), /Unexpected npm pack/);
  });
});

describe('checkRunnerTarballGate', () => {
  it('passes when runner tarball is under limit', () => {
    const rows = [{ name: '@mayjournal/fitness', sizes: { tarball: 20_000, unpacked: 70_000 } }];
    assert.equal(checkRunnerTarballGate(rows, { runnerMaxTarball: 24 * 1024 }).ok, true);
  });

  it('fails when runner tarball exceeds limit', () => {
    const rows = [{ name: '@mayjournal/fitness', sizes: { tarball: 30_000, unpacked: 70_000 } }];
    const gate = checkRunnerTarballGate(rows, { runnerMaxTarball: 24 * 1024 });
    assert.equal(gate.ok, false);
    assert.match(gate.message ?? '', /exceeds limit/);
  });
});

describe('summarizePublint', () => {
  it('marks success when exit code is 0', () => {
    assert.deepEqual(summarizePublint({ status: 0, stdout: 'Linting...\nAll good!' }), {
      ok: true,
      errors: 0,
      warnings: 0,
      output: 'Linting...\nAll good!',
    });
  });

  it('counts numbered errors when exit code is non-zero', () => {
    const stdout = `Errors:
1. pkg.exports types order
2. missing files field`;
    const summary = summarizePublint({ status: 1, stdout });
    assert.equal(summary.ok, false);
    assert.ok(summary.errors >= 1);
  });
});
