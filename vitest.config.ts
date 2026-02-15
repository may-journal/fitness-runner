import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    globals: true,
    coverage: {
      provider: 'v8',
      reporter: ['text',
'lcov'],
      include: ['src/**/*.ts'],
      exclude: ['src/**/*.d.ts',
'src/types/**',
'src/check-node-version.ts',
'src/config/load.ts',
'src/**/*.test.ts',
'src/**/*.spec.ts'],
      thresholds: {
        statements: 100,
        branches: 100,
        functions: 100,
        lines: 100,
      },
    },
  },
  esbuild: { target: 'ES2022' },
});
