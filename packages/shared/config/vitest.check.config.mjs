import { resolve } from 'node:path';
import { defineConfig } from 'vitest/config';

/** @typedef {{ coverage?: boolean; coverageExtraExclude?: string[] }} CheckVitestOptions */

/**
 * Vitest config for @mayjournal/fitness check packages (paths relative to packageRoot).
 * @param {string} [packageRoot]
 * @param {CheckVitestOptions} [options]
 */
export function createCheckVitestConfig(packageRoot = process.cwd(), options = {}) {
  const fitnessTypes = resolve(packageRoot, '../../runner/dist/types/index.types.js');
  /** @type {import('vitest/config').UserConfig} */
  const config = {
    esbuild: { target: 'ES2022' },
    resolve: {
      alias: {
        '@mayjournal/fitness': fitnessTypes,
      },
    },
    root: packageRoot,
    test: {
      globals: true,
      include: ['src/**/*.test.ts'],
    },
  };
  if (options.coverage) {
    const exclude = [
      '**/*.d.ts',
      '**/*.test.ts',
      '**/*.spec.ts',
      '**/*.types.ts',
      ...(options.coverageExtraExclude ?? []),
    ];
    config.test = {
      ...config.test,
      coverage: {
        exclude,
        include: ['src/**/*.ts'],
        provider: 'v8',
        reporter: ['text', 'lcov'],
        thresholds: {
          branches: 100,
          functions: 100,
          lines: 100,
          statements: 100,
        },
      },
    };
  }
  return defineConfig(config);
}

const coverage = process.env.FITNESS_SHARED_TEST_COVERAGE === '1';
const coverageExtraExclude =
  process.env.FITNESS_SHARED_COVERAGE_EXTRA_EXCLUDE?.split(',')
    .map((s) => s.trim())
    .filter(Boolean) ?? [];

export default createCheckVitestConfig(process.cwd(), { coverage, coverageExtraExclude });
