import { defineConfig } from 'vitest/config';

export default defineConfig({
  esbuild: { target: 'ES2022' },
  test: {
    coverage: {
      exclude: [
        '**/*.bench.ts',
        '**/*.d.ts',
        '**/*.test.ts',
        '**/*.spec.ts',
        '**/*.types.ts',
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
