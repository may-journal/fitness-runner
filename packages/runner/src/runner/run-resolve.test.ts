import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect, vi, afterEach } from 'vitest';

describe('getChecks', () => {
  afterEach(() => {
    vi.doUnmock('../checks/load-check.js');
    vi.resetModules();
  });

  it('returns resolutionError when resolveCheckNames throws a non-Error', async () => {
    vi.doMock('../checks/load-check.js', async (importOriginal) => {
      const mod = await importOriginal<typeof import('../checks/load-check.js')>();
      return {
        ...mod,
        resolveCheckNames: async () => {
          throw 'resolve failed';
        },
      };
    });
    const { getChecks } = await import('./run-resolve.js');
    const result = await getChecks(['node', 'fitness'], process.cwd());
    expect(result.checks).toEqual([]);
    expect(result.resolutionError).toBe('resolve failed');
  });

  it('returns resolutionError when resolveCheckNames throws an Error', async () => {
    vi.doMock('../checks/load-check.js', async (importOriginal) => {
      const mod = await importOriginal<typeof import('../checks/load-check.js')>();
      return {
        ...mod,
        resolveCheckNames: async () => {
          throw new Error('resolve boom');
        },
      };
    });
    const { getChecks } = await import('./run-resolve.js');
    const result = await getChecks(['node', 'fitness'], process.cwd());
    expect(result.resolutionError).toBe('resolve boom');
  });

  it('loads checks via loadCheck when resolving bundle defaults', async () => {
    const loadCheckMock = vi.fn(async (name: string) => ({
      name,
      run: async () => ({ ok: true, errors: [], meta: {} }),
    }));
    vi.doMock('../checks/load-check.js', async (importOriginal) => {
      const mod = await importOriginal<typeof import('../checks/load-check.js')>();
      return {
        ...mod,
        resolveCheckNames: async () => ['eslint'],
        loadCheck: loadCheckMock,
      };
    });
    const { getChecks } = await import('./run-resolve.js');
    const dir = mkdtempSync(join(tmpdir(), 'fitness-resolve-bundle-'));
    writeFileSync(join(dir, 'package.json'), '{}');
    const result = await getChecks(['node', 'fitness'], dir);
    expect(loadCheckMock).toHaveBeenCalledWith('eslint', dir);
    expect(result.checks.map((c) => c.name)).toEqual(['eslint']);
  });

  it('returns no checks when resolved names list is empty', async () => {
    vi.doMock('../checks/load-check.js', async (importOriginal) => {
      const mod = await importOriginal<typeof import('../checks/load-check.js')>();
      return { ...mod, resolveCheckNames: async () => [] };
    });
    const { getChecks } = await import('./run-resolve.js');
    const result = await getChecks(['node', 'fitness'], process.cwd());
    expect(result.checks).toEqual([]);
  });

  it('skips config checks that fail to load when allowMissing', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'fitness-resolve-partial-'));
    writeFileSync(join(dir, 'package.json'), '{}');
    writeFileSync(
      join(dir, '.fitnessrc.ts'),
      'export default { checks: ["eslint", "not-installed"] };'
    );
    vi.doMock('../checks/load-check.js', async (importOriginal) => {
      const mod = await importOriginal<typeof import('../checks/load-check.js')>();
      return {
        ...mod,
        tryLoadCheck: async (name: string) =>
          name === 'eslint'
            ? { name: 'eslint', run: async () => ({ ok: true, errors: [], meta: {} }) }
            : null,
      };
    });
    const { getChecks } = await import('./run-resolve.js');
    const result = await getChecks(['node', 'fitness'], dir);
    expect(result.checks.map((c) => c.name)).toEqual(['eslint']);
  });
});
