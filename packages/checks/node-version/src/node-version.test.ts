import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect } from 'vitest';
import nodeVersionCheck from './index.js';

describe('nodeVersionCheck', () => {
  it('passes when current Node meets .nvmrc', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'node-version-'));
    writeFileSync(join(dir, '.nvmrc'), '18');
    const result = await nodeVersionCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('passes when .nvmrc has v-prefix', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'node-version-'));
    writeFileSync(join(dir, '.nvmrc'), 'v20');
    const result = await nodeVersionCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('fails when .nvmrc is missing', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'node-version-'));
    const result = await nodeVersionCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toBe('missing .nvmrc');
  });

  it('fails when current Node is below .nvmrc', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'node-version-'));
    writeFileSync(join(dir, '.nvmrc'), '99');
    const result = await nodeVersionCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toContain('does not satisfy .nvmrc');
    expect(result.errors[0]).toContain('99');
  });
});
