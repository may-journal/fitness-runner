const { defineConfig } = require('vitest/config');

module.exports = defineConfig({
  esbuild: { target: 'ES2022' },
  test: {
    coverage: {
      exclude: ['**/*.d.ts', '**/*.test.ts', '**/*.spec.ts', '**/*.types.ts'],
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
  },
});
