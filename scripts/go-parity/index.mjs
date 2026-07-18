#!/usr/bin/env node
/**
 * Side-by-side parity harness: runs every check name through the TypeScript
 * check module AND the Go check binary against this repo, then diffs
 * ok / errors / filesChecked. Exits non-zero on any disagreement outside the
 * documented exceptions. Requires `npm run build` (TS dist) and
 * `npm run build:go` (Go binaries) to have run.
 */
import { execFileSync } from 'node:child_process';
import { existsSync, readdirSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../..');

/** Check names discovered from the built Go binaries in binDir (fitness-check-*). */
export function discoverCheckNames(binDir) {
  return readdirSync(binDir)
    .filter((f) => f.startsWith(BIN_PREFIX))
    .map((f) => f.slice(BIN_PREFIX.length))
    .sort();
}

const BIN_PREFIX = 'fitness-check-';

/** Checks whose filesChecked semantics differ by documented design (jscpd: the Go soul counts every scanned file; the TS check parses jscpd's recognized-format Total row). */
const FILES_CHECKED_EXEMPT = new Set(['jscpd']);

/** Normalizes an error line for comparison: absolute repo paths become relative. */
export function normalizeError(line, root) {
  return line
    .split(root + '/')
    .join('')
    .trimEnd();
}

/** Compares one check's TS and Go results; returns a list of human-readable differences. */
export function diffResults(name, ts, go, root) {
  const diffs = [];
  if (ts.ok !== go.ok) diffs.push(`ok: ts=${ts.ok} go=${go.ok}`);
  if (!FILES_CHECKED_EXEMPT.has(name) && ts.filesChecked !== go.filesChecked) {
    diffs.push(`filesChecked: ts=${ts.filesChecked} go=${go.filesChecked}`);
  }
  const tsErrors = ts.errors.map((e) => normalizeError(e, root));
  const goErrors = go.errors.map((e) => normalizeError(e, root));
  if (tsErrors.join('\n') !== goErrors.join('\n')) {
    diffs.push(`errors differ (ts ${tsErrors.length} vs go ${goErrors.length})`);
  }
  return diffs;
}

/** Runs the TS check module in-process; returns {ok, errors, filesChecked}. */
async function runTsCheck(name) {
  const mod = await import(`@mayjournal/fitness-checks/checks/${name}`);
  const check = mod.default;
  const context = {
    enabledCheckNames: [name],
    registeredCheckNames: [name],
    stagedFiles: [],
  };
  const result = await check.run(repoRoot, context);
  return { errors: result.errors, filesChecked: result.meta?.filesChecked ?? -1, ok: result.ok };
}

/** Runs the Go check binary; returns {ok, errors, filesChecked}. */
function runGoCheck(name) {
  const bin = join(repoRoot, 'go/bin', BIN_PREFIX + name);
  if (!existsSync(bin)) throw new Error(`missing Go binary: ${bin} (run npm run build:go)`);
  let stdout = '';
  try {
    stdout = execFileSync(bin, ['--root', repoRoot], {
      cwd: repoRoot,
      encoding: 'utf8',
      env: {
        ...process.env,
        FITNESS_CHECK_NAME: name,
        FITNESS_ENABLED_CHECKS: name,
        FITNESS_STAGED_FILES: '',
      },
      stdio: ['ignore', 'pipe', 'ignore'],
    });
  } catch (err) {
    stdout = String(err.stdout ?? '');
  }
  const lines = stdout.trim().split('\n').filter(Boolean);
  return JSON.parse(lines[lines.length - 1]);
}

/** Runs the full harness; returns the number of disagreeing checks. */
async function main() {
  let failures = 0;
  const names = discoverCheckNames(join(repoRoot, 'go/bin'));
  for (const name of names) {
    const started = Date.now();
    const [ts, go] = [await runTsCheck(name), runGoCheck(name)];
    const diffs = diffResults(name, ts, go, repoRoot);
    const ms = Date.now() - started;
    if (diffs.length === 0) {
      process.stdout.write(`MATCH ${name} (ok=${ts.ok}, files=${ts.filesChecked}, ${ms}ms)\n`);
    } else {
      failures += 1;
      process.stdout.write(`DIFF  ${name}: ${diffs.join('; ')}\n`);
    }
  }
  process.stdout.write(`\n${names.length - failures}/${names.length} checks agree\n`);
  return failures;
}

const invokedDirectly = process.argv[1] === fileURLToPath(import.meta.url);
if (invokedDirectly) {
  main().then((failures) => process.exit(failures > 0 ? 1 : 0));
}
