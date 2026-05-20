#!/usr/bin/env node
import { spawnSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { existsSync, readFileSync } from 'node:fs';
import { dirname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const eslintConfig = join(dirname(fileURLToPath(import.meta.url)), '../config/eslint.config.cjs');

function findWorkspaceRoot(start) {
  let dir = start;
  while (true) {
    const parent = dirname(dir);
    if (parent === dir) return start;
    const pkgPath = join(dir, 'package.json');
    if (existsSync(pkgPath)) {
      const pkg = JSON.parse(readFileSync(pkgPath, 'utf8'));
      if (pkg.workspaces) return dir;
    }
    dir = parent;
  }
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
  }

  const result = spawnSync('eslint', ['-c', eslintConfig, ...paths], {
    cwd: root,
    env: { ...process.env, FITNESS_TSCONFIG_ROOT: root },
    stdio: 'inherit',
  });
  process.exit(result.status ?? 1);
}
