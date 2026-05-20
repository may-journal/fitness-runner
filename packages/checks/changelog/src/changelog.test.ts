import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect } from 'vitest';
import {
  changelogCheck,
  getPackageChangelogMismatch,
  MSG_VERSION_CHANGELOG,
  MSG_VERSION_LOCK,
} from './index.ts';

describe('changelogCheck', () => {
  it('passes when CHANGELOG.md has ### yyyy.mm.dd.HHMM section', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026.02.15.1100\n\n- item');
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('fails when only ## yyyy.mm.dd.HHMM (no ### heading)', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n## 2026.02.15.1100\n\n- item');
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toContain('at least one ### yyyy.mm.dd.HHMM');
  });

  it('fails when a ### heading uses old @time format', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026-02-15@11am\n\n- item');
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toContain('invalid:');
  });

  it('fails when a ### heading has date but no time', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026.02.15\n\n- item');
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toContain('invalid:');
  });

  it('fails when CHANGELOG.md has no ### heading', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\nNo dates.');
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toContain('at least one ### yyyy.mm.dd.HHMM');
  });

  it('passes when all ### headings match ### yyyy.mm.dd.HHMM', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(
      join(dir, 'CHANGELOG.md'),
      '# Changelog\n\n### 2026.02.15.1100\n\n- a\n\n### 2026.02.14.2100\n\n- b'
    );
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('fails when one ### heading does not match format', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(
      join(dir, 'CHANGELOG.md'),
      '# Changelog\n\n### 2026.02.15.1100\n\n- a\n\n### 2026.02.15\n\n- b'
    );
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toHaveLength(1);
    expect(result.errors![0]).toContain('invalid:');
    expect(result.errors![0]).toContain('### 2026.02.15');
  });

  it('fails when CHANGELOG.md is missing at root', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toBe('missing root CHANGELOG.md');
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('passes when package.json and package-lock version match first CHANGELOG heading', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026.02.15.1100\n\n- item');
    writeFileSync(join(dir, 'package.json'), '{"version":"0.1.0-2026.02.15.1100"}');
    writeFileSync(
      join(dir, 'package-lock.json'),
      '{"version":"0.1.0-2026.02.15.1100","packages":{"":{}}}'
    );
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('getPackageChangelogMismatch returns null when version suffix matches changelog timestamp', () => {
    expect(getPackageChangelogMismatch('0.1.0-2026.02.15.1100', '2026.02.15.1100')).toBe(null);
  });

  it('getPackageChangelogMismatch returns error when pkgVersion is undefined', () => {
    expect(getPackageChangelogMismatch(undefined, '2026.02.15.1100')).toBe(MSG_VERSION_CHANGELOG);
  });

  it('passes when package.json version matches first CHANGELOG heading and no package-lock', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026.02.15.1100\n\n- item');
    writeFileSync(join(dir, 'package.json'), '{"version":"0.1.0-2026.02.15.1100"}');
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('fails when package.json version suffix does not match first CHANGELOG heading', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026.02.15.1100\n\n- item');
    writeFileSync(join(dir, 'package.json'), '{"version":"0.1.0-2026.02.15.1200"}');
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toContain(MSG_VERSION_CHANGELOG);
  });

  it('fails when package.json version has no yyyy.mm.dd.HHMM suffix', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026.02.15.1100\n\n- item');
    writeFileSync(join(dir, 'package.json'), '{"version":"1.0.0"}');
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toContain(MSG_VERSION_CHANGELOG);
  });

  it('fails when package-lock version does not match package.json version', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026.02.15.1100\n\n- item');
    writeFileSync(join(dir, 'package.json'), '{"version":"0.1.0-2026.02.15.1100"}');
    writeFileSync(
      join(dir, 'package-lock.json'),
      '{"version":"0.1.0-2026.02.15.1200","packages":{"":{}}}'
    );
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toContain(MSG_VERSION_LOCK);
  });

  it('fails when package.json is invalid JSON', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026.02.15.1100\n\n- item');
    writeFileSync(join(dir, 'package.json'), 'not json');
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toContain('package.json is invalid JSON');
  });

  it('fails when package-lock.json is invalid JSON', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'changelog-'));
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026.02.15.1100\n\n- item');
    writeFileSync(join(dir, 'package.json'), '{"version":"0.1.0-2026.02.15.1100"}');
    writeFileSync(join(dir, 'package-lock.json'), 'not json');
    const result = await changelogCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toContain('package-lock.json is invalid JSON');
  });
});
