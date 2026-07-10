#!/usr/bin/env node
import { spawnSync } from 'node:child_process';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { findWorkspaceRoot, readPackageName } from './workspace-root.js';

const COVERAGE_FLAG = '--coverage';

export function runTest(argv = process.argv.slice(2)) {
  const cwd = process.cwd();
  const name = readPackageName(cwd);

  if (name === '@mayjournal/fitness') {
    const root = findWorkspaceRoot(cwd);
    const configPath = join(dirname(fileURLToPath(import.meta.url)), '../config/vitest.config.mjs');
    const vitestArgs = ['run', '--config', configPath];
    if (argv.includes(COVERAGE_FLAG)) vitestArgs.push(COVERAGE_FLAG);
    const result = spawnSync('vitest', vitestArgs, { cwd: root, stdio: 'inherit' });
    process.exit(result.status ?? 1);
  }

  const configPath = join(
    dirname(fileURLToPath(import.meta.url)),
    '../config/vitest.check.config.mjs'
  );
  const extraExclude = [];
  const vitestArgs = [];

  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (arg === COVERAGE_FLAG) continue;
    if (arg === '--coverage-exclude' && argv[i + 1]) {
      extraExclude.push(argv[++i]);
      continue;
    }
    vitestArgs.push(arg);
  }

  const env = { ...process.env };
  if (argv.includes(COVERAGE_FLAG)) env.FITNESS_SHARED_TEST_COVERAGE = '1';
  if (extraExclude.length > 0) {
    env.FITNESS_SHARED_COVERAGE_EXTRA_EXCLUDE = extraExclude.join(',');
  }

  const result = spawnSync('vitest', ['run', '--config', configPath, ...vitestArgs], {
    cwd,
    env,
    stdio: 'inherit',
  });
  process.exit(result.status ?? 1);
}
