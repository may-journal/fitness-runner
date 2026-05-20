import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vitest/config';

const packageRoot = dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  esbuild: { target: 'ES2022' },
  resolve: {
    alias: {
      '@mayjournal/fitness': join(packageRoot, '../../runner/dist/types/index.types.js'),
    },
  },
  test: {
    coverage: {
      exclude: ['**/*.d.ts', '**/*.test.ts'],
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
