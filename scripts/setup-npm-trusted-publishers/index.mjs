#!/usr/bin/env node
/** @deprecated Use provision-npm-packages instead. */
import { spawnSync } from 'node:child_process';
import { dirname, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

/** @returns {number} exit code */
export function runSetupNpmTrustedPublishers(
  argv = process.argv.slice(2),
  { spawnSync: spawnSyncImpl = spawnSync, execPath = process.execPath } = {}
) {
  const provision = join(
    dirname(fileURLToPath(import.meta.url)),
    '../provision-npm-packages/index.mjs'
  );
  const result = spawnSyncImpl(execPath, [provision, ...argv], { stdio: 'inherit' });
  return result.status ?? 1;
}

const isMain =
  process.argv[1] &&
  fileURLToPath(import.meta.url) === fileURLToPath(pathToFileURL(process.argv[1]).href);
if (isMain) {
  process.exit(runSetupNpmTrustedPublishers());
}
