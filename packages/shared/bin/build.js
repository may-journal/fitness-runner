#!/usr/bin/env node
import { spawnSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { unlinkSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const configDir = join(dirname(fileURLToPath(import.meta.url)), '../config');

export function runBuild() {
  require(join(configDir, 'generate-tsconfig-json.cjs')).generateTsconfigJson(configDir);

  const cwd = process.cwd();
  const tsconfigPath = join(cwd, '.fitness-check-tsconfig.json');

  writeFileSync(
    tsconfigPath,
    `${JSON.stringify(
      {
        compilerOptions: {
          outDir: './dist',
          paths: {
            '@mayjournal/fitness': ['../../runner/dist/types/index.types.d.ts'],
          },
          rootDir: './src',
        },
        exclude: ['src/**/*.test.ts'],
        extends: '@mayjournal/fitness-shared/tsconfig.check',
        include: ['src/**/*.ts'],
      },
      null,
      2
    )}\n`
  );

  const result = spawnSync('tsc', ['-p', tsconfigPath], { cwd, stdio: 'inherit' });
  try {
    unlinkSync(tsconfigPath);
  } catch {
    // ignore cleanup errors
  }
  process.exit(result.status ?? 1);
}
