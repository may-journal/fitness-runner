import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect } from 'vitest';
import { changelogCheck } from './index.js';

describe('changelogCheck', () => {
  it('passes when CHANGELOG.md has ### yyyy-mm-dd@11am section', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026-02-15@11am\n\n- item');
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('fails when only ## yyyy-mm-dd@time (no ### heading)', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n## 2026-02-15@11am\n\n- item');
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toContain('at least one ### yyyy-mm-dd@time');
  });

  it('fails when a ### heading has space+time instead of @time', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026-02-15 14:00\n\n- item');
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toContain('invalid:');
  });

  it('fails when a ### heading has date but no hour', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026-02-15\n\n- item');
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toContain('invalid:');
  });

  it('fails when CHANGELOG.md has no ### heading', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\nNo dates.');
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toContain('at least one ### yyyy-mm-dd@time');
  });

  it('passes when all ### headings match ### yyyy-mm-dd@time', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(
      join(dir, 'CHANGELOG.md'),
      '# Changelog\n\n### 2026-02-15@11am\n\n- a\n\n### 2026-02-14@9pm\n\n- b',
    );
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('fails when one ### heading does not match format', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(
      join(dir, 'CHANGELOG.md'),
      '# Changelog\n\n### 2026-02-15@11am\n\n- a\n\n### 2026-02-15\n\n- b',
    );
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toHaveLength(1);
    expect(result.errors![0]).toContain('invalid:');
    expect(result.errors![0]).toContain('### 2026-02-15');
  });

  it('fails when CHANGELOG.md is missing at root', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toBe('missing root CHANGELOG.md');
    expect(result.meta?.filesChecked).toBe(1);
  });
});
