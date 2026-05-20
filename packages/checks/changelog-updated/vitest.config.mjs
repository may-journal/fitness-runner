import { defineConfig } from 'vitest/config';

export default defineConfig({
  esbuild: { target: 'ES2022' },
  test: {
    globals: true,
    include: ['src/**/*.test.ts'],
  },
});
