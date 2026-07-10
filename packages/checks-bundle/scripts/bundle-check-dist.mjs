#!/usr/bin/env node
/** Copies built check packages into dist/checks/ for @mayjournal/fitness-checks publish. */
import { spawnSync } from 'node:child_process';
import { cpSync, existsSync, mkdirSync, readdirSync, readFileSync, rmSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import {
  NPM,
  NPM_SCRIPT_BUILD,
  PACKAGE_JSON,
  RUN_SUBCOMMAND,
} from '../../../scripts/constants.cjs';

const bundleRoot = join(dirname(fileURLToPath(import.meta.url)), '..');
const repoRoot = join(bundleRoot, '../..');
const checksRoot = join(repoRoot, 'packages/checks');
const checksDest = join(bundleRoot, 'dist/checks');

/**
 * Every check to bundle for publishing, derived from the packages/checks/* directories (closes the
 * hand-maintained-list drift, #41) — a superset of src/index.ts defaultChecks; opt-in checks
 * (jscpd, swiftlint) are bundled too so `.fitnessrc` `checks` can resolve them, even though they
 * never run by default.
 *
 * A directory bundles as a package whose name === check name, from its `dist/index.js` — unless its
 * package.json has a `fitnessChecks` field mapping ONE source package to MULTIPLE check names, each
 * from its own entry module (`dist/<entry>.js`). That form is for flavor packs like
 * `markdown-filename-convention`, which exports a kebab-case and a camelCase flavor of one shared
 * function under two check names.
 */
function deriveCheckSpecs() {
  return (
    readdirSync(checksRoot, { withFileTypes: true })
      .filter((entry) => entry.isDirectory())
      // A dir without package.json is not a check package yet (e.g. a placeholder) — skip it.
      .filter((entry) => existsSync(join(checksRoot, entry.name, PACKAGE_JSON)))
      .map((entry) => {
        const manifest = JSON.parse(
          readFileSync(join(checksRoot, entry.name, PACKAGE_JSON), 'utf8')
        );
        return manifest.fitnessChecks
          ? { checks: manifest.fitnessChecks, package: entry.name }
          : entry.name;
      })
  );
}

const CHECK_SPECS = deriveCheckSpecs();

/** Builds a check workspace and returns its `dist` dir, exiting on failure. */
function buildCheckPackage(pkg) {
  const workspace = `@mayjournal/fitness-check-${pkg}`;
  // Always rebuild before copying: a prior build may have left a stale dist, and workspace build
  // order does not guarantee this check compiled before the bundle. Copying a stale dist silently
  // ships old code (has caused false check failures).
  console.log(`building ${workspace}…`);
  const build = spawnSync(NPM, [RUN_SUBCOMMAND, NPM_SCRIPT_BUILD, '-w', workspace], {
    cwd: repoRoot,
    stdio: 'inherit',
  });
  if (build.status !== 0) process.exit(build.status ?? 1);
  const src = join(checksRoot, pkg, 'dist');
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
