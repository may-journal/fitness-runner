#!/usr/bin/env node
/** @deprecated Use provision-npm-packages instead. */
import { spawnSync } from 'node:child_process';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const provision = join(
  dirname(fileURLToPath(import.meta.url)),
  '../provision-npm-packages/index.mjs'
);
const result = spawnSync(process.execPath, [provision, ...process.argv.slice(2)], {
  stdio: 'inherit',
});
process.exit(result.status ?? 1);
