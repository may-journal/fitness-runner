#!/usr/bin/env node
import { spawnSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { existsSync, writeFileSync } from 'node:fs';
import { dirname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';
import { findWorkspaceRoot } from './workspace-root.js';

const require = createRequire(import.meta.url);
const eslintConfig = join(dirname(fileURLToPath(import.meta.url)), '../config/eslint.config.cjs');

/** Ephemeral tsconfig so projectService maps runner sources (not committed). */
function ensureRunnerLintTsconfig(root) {
  const runnerDir = join(root, 'packages/runner');
  if (!existsSync(runnerDir)) return;
  writeFileSync(
    join(runnerDir, 'tsconfig.json'),
    `${JSON.stringify(
      {
        compilerOptions: { noEmit: true, rootDir: './src' },
        exclude: ['src/**/*.test.ts'],
        extends: '../shared/config/tsconfig.check.json',
        include: ['src/**/*.ts'],
      },
      null,
      2
    )}\n`
  );
}

export function runLint(argv = process.argv.slice(3)) {
  const cwd = process.cwd();
  const root = findWorkspaceRoot(cwd);
  const paths = argv.length > 0 ? argv : [relative(root, cwd) || '.'];

  if (existsSync(join(root, 'packages/shared/config/tsconfig.cjs'))) {
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
    require(join(root, 'packages/shared/config/generate-tsconfig-json.cjs')).generateTsconfigJson(
      join(root, 'packages/shared/config')
    );
    ensureRunnerLintTsconfig(root);
  }

  const result = spawnSync('eslint', ['-c', eslintConfig, ...paths], {
    cwd: root,
    env: { ...process.env, FITNESS_TSCONFIG_ROOT: root },
    stdio: 'inherit',
  });
  process.exit(result.status ?? 1);
}
