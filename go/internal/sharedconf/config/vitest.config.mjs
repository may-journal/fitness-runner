import { defineConfig } from 'vitest/config';
import { ES_TARGET } from './constants.cjs';

export default defineConfig({
  esbuild: { target: ES_TARGET },
  test: {
    coverage: {
      // Empty on purpose: Vitest's default exclude list skips tests and declarations.
      exclude: [],
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
    globals: true,
    include: ['src/**/*.test.ts'],
  },
});
