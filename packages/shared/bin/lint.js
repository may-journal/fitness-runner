#!/usr/bin/env node
import { spawnSync } from 'node:child_process';
import { dirname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';
import { findWorkspaceRoot } from './workspace-root.js';

const eslintConfig = join(dirname(fileURLToPath(import.meta.url)), '../config/eslint.config.mjs');

export function runLint(argv = process.argv.slice(3)) {
  const cwd = process.cwd();
  const root = findWorkspaceRoot(cwd);
  const paths = argv.length > 0 ? argv : [relative(root, cwd) || '.'];

  const result = spawnSync('eslint', ['-c', eslintConfig, ...paths], {
    cwd: root,
    stdio: 'inherit',
  });
  process.exit(result.status ?? 1);
}
