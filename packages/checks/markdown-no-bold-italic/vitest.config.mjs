import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vitest/config';

const packageRoot = dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  esbuild: { target: 'ES2022' },
  resolve: {
    alias: {
      '@mayjournal/fitness': join(packageRoot, '../../runner/dist/src/types/index.types.js'),
    },
  },
  test: { globals: true },
});
