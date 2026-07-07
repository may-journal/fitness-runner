#!/usr/bin/env node
/** Copies built check packages into dist/checks/ for @mayjournal/fitness-checks publish. */
import { spawnSync } from 'node:child_process';
import { cpSync, existsSync, mkdirSync, rmSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

/**
 * Every check to bundle for publishing — a superset of src/index.ts defaultChecks; opt-in checks
 * (jscpd, swiftlint) belong here too so `.fitnessrc` `checks` can resolve them, even though they
 * never run by default.
 *
 * A bare string is a package whose name === check name, bundled from its `dist/index.js`. An object
 * maps ONE source package to MULTIPLE check names, each from its own entry module (`dist/<entry>.js`)
 * — used for flavor packs like `markdown-filename-convention`, which exports a kebab-case and a
 * camelCase flavor of one shared function under two check names.
 */
const CHECK_SPECS = [
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
  {
    checks: [
      { entry: 'kebab-case', name: 'markdown-filename-kebab-case' },
      { entry: 'camel-case', name: 'markdown-filename-camel-case' },
    ],
    package: 'markdown-filename-convention',
  },
];

const bundleRoot = join(dirname(fileURLToPath(import.meta.url)), '..');
const repoRoot = join(bundleRoot, '../..');
const checksDest = join(bundleRoot, 'dist/checks');

/** Builds a check workspace and returns its `dist` dir, exiting on failure. */
function buildCheckPackage(pkg) {
  const workspace = `@mayjournal/fitness-check-${pkg}`;
  // Always rebuild before copying: a prior build may have left a stale dist, and workspace build
  // order does not guarantee this check compiled before the bundle. Copying a stale dist silently
  // ships old code (has caused false check failures).
  console.log(`building ${workspace}…`);
  const build = spawnSync('npm', ['run', 'build', '-w', workspace], {
    cwd: repoRoot,
    stdio: 'inherit',
  });
  if (build.status !== 0) process.exit(build.status ?? 1);
  const src = join(repoRoot, 'packages/checks', pkg, 'dist');
  if (!existsSync(src)) {
    console.error(`Missing ${src} after build.`);
    process.exit(1);
  }
  return src;
}

/** Copies a package `dist` to dist/checks/<name>, making `<entry>.*` the resolved `index.*`. */
function bundleCheck(src, name, entry) {
  const dest = join(checksDest, name);
  rmSync(dest, { force: true, recursive: true }); // clear any stale bundled copy first
  cpSync(src, dest, { force: true, recursive: true });
  if (entry !== 'index') {
    // The bundle resolves `checks/<name>` via `./dist/checks/*/index.js`, so promote the flavor's
    // entry module to index.*. Entry modules import only from published packages (no sibling
    // relative imports), so the copied index.js resolves standalone.
    for (const ext of ['js', 'd.ts', 'd.ts.map']) {
      const from = join(dest, `${entry}.${ext}`);
      if (existsSync(from)) cpSync(from, join(dest, `index.${ext}`), { force: true });
    }
  }
  console.log(`bundled check: ${name}`);
}

mkdirSync(checksDest, { recursive: true });

for (const spec of CHECK_SPECS) {
  if (typeof spec === 'string') {
    bundleCheck(buildCheckPackage(spec), spec, 'index');
    continue;
  }
  const src = buildCheckPackage(spec.package);
  for (const { name, entry } of spec.checks) bundleCheck(src, name, entry);
}
