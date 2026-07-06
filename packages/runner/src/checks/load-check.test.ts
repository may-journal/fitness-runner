import { mkdtempSync, mkdirSync, writeFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';
import { fileURLToPath } from 'node:url';
import { describe, it, expect } from 'vitest';
import {
  checkPackageName,
  findInstallRoot,
  isPathLoadedCheck,
  isPathSpec,
  loadCheck,
  loadCheckFromPath,
  loadCheckFromPathOrThrow,
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

  it('resolveCheckNames resolves the repo-root .fitnessrc (bundle defaults + opt-in mermaid + dependency-currency)', async () => {
    const names = await resolveCheckNames(REPO_ROOT);
    expect(names).toContain('changelog'); // a bundle default check
    expect(names).toContain('jscpd'); // a bundle default (nothing disabled)
    expect(names).toContain('mermaid-callouts'); // opt-in mermaid check enabled in .fitnessrc.js
    expect(names).toContain('dependency-currency'); // opt-in check enabled in .fitnessrc.js
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

  it('isPathSpec distinguishes paths from names', () => {
    expect(isPathSpec('./my-check.js')).toBe(true);
    expect(isPathSpec('fitness/checks/my-check.mjs')).toBe(true);
    expect(isPathSpec('eslint')).toBe(false);
  });

  it('loadCheckFromPath loads a default-exported check and marks it path-loaded', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-path-default-'));
    writeFileSync(
      join(dir, 'my-check.js'),
      'export default { name: "my-check", run: async () => ({ ok: true, errors: [] }) };'
    );
    const check = await loadCheckFromPath(dir, './my-check.js');
    expect(check?.name).toBe('my-check');
    expect(isPathLoadedCheck(check!)).toBe(true);
  });

  it('loadCheckFromPath returns null when the path does not exist', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-path-missing-'));
    await expect(loadCheckFromPath(dir, './nope.js')).resolves.toBeNull();
  });

  it('loadCheckFromPathOrThrow throws with the path in the message when missing', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-path-throw-'));
    await expect(loadCheckFromPathOrThrow(dir, './nope.js')).rejects.toThrow(
      enUS.CheckPathInvalid.replace('{{path}}', './nope.js')
    );
  });

  it('loadCheckFromPathOrThrow resolves the check when valid', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-path-throw-ok-'));
    writeFileSync(
      join(dir, 'ok-check.js'),
      'export default { name: "ok-check", run: async () => ({ ok: true, errors: [] }) };'
    );
    await expect(loadCheckFromPathOrThrow(dir, './ok-check.js')).resolves.toMatchObject({
      name: 'ok-check',
    });
  });

  it('resolveCheckNames keeps path specs alongside names, in order', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-mixed-specs-'));
    writeFileSync(join(dir, 'package.json'), '{}');
    writeFileSync(
      join(dir, '.fitnessrc.ts'),
      'export default { checks: ["eslint", "./fitness/my-check.js", "prettier"] };'
    );
    await expect(resolveCheckNames(dir)).resolves.toEqual([
      'eslint',
      './fitness/my-check.js',
      'prettier',
    ]);
  });

  it('resolveCheckNames dedupes path specs by resolved absolute path', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-dedupe-path-'));
    writeFileSync(join(dir, 'package.json'), '{}');
    writeFileSync(
      join(dir, '.fitnessrc.ts'),
      'export default { checks: ["./fitness/my-check.js", "./fitness/../fitness/my-check.js"] };'
    );
    await expect(resolveCheckNames(dir)).resolves.toEqual(['./fitness/my-check.js']);
  });

  it('resolveCheckNames never removes path specs via disabledChecks', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-disabled-path-'));
    writeFileSync(join(dir, 'package.json'), '{}');
    writeFileSync(
      join(dir, '.fitnessrc.ts'),
      'export default { checks: ["eslint", "./fitness/my-check.js"], disabledChecks: ["./fitness/my-check.js"] };'
    );
    await expect(resolveCheckNames(dir)).resolves.toEqual(['eslint', './fitness/my-check.js']);
  });
});
