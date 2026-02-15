import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect } from 'vitest';
import { changelogCheck } from '../../src/checks/changelog.js';

describe('changelogCheck', () => {
  it('passes when CHANGELOG.md has at least one yyyy-mm-dd section', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(
      join(dir, 'CHANGELOG.md'),
      '# Changelog\n\n## Changes\n\n### 2026-02-15\n\n- item',
    );
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('passes when section uses ## yyyy-mm-dd@time format', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026-02-15@10AM\n\n- item');
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('fails when CHANGELOG.md exists but has no dated section', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n## Changes\n\nNo dates.');
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toContain('at least one dated section');
  });

  it('fails when CHANGELOG.md is missing at root', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toBe('missing root CHANGELOG.md');
    expect(result.meta?.filesChecked).toBe(1);
  });
});
