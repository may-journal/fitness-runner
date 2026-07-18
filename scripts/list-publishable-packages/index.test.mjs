import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import { listPublishablePackages } from './index.mjs';

const repoRoot = join(fileURLToPath(new URL('.', import.meta.url)), '../..');

describe('listPublishablePackages', () => {
  it('returns the one publishable package (shared configs)', () => {
    const packages = listPublishablePackages();

    assert.equal(packages.length, 1);
    assert.ok(packages.every(({ name }) => name.startsWith('@mayjournal/')));
  });

  it('returns entries with { dir, name } shape', () => {
    const packages = listPublishablePackages();

    for (const entry of packages) {
      assert.deepEqual(Object.keys(entry).sort(), ['dir', 'name']);
      assert.equal(typeof entry.dir, 'string');
      assert.equal(typeof entry.name, 'string');
      assert.match(entry.dir, /^packages\//);
    }
  });

  it('lists the shared configs package', () => {
    const packages = listPublishablePackages();
    const names = packages.map(({ name }) => name);

    assert.deepEqual(names, ['@mayjournal/fitness-shared']);
  });

  it('skips private packages', () => {
    const packages = listPublishablePackages();
    const names = new Set(packages.map(({ name }) => name));

    assert.ok(!names.has('fitness-runner'));
    assert.ok(!names.has('@mayjournal/fitness-check-prettier'));

    for (const { dir } of packages) {
      const pkg = JSON.parse(readFileSync(join(repoRoot, dir, 'package.json'), 'utf8'));
      assert.notEqual(pkg.private, true);
    }
  });
});
