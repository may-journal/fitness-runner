import { defineConfig } from 'vitest/config';
import { DTS_GLOB, ES_TARGET, SPEC_GLOB, TEST_GLOB, TYPES_GLOB } from './constants.cjs';

export default defineConfig({
  esbuild: { target: ES_TARGET },
  test: {
    coverage: {
      exclude: [
        '**/*.bench.ts',
        DTS_GLOB,
        TEST_GLOB,
        SPEC_GLOB,
        TYPES_GLOB,
        'packages/runner/src/index.ts',
        'packages/runner/src/runner/index.ts',
      ],
      include: ['packages/runner/src/**/*.ts'],
      provider: 'v8',
      reporter: ['text', 'lcov'],
      thresholds: {
        branches: 100,
        functions: 100,
        lines: 100,
        statements: 100,
      },
    },
    globals: true,
    include: ['packages/runner/src/**/*.test.ts'],
  },
});
