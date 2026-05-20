import { defineConfig } from 'vitest/config';

export default defineConfig({
  esbuild: { target: 'ES2022' },
  test: {
    exclude: ['dist/**'],
    include: ['src/**/*.bench.ts'],
  },
});
