import { mkdtempSync, mkdirSync, writeFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';
import { fileURLToPath } from 'node:url';
import { describe, it, expect } from 'vitest';
import {
  checkPackageName,
  findInstallRoot,
  loadCheck,
  resolveCheckNames,
  tryLoadCheck,
} from './load-check.js';
import { enUS } from '../runner/enUS.js';

const REPO_ROOT = resolve(fileURLToPath(new URL('../../../..', import.meta.url)));

describe('load-check', () => {
  it('checkPackageName returns bundled check subpath', () => {
    expect(checkPackageName('eslint')).toBe('@mayjournal/fitness-checks/checks/eslint');
  });

  it('findInstallRoot walks up from a nested directory', () => {
    const nested = join(REPO_ROOT, 'packages', 'runner', 'src', 'checks');
    const installRoot = findInstallRoot(nested);
    expect(nested.startsWith(installRoot)).toBe(true);
    createRequire(join(installRoot, 'package.json')).resolve(
      '@mayjournal/fitness-checks/defaultChecks'
    );
  });

  it('findInstallRoot returns startRoot when no install is found', () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-load-check-'));
    expect(findInstallRoot(dir)).toBe(dir);
  });

  it('resolveCheckNames returns bundle defaultChecks from repo root', async () => {
    const names = await resolveCheckNames(REPO_ROOT);
    expect(names).toContain('changelog');
    expect(names.length).toBe(12);
  });

  it('resolveCheckNames throws when bundle is unavailable', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-no-bundle-'));
    writeFileSync(join(dir, 'package.json'), '{}');
    await expect(resolveCheckNames(dir)).rejects.toThrow(enUS.NoChecksConfigured);
  });

  it('resolveCheckNames accepts default export from bundle subpath', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-default-export-'));
    const bundleDir = join(dir, 'node_modules', '@mayjournal', 'fitness-checks');
    mkdirSync(bundleDir, { recursive: true });
    writeFileSync(join(dir, 'package.json'), '{}');
    writeFileSync(
      join(bundleDir, 'package.json'),
      JSON.stringify({
        name: '@mayjournal/fitness-checks',
        type: 'module',
        exports: { './defaultChecks': './defaultChecks.js' },
      })
    );
    writeFileSync(join(bundleDir, 'defaultChecks.js'), 'export default ["changelog"];');
    await expect(resolveCheckNames(dir)).resolves.toEqual(['changelog']);
  });

  it('resolveCheckNames throws when defaultChecks export is empty', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-empty-bundle-'));
    const bundleDir = join(dir, 'node_modules', '@mayjournal', 'fitness-checks');
    mkdirSync(bundleDir, { recursive: true });
    writeFileSync(join(dir, 'package.json'), '{}');
    writeFileSync(
      join(bundleDir, 'package.json'),
      JSON.stringify({
        name: '@mayjournal/fitness-checks',
        type: 'module',
        exports: { './defaultChecks': './defaultChecks.js' },
      })
    );
    writeFileSync(join(bundleDir, 'defaultChecks.js'), 'export const defaultChecks = [];');
    await expect(resolveCheckNames(dir)).rejects.toThrow(enUS.NoChecksConfigured);
  });

  it('resolveCheckNames dedupes explicit checks list', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-dedupe-'));
    writeFileSync(join(dir, 'package.json'), '{}');
    writeFileSync(
      join(dir, '.fitnessrc.ts'),
      'export default { checks: ["eslint", "eslint", "prettier"] };'
    );
    await expect(resolveCheckNames(dir)).resolves.toEqual(['eslint', 'prettier']);
  });

  it('resolveCheckNames applies disabledChecks to explicit checks list', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-disabled-explicit-'));
    writeFileSync(join(dir, 'package.json'), '{}');
    writeFileSync(
      join(dir, '.fitnessrc.ts'),
      'export default { checks: ["eslint", "prettier"], disabledChecks: ["prettier"] };'
    );
    await expect(resolveCheckNames(dir)).resolves.toEqual(['eslint']);
  });

  it('resolveCheckNames applies disabledChecks to bundle defaultChecks', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-disabled-bundle-'));
    const bundleDir = join(dir, 'node_modules', '@mayjournal', 'fitness-checks');
    mkdirSync(bundleDir, { recursive: true });
    writeFileSync(join(dir, 'package.json'), '{}');
    writeFileSync(
      join(bundleDir, 'package.json'),
      JSON.stringify({
        name: '@mayjournal/fitness-checks',
        type: 'module',
        exports: { './defaultChecks': './defaultChecks.js' },
      })
    );
    writeFileSync(
      join(bundleDir, 'defaultChecks.js'),
      'export const defaultChecks = ["changelog", "eslint", "prettier"];'
    );
    writeFileSync(join(dir, '.fitnessrc.ts'), 'export default { disabledChecks: ["eslint"] };');
    await expect(resolveCheckNames(dir)).resolves.toEqual(['changelog', 'prettier']);
  });

  it('resolveCheckNames returns empty list when all checks are disabled', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-disabled-all-'));
    writeFileSync(join(dir, 'package.json'), '{}');
    writeFileSync(
      join(dir, '.fitnessrc.ts'),
      'export default { checks: ["changelog"], disabledChecks: ["changelog"] };'
    );
    await expect(resolveCheckNames(dir)).resolves.toEqual([]);
  });

  it('loadCheck loads a real check package', async () => {
    const check = await loadCheck('changelog', REPO_ROOT);
    expect(check.name).toBe('changelog');
    expect(typeof check.run).toBe('function');
  });

  it('loadCheck throws when package is missing', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-no-pkg-'));
    writeFileSync(join(dir, 'package.json'), '{}');
    await expect(loadCheck('changelog', dir)).rejects.toThrow(
      enUS.CheckPackageNotInstalled.replace(
        '{{pkg}}',
        '@mayjournal/fitness-checks/checks/changelog'
      )
    );
  });

  it('loadCheck throws when default export name mismatches', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-bad-export-'));
    const bundleDir = join(dir, 'node_modules', '@mayjournal', 'fitness-checks');
    mkdirSync(join(bundleDir, 'checks', 'fake'), { recursive: true });
    writeFileSync(join(dir, 'package.json'), '{}');
    writeFileSync(
      join(bundleDir, 'package.json'),
      JSON.stringify({
        name: '@mayjournal/fitness-checks',
        type: 'module',
        exports: { './checks/fake': './checks/fake/index.js' },
      })
    );
    writeFileSync(
      join(bundleDir, 'checks/fake/index.js'),
      'export default { name: "other", run: async () => ({ ok: true, errors: [] }) };'
    );
    await expect(loadCheck('fake', dir)).rejects.toThrow(
      enUS.InvalidCheckExport.replace('{{pkg}}', '@mayjournal/fitness-checks/checks/fake').replace(
        '{{name}}',
        'fake'
      )
    );
  });

  it('loadCheck throws when default export is not a check', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-bad-run-'));
    const bundleDir = join(dir, 'node_modules', '@mayjournal', 'fitness-checks');
    mkdirSync(join(bundleDir, 'checks', 'bad'), { recursive: true });
    writeFileSync(join(dir, 'package.json'), '{}');
    writeFileSync(
      join(bundleDir, 'package.json'),
      JSON.stringify({
        name: '@mayjournal/fitness-checks',
        type: 'module',
        exports: { './checks/bad': './checks/bad/index.js' },
      })
    );
    writeFileSync(join(bundleDir, 'checks/bad/index.js'), 'export default { name: "bad" };');
    await expect(loadCheck('bad', dir)).rejects.toThrow(/Invalid check export/);
  });

  it('tryLoadCheck returns null on failure', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-try-null-'));
    writeFileSync(join(dir, 'package.json'), '{}');
    await expect(tryLoadCheck('missing', dir)).resolves.toBeNull();
  });
});
