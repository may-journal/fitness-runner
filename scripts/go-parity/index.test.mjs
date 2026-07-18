import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { diffResults, discoverCheckNames, normalizeError } from './index.mjs';

test('normalizeError strips absolute repo paths', () => {
  assert.equal(normalizeError('/repo/a/b.md: bad', '/repo'), 'a/b.md: bad');
  assert.equal(normalizeError('plain message', '/repo'), 'plain message');
});

test('diffResults matches identical results', () => {
  const r = { errors: ['x'], filesChecked: 3, ok: false };
  assert.deepEqual(diffResults('changelog', r, r, '/repo'), []);
});

test('diffResults reports ok and filesChecked differences', () => {
  const ts = { errors: [], filesChecked: 3, ok: true };
  const go = { errors: [], filesChecked: 4, ok: false };
  const diffs = diffResults('changelog', ts, go, '/repo');
  assert.equal(diffs.length, 2);
});

test('diffResults exempts jscpd filesChecked but not its verdict', () => {
  const ts = { errors: [], filesChecked: 150, ok: true };
  const go = { errors: [], filesChecked: 190, ok: true };
  assert.deepEqual(diffResults('jscpd', ts, go, '/repo'), []);
  assert.equal(diffResults('jscpd', { ...ts, ok: false }, go, '/repo').length, 1);
});

test('discoverCheckNames derives the catalog from built binaries', () => {
  const dir = mkdtempSync(join(tmpdir(), 'parity-'));
  for (const f of ['fitness-check-changelog', 'fitness-check-cspell', 'fitness']) {
    writeFileSync(join(dir, f), '');
  }
  assert.deepEqual(discoverCheckNames(dir), ['changelog', 'cspell']);
});
