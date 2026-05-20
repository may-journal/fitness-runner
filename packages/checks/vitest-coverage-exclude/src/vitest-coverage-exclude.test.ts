import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect } from 'vitest';
import { getCoverageExcludeFromConfig, loadVitestConfig } from '@mayjournal/fitness-shared';
import {
  ALLOWED_COVERAGE_EXCLUDE_PATTERNS,
  ALLOWED_SUFFIXES,
  configFromMod,
  getCoverageExclude,
  VITEST_CONFIG_NAMES,
  vitestCoverageExcludeCheck,
} from './index.js';

describe('vitestCoverageExcludeCheck', () => {
  it('passes when no Vitest config exists', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    const result = await vitestCoverageExcludeCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('passes when coverage.exclude is empty or missing', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    writeFileSync(
      join(dir, 'vitest.config.js'),
      'module.exports = { test: { coverage: { exclude: [] } } };'
    );
    const result = await vitestCoverageExcludeCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('passes when exclude has no .ts patterns', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    writeFileSync(
      join(dir, 'vitest.config.js'),
      'module.exports = { test: { coverage: { exclude: ["node_modules", "dist"] } } };'
    );
    const result = await vitestCoverageExcludeCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('passes when exclude uses only conventional patterns', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    writeFileSync(
      join(dir, 'vitest.config.js'),
      `module.exports = { test: { coverage: { exclude: ${JSON.stringify([...ALLOWED_COVERAGE_EXCLUDE_PATTERNS])} } } };`
    );
    const result = await vitestCoverageExcludeCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('passes when exclude uses src-prefixed conventional patterns', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    writeFileSync(
      join(dir, 'vitest.config.js'),
      'module.exports = { test: { coverage: { exclude: ["src/**/*.types.ts"] } } };'
    );
    const result = await vitestCoverageExcludeCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('passes when exclude contains *.test.ts', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    writeFileSync(
      join(dir, 'vitest.config.js'),
      'module.exports = { test: { coverage: { exclude: ["src/**/*.test.ts"] } } };'
    );
    const result = await vitestCoverageExcludeCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('fails when exclude contains path ending in .ts (non-conventional)', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    writeFileSync(
      join(dir, 'vitest.config.js'),
      'module.exports = { test: { coverage: { exclude: ["src/config/load.ts"] } } };'
    );
    const result = await vitestCoverageExcludeCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors.some((e) => e.includes('load.ts'))).toBe(true);
  });

  it('fails when exclude contains directory glob that can match .ts', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    writeFileSync(
      join(dir, 'vitest.config.js'),
      'module.exports = { test: { coverage: { exclude: ["src/types/**"] } } };'
    );
    const result = await vitestCoverageExcludeCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors.some((e) => e.includes('src/types/**'))).toBe(true);
  });

  it('loads config from package.json vitest when no vitest.config file', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    writeFileSync(
      join(dir, 'package.json'),
      JSON.stringify({ vitest: { test: { coverage: { exclude: ['**/*.types.ts'] } } } })
    );
    const result = await vitestCoverageExcludeCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('uses mod when config default is null (fallback to mod)', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    writeFileSync(
      join(dir, 'vitest.config.js'),
      'const c = { test: { coverage: { exclude: ["**/*.types.ts"] } } }; module.exports = c; module.exports.default = null;'
    );
    const result = await vitestCoverageExcludeCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('uses config.coverage.exclude when test.coverage.exclude missing and non-array returns []', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    writeFileSync(
      join(dir, 'package.json'),
      JSON.stringify({ vitest: { coverage: { exclude: 'not-array' } } })
    );
    const result = await vitestCoverageExcludeCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('ignores package.json when vitest is non-object (e.g. string)', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    writeFileSync(join(dir, 'package.json'), JSON.stringify({ vitest: 'invalid' }));
    const result = await vitestCoverageExcludeCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('uses default export when config file exports { default: config }', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    writeFileSync(
      join(dir, 'vitest.config.mjs'),
      'const c = { test: { coverage: { exclude: ["**/*.types.ts"] } } }; export { c as default };'
    );
    const result = await vitestCoverageExcludeCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('configFromMod uses default when module has object default', () => {
    const cfg = { test: { coverage: { exclude: ['**/*.types.ts'] } } };
    expect(configFromMod({ default: cfg })).toEqual(cfg);
  });

  it('configFromMod uses mod when default is not object', () => {
    const mod = { default: null, test: { coverage: { exclude: ['**/*.types.ts'] } } };
    expect(configFromMod(mod)).toBe(mod);
  });

  it('configFromMod returns null for non-object input', () => {
    expect(configFromMod(null)).toBe(null);
    expect(configFromMod(42)).toBe(null);
  });

  it('VITEST_CONFIG_NAMES includes expected config file names', () => {
    expect(VITEST_CONFIG_NAMES).toContain('vitest.config.cjs');
    expect(VITEST_CONFIG_NAMES).toContain('vitest.config.js');
    expect(VITEST_CONFIG_NAMES).toContain('vitest.config.mjs');
    expect(VITEST_CONFIG_NAMES).toContain('vitest.config.mts');
    expect(VITEST_CONFIG_NAMES).toContain('vitest.config.ts');
  });

  it('ALLOWED_SUFFIXES matches conventional endings', () => {
    expect(ALLOWED_SUFFIXES.test('foo.d.ts')).toBe(true);
    expect(ALLOWED_SUFFIXES.test('bar.types.ts')).toBe(true);
    expect(ALLOWED_SUFFIXES.test('baz.test.ts')).toBe(true);
    expect(ALLOWED_SUFFIXES.test('baz.spec.ts')).toBe(true);
    expect(ALLOWED_SUFFIXES.test('index.ts')).toBe(true);
    expect(ALLOWED_SUFFIXES.test('baz.ts')).toBe(false);
  });

  it('passes when exclude contains *.spec.ts', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    writeFileSync(
      join(dir, 'vitest.config.js'),
      'module.exports = { test: { coverage: { exclude: ["**/*.spec.ts"] } } };'
    );
    const result = await vitestCoverageExcludeCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('error message includes allowed patterns', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    writeFileSync(
      join(dir, 'vitest.config.js'),
      'module.exports = { test: { coverage: { exclude: ["src/foo.ts"] } } };'
    );
    const result = await vitestCoverageExcludeCheck.run(dir);
    expect(result.ok).toBe(false);
    for (const p of ALLOWED_COVERAGE_EXCLUDE_PATTERNS) expect(result.errors[0]).toContain(p);
    expect(result.errors[0]).toContain('disallowed: "src/foo.ts"');
  });

  it('ignores non-string exclude entries', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    writeFileSync(
      join(dir, 'vitest.config.js'),
      'module.exports = { test: { coverage: { exclude: ["**/*.types.ts", 1, null] } } };'
    );
    const result = await vitestCoverageExcludeCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('skips vitest.config.js that exports null and falls through', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    writeFileSync(join(dir, 'vitest.config.js'), 'module.exports = null;');
    const result = await vitestCoverageExcludeCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('passes when vitest.config.js throws (invalid) and no package.json vitest', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    writeFileSync(join(dir, 'vitest.config.js'), 'throw new Error("bad");');
    const result = await vitestCoverageExcludeCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('passes when package.json exists but is invalid JSON', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    writeFileSync(join(dir, 'package.json'), 'not json');
    const result = await vitestCoverageExcludeCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('reports all disallowed patterns', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    writeFileSync(
      join(dir, 'vitest.config.js'),
      'module.exports = { test: { coverage: { exclude: ["src/foo.ts", "src/types/**"] } } };'
    );
    const result = await vitestCoverageExcludeCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toHaveLength(2);
  });

  it('getCoverageExclude returns [] for null config or non-array exclude', () => {
    expect(getCoverageExclude(null)).toEqual([]);
    expect(
      getCoverageExclude({ test: { coverage: { exclude: 'not-array' as unknown as string[] } } })
    ).toEqual([]);
  });

  it('passes when vitest config has no coverage exclude', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    writeFileSync(join(dir, 'vitest.config.mjs'), 'export default { test: {} };');
    const result = await vitestCoverageExcludeCheck.run(dir);
    expect(result.ok).toBe(true);
  });

  it('loadVitestConfig falls back to package.json vitest on fallbackRoot', () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    const fallback = mkdtempSync(join(tmpdir(), 'vitest-fallback-'));
    writeFileSync(
      join(fallback, 'package.json'),
      '{"vitest":{"test":{"coverage":{"exclude":["**/*.d.ts"]}}}}'
    );
    const config = loadVitestConfig(dir, fallback);
    expect(getCoverageExcludeFromConfig(config)).toEqual(['**/*.d.ts']);
  });

  it('loadVitestConfig falls back to vitest.config.mjs on fallbackRoot', () => {
    const dir = mkdtempSync(join(tmpdir(), 'vitest-exclude-'));
    const fallback = mkdtempSync(join(tmpdir(), 'vitest-fallback-'));
    writeFileSync(
      join(fallback, 'vitest.config.mjs'),
      'export default { test: { coverage: { exclude: ["**/*.d.ts"] } } };'
    );
    const config = loadVitestConfig(dir, fallback);
    expect(getCoverageExcludeFromConfig(config)).toEqual(['**/*.d.ts']);
  });
});
