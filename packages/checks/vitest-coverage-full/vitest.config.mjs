import { defineConfig, mergeConfig } from 'vitest/config';
import base from '@mayjournal/fitness-shared/vitest.config';

export default mergeConfig(
  base,
  defineConfig({
    test: {
      coverage: {
        include: ['src/**/*.ts'],
      },
    },
  })
);
