import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { createRequire } from 'node:module';
import { describe, test } from 'node:test';

const require = createRequire(import.meta.url);
const {
  bumpPackageVersion,
  collectPackageJsonPaths,
  formatTimestamp,
  isChangelogStaged,
  replaceChangelogTimestamp,
} = require('./index.cjs');

describe('formatTimestamp', () => {
  test('formats date as YYYY.MM.DD.HHMM', () => {
    assert.equal(formatTimestamp(new Date(2026, 4, 20, 15, 44)), '2026.05.20.1544');
  });

  test('zero-pads month, day, hour, and minute', () => {
    assert.equal(formatTimestamp(new Date(2026, 0, 5, 9, 3)), '2026.01.05.0903');
  });
});

describe('replaceChangelogTimestamp', () => {
  test('replaces only the first section heading timestamp', () => {
    const content = `# Changelog

## Changes

### 2026.05.20.1544

- First entry

### 2026.05.19.1200

- Older entry
`;
    const updated = replaceChangelogTimestamp(content, '2026.05.20.1600');
    assert.match(updated, /^### 2026\.05\.20\.1600/m);
    assert.match(updated, /### 2026\.05\.19\.1200/);
  });

  test('leaves content unchanged when no heading matches', () => {
    const content = '# Changelog\n\nNo timestamp heading here.\n';
    assert.equal(replaceChangelogTimestamp(content, '2026.05.20.1600'), content);
  });
});

describe('bumpPackageVersion', () => {
  test('replaces timestamp suffix on semver version', () => {
    assert.equal(
      bumpPackageVersion('0.1.0-2026.05.20.1544', '2026.05.20.1600'),
      '0.1.0-2026.05.20.1600'
    );
  });

  test('leaves version unchanged when no timestamp suffix is present', () => {
    assert.equal(bumpPackageVersion('1.2.3', '2026.05.20.1600'), '1.2.3');
  });
});

describe('isChangelogStaged', () => {
  test('returns true when CHANGELOG.md is staged', () => {
    assert.equal(isChangelogStaged(['package.json', 'CHANGELOG.md']), true);
  });

  test('returns false when CHANGELOG.md is not staged', () => {
    assert.equal(isChangelogStaged(['package.json', 'README.md']), false);
  });
});

describe('collectPackageJsonPaths', () => {
  /** @type {string | undefined} */
  let tempDir;

  test.afterEach(() => {
    if (tempDir) {
      rmSync(tempDir, { force: true, recursive: true });
      tempDir = undefined;
    }
  });

  test('finds package.json files recursively', () => {
    tempDir = mkdtempSync(join(tmpdir(), 'ensure-changelog-timestamp-'));
    writeFileSync(join(tempDir, 'package.json'), '{}');
    mkdirSync(join(tempDir, 'packages', 'a'), { recursive: true });
    mkdirSync(join(tempDir, 'packages', 'b', 'nested'), { recursive: true });
    writeFileSync(join(tempDir, 'packages', 'a', 'package.json'), '{}');
    writeFileSync(join(tempDir, 'packages', 'b', 'nested', 'package.json'), '{}');
    writeFileSync(join(tempDir, 'packages', 'b', 'readme.txt'), 'not a package');

    const paths = collectPackageJsonPaths(tempDir).sort();
    assert.deepEqual(paths, [
      join(tempDir, 'package.json'),
      join(tempDir, 'packages', 'a', 'package.json'),
      join(tempDir, 'packages', 'b', 'nested', 'package.json'),
    ]);
  });
});
