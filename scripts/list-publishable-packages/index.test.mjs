import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import { listPublishablePackages } from './index.mjs';

const repoRoot = join(fileURLToPath(new URL('.', import.meta.url)), '../..');

describe('listPublishablePackages', () => {
  it('returns 15 @mayjournal packages', () => {
    const packages = listPublishablePackages();

    assert.equal(packages.length, 15);
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

  it('sorts shared first, check packages next, bundle, then fitness last', () => {
    const packages = listPublishablePackages();
    const names = packages.map(({ name }) => name);

    assert.equal(names[0], '@mayjournal/fitness-shared');
    assert.equal(names.at(-2), '@mayjournal/fitness-checks');
    assert.equal(names.at(-1), '@mayjournal/fitness');

    const checkNames = names.slice(1, -2);
    assert.ok(checkNames.every((name) => name.startsWith('@mayjournal/fitness-check-')));
    assert.deepEqual(checkNames, [...checkNames].sort());
  });

  it('skips private packages', () => {
    const packages = listPublishablePackages();
    const names = new Set(packages.map(({ name }) => name));

    assert.ok(!names.has('fitness-runner'));

    for (const { dir } of packages) {
      const pkg = JSON.parse(readFileSync(join(repoRoot, dir, 'package.json'), 'utf8'));
      assert.notEqual(pkg.private, true);
    }
  });
});
