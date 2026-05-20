#!/usr/bin/env node
/**
 * Audit publish footprint: publint + npm pack --dry-run per publishable workspace.
 * See https://github.com/may-journal/fitness-runner/issues/13
 */
import { spawnSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { listPublishablePackages } from '../list-publishable-packages/index.mjs';
import { runAttwAudit } from './attw.mjs';
import {
  checkRunnerTarballGate,
  DEFAULT_RUNNER_MAX_TARBALL,
  formatBytes,
  formatMarkdownReport,
} from './format-report.mjs';

const root = join(dirname(fileURLToPath(import.meta.url)), '../..');

/** @returns {string[]} argv to run publint from root devDependency */
function publintCommand() {
  const pkgPath = join(root, 'node_modules/publint/package.json');
  const bin = JSON.parse(readFileSync(pkgPath, 'utf8')).bin;
  const cli = join(dirname(pkgPath), String(bin).replace(/^\.\//, ''));
  return [cli, 'run'];
}

/**
 * @param {string} jsonText npm pack --dry-run --json stdout
 * @returns {{ tarball: number, unpacked: number }}
 */
export function parsePackDryRun(jsonText) {
  const entries = JSON.parse(jsonText.trim());
  const entry = Array.isArray(entries) ? entries[0] : entries;
  if (!entry || typeof entry.size !== 'number') {
    throw new Error('Unexpected npm pack --dry-run --json output');
  }
  return { tarball: entry.size, unpacked: entry.unpackedSize ?? entry.size };
}

/**
 * @param {{ status: number | null, stdout?: string, stderr?: string }} result
 * @returns {{ ok: boolean, errors: number, warnings: number, output: string }}
 */
export function summarizePublint(result) {
  const text = `${result.stdout ?? ''}${result.stderr ?? ''}`.trim();
  const ok = (result.status ?? 1) === 0;
  const countSection = (label) => {
    if (!text.includes(`${label}:`)) return 0;
    const chunk = text.split(`${label}:`)[1]?.split(/\n[A-Z]/)[0] ?? '';
    return chunk.split('\n').filter((line) => /^\d+\./.test(line)).length;
  };
  const errors = ok ? 0 : Math.max(countSection('Errors'), 1);
  const warnings = countSection('Warnings');
  return { errors, ok, output: text, warnings };
}

/**
 * @param {string} dir workspace path relative to repo root
 * @param {{ execSync?: typeof spawnSync, publintBin?: string }} [hooks]
 */
export function auditOnePackage(dir, hooks = {}) {
  const execSync = hooks.execSync ?? spawnSync;
  const cwd = join(root, dir);
  const publintArgv = hooks.publintArgv ?? [...publintCommand(), '.'];

  const pack = execSync('npm', ['pack', '--dry-run', '--json'], {
    cwd,
    encoding: 'utf8',
    shell: false,
  });
  if (pack.status !== 0) {
    throw new Error(`npm pack failed in ${dir}: ${pack.stderr || pack.stdout}`);
  }

  const lint = execSync(process.execPath, publintArgv, {
    cwd,
    encoding: 'utf8',
    shell: false,
  });

  const sizes = parsePackDryRun(pack.stdout);
  const publint = summarizePublint(lint);

  return { dir, publint, sizes };
}

/**
 * @param {{ skipBuild?: boolean, strict?: boolean, json?: boolean, attw?: boolean, gateRunner?: boolean, runnerMaxTarball?: number, markdown?: boolean, execSync?: typeof spawnSync }} [opts]
 */
export async function runPublishAudit(opts = {}) {
  const execSync = opts.execSync ?? spawnSync;

  if (!opts.skipBuild) {
    const build = execSync('npm', ['run', 'build'], { cwd: root, encoding: 'utf8', shell: false });
    if (build.status !== 0) {
      process.stderr.write(build.stderr || build.stdout);
      process.exit(build.status ?? 1);
    }
  }

  const packages = listPublishablePackages();
  const rows = [];

  for (const { dir, name } of packages) {
    try {
      const result = auditOnePackage(dir, { execSync, publintArgv: opts.publintArgv });
      rows.push({ dir: result.dir, name, publint: result.publint, sizes: result.sizes });
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      rows.push({ dir, error: message, name });
    }
  }

  const publintFailures = rows.filter((r) => r.publint && !r.publint.ok);
  const packErrors = rows.filter((r) => r.error);
  const attwRows = opts.attw ? runAttwAudit({ execSync }) : [];
  const gate = opts.gateRunner
    ? checkRunnerTarballGate(rows, { runnerMaxTarball: opts.runnerMaxTarball })
    : undefined;

  const report = { attw: attwRows, gate, packages: rows };

  if (opts.json) {
    console.log(JSON.stringify(report, null, 2));
  } else if (opts.markdown) {
    process.stdout.write(formatMarkdownReport(rows, { attwRows, gate }));
  } else {
    printTable(rows);
    if (publintFailures.length) {
      process.stderr.write('\nPublint issues:\n');
      for (const row of publintFailures) {
        process.stderr.write(`\n## ${row.name} (${row.dir})\n${row.publint.output}\n`);
      }
    }
    if (packErrors.length) {
      process.stderr.write('\nPack errors:\n');
      for (const row of packErrors) {
        process.stderr.write(`${row.name}: ${row.error}\n`);
      }
    }
  }

  if (packErrors.length) process.exit(1);
  if (opts.strict && publintFailures.length) process.exit(1);
  if (opts.gateRunner && gate && !gate.ok) process.exit(1);
  return report;
}

/** @param {Awaited<ReturnType<typeof runPublishAudit>>} rows */
function printTable(rows) {
  const sorted = [...rows].sort((a, b) => (b.sizes?.tarball ?? 0) - (a.sizes?.tarball ?? 0));
  const nameW = Math.max(4, ...sorted.map((r) => r.name.length));
  console.log('Publish audit (tarball = npm pack --dry-run size)\n');
  console.log(
    `${'package'.padEnd(nameW)}  ${'tarball'.padStart(10)}  ${'unpacked'.padStart(10)}  publint`
  );
  console.log(`${'-'.repeat(nameW)}  ${'-'.repeat(10)}  ${'-'.repeat(10)}  -------`);
  for (const row of sorted) {
    if (row.error) {
      console.log(`${row.name.padEnd(nameW)}  ${'ERROR'.padStart(10)}  ${''.padStart(10)}  —`);
      continue;
    }
    const lint = row.publint.ok ? 'ok' : `${row.publint.errors} err`;
    console.log(
      `${row.name.padEnd(nameW)}  ${formatBytes(row.sizes.tarball).padStart(10)}  ${formatBytes(row.sizes.unpacked).padStart(10)}  ${lint}`
    );
  }
  const total = sorted.reduce((n, r) => n + (r.sizes?.tarball ?? 0), 0);
  console.log(`\nTotal tarball (all ${sorted.length} packages): ${formatBytes(total)}`);
}

function parseArgs(argv) {
  const maxIdx = argv.indexOf('--runner-max-tarball');
  const maxRaw = maxIdx >= 0 ? argv[maxIdx + 1] : undefined;
  const runnerMaxTarball = maxRaw ? Number(maxRaw) : DEFAULT_RUNNER_MAX_TARBALL;
  return {
    attw: argv.includes('--attw'),
    gateRunner: argv.includes('--gate-runner'),
    json: argv.includes('--json'),
    markdown: argv.includes('--markdown'),
    runnerMaxTarball: Number.isFinite(runnerMaxTarball)
      ? runnerMaxTarball
      : DEFAULT_RUNNER_MAX_TARBALL,
    skipBuild: argv.includes('--no-build'),
    strict: argv.includes('--strict'),
  };
}

const isMain =
  process.argv[1] &&
  fileURLToPath(import.meta.url) === fileURLToPath(new URL(process.argv[1], import.meta.url));
if (isMain) {
  await runPublishAudit(parseArgs(process.argv.slice(2)));
}

export {
  checkRunnerTarballGate,
  DEFAULT_RUNNER_MAX_TARBALL,
  formatMarkdownReport,
} from './format-report.mjs';
