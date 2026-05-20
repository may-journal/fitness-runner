import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import { defineConfig } from 'vitest/config';

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), '../../..');

export default defineConfig({
  esbuild: { target: 'ES2022' },
  resolve: {
    alias: {
      '@mayjournal/fitness': join(repoRoot, 'src/types/index.types.ts'),
    },
  },
  test: {
    globals: true,
  },
});
