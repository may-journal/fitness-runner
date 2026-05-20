#!/usr/bin/env node
import { spawnSync } from 'node:child_process';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const runnerEntry = join(dirname(fileURLToPath(import.meta.url)), '../dist/index.js');
const result = spawnSync(process.execPath, [runnerEntry, ...process.argv.slice(2)], {
  stdio: 'inherit',
});
process.exit(result.status ?? 1);
