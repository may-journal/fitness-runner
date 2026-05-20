#!/usr/bin/env node
/** Copies built check packages into dist/checks/ for @mayjournal/fitness-checks publish. */
import { spawnSync } from 'node:child_process';
import { cpSync, existsSync, mkdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

/** Keep in sync with src/index.ts defaultChecks. */
const CHECK_NAMES = [
  'read-repo-first',
  'changelog',
  'changelog-updated',
  'cspell',
  'eslint',
  'markdown-no-bold-italic',
  'prettier',
  'node-version',
  'markdown-front-matter',
  'semantic-commit',
  'vitest-coverage-exclude',
  'vitest-coverage-full',
];

const bundleRoot = join(dirname(fileURLToPath(import.meta.url)), '..');
const repoRoot = join(bundleRoot, '../..');

for (const name of CHECK_NAMES) {
  const src = join(repoRoot, 'packages/checks', name, 'dist');
  const workspace = `@mayjournal/fitness-check-${name}`;
  if (!existsSync(src)) {
    console.log(`building ${workspace}…`);
    const build = spawnSync('npm', ['run', 'build', '-w', workspace], {
      cwd: repoRoot,
      stdio: 'inherit',
    });
    if (build.status !== 0) process.exit(build.status ?? 1);
  }
  if (!existsSync(src)) {
    console.error(`Missing ${src} after build.`);
    process.exit(1);
  }
  const dest = join(bundleRoot, 'dist/checks', name);
  mkdirSync(join(bundleRoot, 'dist/checks'), { recursive: true });
  cpSync(src, dest, { force: true, recursive: true });
  console.log(`bundled check: ${name}`);
}
