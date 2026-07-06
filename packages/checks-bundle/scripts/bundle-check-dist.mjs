#!/usr/bin/env node
/** Copies built check packages into dist/checks/ for @mayjournal/fitness-checks publish. */
import { spawnSync } from 'node:child_process';
import { cpSync, existsSync, mkdirSync, rmSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

/** Every check to bundle for publishing — a superset of src/index.ts defaultChecks; opt-in checks (jscpd, swiftlint) belong here too so `.fitnessrc` `checks` can resolve them, even though they never run by default. */
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
  'jscpd',
  'swiftlint',
  'mermaid-callouts',
  'mermaid-callout-why',
  'mermaid-diagram-prose',
  'mermaid-legend',
  'mermaid-level-bleed',
  'dependency-currency',
  'gitignore-why',
  'no-eslint-disable',
  'markdown-filename-convention',
];

const bundleRoot = join(dirname(fileURLToPath(import.meta.url)), '..');
const repoRoot = join(bundleRoot, '../..');

for (const name of CHECK_NAMES) {
  const src = join(repoRoot, 'packages/checks', name, 'dist');
  const workspace = `@mayjournal/fitness-check-${name}`;
  // Always rebuild before copying: a prior build may have left a stale dist, and workspace build
  // order does not guarantee this check compiled before the bundle. Copying a stale dist silently
  // ships old code (has caused false check failures).
  console.log(`building ${workspace}…`);
  const build = spawnSync('npm', ['run', 'build', '-w', workspace], {
    cwd: repoRoot,
    stdio: 'inherit',
  });
  if (build.status !== 0) process.exit(build.status ?? 1);
  if (!existsSync(src)) {
    console.error(`Missing ${src} after build.`);
    process.exit(1);
  }
  const dest = join(bundleRoot, 'dist/checks', name);
  rmSync(dest, { force: true, recursive: true }); // clear any stale bundled copy first
  mkdirSync(join(bundleRoot, 'dist/checks'), { recursive: true });
  cpSync(src, dest, { force: true, recursive: true });
  console.log(`bundled check: ${name}`);
}
