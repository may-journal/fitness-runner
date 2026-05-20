#!/usr/bin/env node
import { spawnSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { rmSync, unlinkSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { findWorkspaceRoot, readPackageName } from './workspace-root.js';

const require = createRequire(import.meta.url);
const configDir = join(dirname(fileURLToPath(import.meta.url)), '../config');

function runCheckBuild(cwd) {
  require(join(configDir, 'generate-tsconfig-json.cjs')).generateTsconfigJson(configDir);

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

function runRunnerBuild(root, cwd) {
  const prep = spawnSync(
    'tsconfig.js',
    [
      '--once',
      '--root',
      'packages/shared/config',
      '--extensions=js,cjs',
      '--extends-strategy=ignore',
    ],
    { cwd: root, stdio: 'inherit' }
  );
  if (prep.status !== 0) process.exit(prep.status ?? 1);

  rmSync(join(cwd, 'dist'), { force: true, recursive: true });

  const result = spawnSync('tsc', ['-p', 'packages/shared/config/tsconfig.json'], {
    cwd: root,
    stdio: 'inherit',
  });
  process.exit(result.status ?? 1);
}

export function runBuild() {
  const cwd = process.cwd();
  const name = readPackageName(cwd);
  if (name === '@mayjournal/fitness') {
    runRunnerBuild(findWorkspaceRoot(cwd), cwd);
    return;
  }
  runCheckBuild(cwd);
}
