#!/usr/bin/env node
import { spawnSync } from 'node:child_process';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

export function runTest(argv = process.argv.slice(2)) {
  const configPath = join(
    dirname(fileURLToPath(import.meta.url)),
    '../config/vitest.check.config.mjs'
  );
  const extraExclude = [];
  const vitestArgs = [];

  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (arg === '--coverage') continue;
    if (arg === '--coverage-exclude' && argv[i + 1]) {
      extraExclude.push(argv[++i]);
      continue;
    }
    vitestArgs.push(arg);
  }

  const env = { ...process.env };
  if (argv.includes('--coverage')) env.FITNESS_SHARED_TEST_COVERAGE = '1';
  if (extraExclude.length > 0) {
    env.FITNESS_SHARED_COVERAGE_EXTRA_EXCLUDE = extraExclude.join(',');
  }

  const result = spawnSync('vitest', ['run', '--config', configPath, ...vitestArgs], {
    cwd: process.cwd(),
    env,
    stdio: 'inherit',
  });
  process.exit(result.status ?? 1);
}
