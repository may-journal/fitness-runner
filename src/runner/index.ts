import { readFileSync, existsSync } from 'node:fs';
import { execSync } from 'node:child_process';
import { resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import chalk from 'chalk';
import Table from 'cli-table3';
import { registry } from '../checks/index.js';
import { getColumns } from '../utils/terminal.js';
import type { Check, RunContext } from '../types/index.types.js';
import { loadConfig } from '../config/load.js';

export const UNKNOWN_CHECK_PREFIX = 'Unknown check: ';
export const UNKNOWN_CHECK_SPEC_NONE = '(none)';
export const PLEASE_FIX_ITEMS = 'Please fix these items:';
export const ERROR_BULLET = '  ✖ ';

/** Returns staged file paths from git for context (always attempted). */
function getStagedContext(): RunContext | undefined {
  try {
    const out = execSync('git diff --cached --name-only', { encoding: 'utf8' });
    return { stagedFiles: out.trim() ? out.trim().split('\n') : [] };
  } catch {
    return undefined;
  }
}

/** Resolves commit-msg file path from positionals (second when spec was positional, first when from --check=). */
function getCommitMsgPath(positionals: string[], specFromPositional: boolean): string | undefined {
  if (specFromPositional && positionals.length >= 2) return positionals[1];
  if (!specFromPositional && positionals.length >= 1) return positionals[0];
  return undefined;
}

/** Returns args after the check spec when running a single check (excludes commit-msg path for semantic-commit). */
function getPassthroughArgs(
  argv: string[],
  checkName: string | undefined,
  specFromPositional: boolean
): string[] {
  const raw = specFromPositional
    ? argv.slice(3)
    : argv.slice(2).filter((a) => !a.startsWith('--check='));
  if (checkName !== 'semantic-commit') return raw;
  const positionals = argv.slice(2).filter((a) => !a.startsWith('-'));
  const commitPath = getCommitMsgPath(positionals, specFromPositional);
  return commitPath ? raw.filter((a) => a !== commitPath) : raw;
}

/** When running semantic-commit, positional(s) may include the message file path (commit-msg hook). */
function getCommitMsgContext(
  argv: string[],
  checkName: string | undefined,
  specFromPositional: boolean
): RunContext | undefined {
  if (checkName !== 'semantic-commit') return undefined;
  const positionals = argv.slice(2).filter((a) => !a.startsWith('-'));
  const path = getCommitMsgPath(positionals, specFromPositional);
  if (!path) return undefined;
  try {
    const content = readFileSync(path, 'utf8');
    const subject = content.split('\n')[0] || '';
    return { proposedCommitMessage: subject };
  } catch {
    return { proposedCommitMessage: '' };
  }
}

/** Resolves checks from config (if present) or full registry, in order. */
function resolveChecks(root: string): Check[] {
  const config = loadConfig(root);
  if (config?.checks?.length) {
    const byName = new Map(registry.map((c) => [c.name, c]));
    return config.checks.map((name) => byName.get(name)).filter((c): c is Check => c != null);
  }
  return [...registry];
}

/** Returns value of --check=<name-or-path> from argv if present. */
function getCheckSpecFromArg(argv: string[]): string | undefined {
  return argv.find((a) => a.startsWith('--check='))?.slice(8);
}

/** Returns first positional as check name or path (so second positional can be commit-msg path). */
function getPositionalSpec(argv: string[]): string | undefined {
  const positionals = argv.slice(2).filter((a) => !a.startsWith('-'));
  return positionals.length >= 1 ? positionals[0] : undefined;
}

/** Resolves check spec (name or path) from --check= or single positional. */
function resolveCheckSpec(argv: string[]): string | undefined {
  return getCheckSpecFromArg(argv) ?? getPositionalSpec(argv);
}

/** True if spec looks like a file path (for loading a check module). */
function isPathSpec(spec: string): boolean {
  return /[/\\]/.test(spec) || /\.(?:js|mjs|cjs|ts)$/i.test(spec);
}

/** True if value looks like a Check. */
function isCheckLike(v: unknown): v is Check {
  return !!v && typeof (v as Check).run === 'function' && (v as Check).name != null;
}

/** Returns default or first Check-like export from module. */
function getCheckFromModule(mod: { default?: unknown; [k: string]: unknown }): Check | null {
  if (isCheckLike(mod.default)) return mod.default;
  for (const v of Object.values(mod)) if (isCheckLike(v)) return v;
  return null;
}

/** Loads a Check from a module path (default or first Check-like export). */
async function loadCheckFromPath(root: string, spec: string): Promise<Check | null> {
  const abs = resolve(root, spec);
  if (!existsSync(abs)) return null;
  try {
    const url = pathToFileURL(abs).href;
    const mod = (await import(url)) as { default?: unknown; [k: string]: unknown };
    return getCheckFromModule(mod);
  } catch {
    return null;
  }
}

/** Returns checks to run for a given spec (name or path), or all from config when spec is undefined. */
async function resolveChecksBySpec(spec: string | undefined, root: string): Promise<Check[]> {
  if (!spec) return resolveChecks(root);
  if (isPathSpec(spec)) {
    const check = await loadCheckFromPath(root, spec);
    return check ? [check] : [];
  }
  const one = registry.find((r) => r.name === spec);
  return one ? [one] : [];
}

/** Builds context with registeredCheckNames, enabledCheckNames, and optional staged/commit-msg/passthrough data. */
function buildContext(
  staged: RunContext | undefined,
  commitMsg: RunContext | undefined,
  enabledChecks: Check[],
  passthroughArgs: string[]
): RunContext {
  return {
    enabledCheckNames: enabledChecks.map((c) => c.name),
    registeredCheckNames: registry.map((c) => c.name),
    ...(staged ?? {}),
    ...(commitMsg ?? {}),
    ...(passthroughArgs.length > 0 ? { passthroughArgs } : {}),
  };
}

/** Returns checks to run, spec for error display, and optional context from argv. */
async function getChecks(
  argv: string[],
  root: string
): Promise<{
  checks: Check[];
  spec: string | undefined;
  context: RunContext | undefined;
}> {
  const spec = resolveCheckSpec(argv);
  const checks = await resolveChecksBySpec(spec, root);
  const checkName = checks.length === 1 ? checks[0].name : undefined;
  const specFromPositional = spec !== undefined && getPositionalSpec(argv) === spec;
  const staged = getStagedContext();
  const commitMsg = getCommitMsgContext(argv, checkName, specFromPositional);
  const passthrough =
    checks.length === 1 ? getPassthroughArgs(argv, checkName, specFromPositional) : [];
  return { checks, context: buildContext(staged, commitMsg, checks, passthrough), spec };
}

/** Logs unknown check/path and exits 1. */
function exitUnknown(spec: string | undefined): never {
  console.error(chalk.red(UNKNOWN_CHECK_PREFIX + (spec ?? UNKNOWN_CHECK_SPEC_NONE)));
  process.exit(1);
  throw new Error('exit');
}

type ResultRow = { name: string; ok: boolean; filesChecked: number; ms: number; errors?: string[] };

/** Formats error block for colspan row. */
function formatErrorsBlock(checkName: string, errors: string[]): string {
  const header = chalk.red(`[${checkName}] ${PLEASE_FIX_ITEMS}`);
  const bullets = errors.map((e) => chalk.red(ERROR_BULLET + e)).join('\n');
  return header + '\n' + bullets;
}

/** Returns green "passed" or red "failed" string. */
function formatStatus(ok: boolean): string {
  return ok ? chalk.green('passed') : chalk.red('failed');
}

/** Pushes colspan row with formatted errors to table. */
function pushErrorRow(table: InstanceType<typeof Table>, name: string, errors: string[]): void {
  table.push([{ colSpan: 4, content: formatErrorsBlock(name, errors), wordWrap: true }]);
}

/** Pushes one check row and optional error colspan row to table. */
function pushResultRow(table: InstanceType<typeof Table>, r: ResultRow): void {
  const files = r.filesChecked >= 0 ? String(r.filesChecked) : '-';
  table.push([r.name, formatStatus(r.ok), files, `${r.ms}ms`]);
  if (!r.ok && r.errors?.length) pushErrorRow(table, r.name, r.errors);
}

/** Builds table: check rows plus full-width colspan row per failed check with errors. */
function buildTable(rows: ResultRow[]): string {
  const timeCol = Math.max(10, getColumns() - 51);
  const table = new Table({
    colWidths: [28, 10, 8, timeCol],
    head: [
      chalk.bold.white('Check'),
      chalk.bold.white('Status'),
      chalk.bold.white('Files'),
      chalk.bold.white('Time'),
    ],
    wordWrap: true,
  });
  for (const r of rows) pushResultRow(table, r);
  return table.toString();
}

/** Runs a single check; returns result data for table and errors. */
async function runOneCheck(
  check: Check,
  root: string,
  context?: RunContext
): Promise<{ name: string; ok: boolean; filesChecked: number; ms: number; errors: string[] }> {
  const start = performance.now();
  const result = await check.run(root, context);
  const ms = Math.round(performance.now() - start);
  const filesChecked = result.meta?.filesChecked ?? -1;
  return { errors: result.errors, filesChecked, ms, name: check.name, ok: result.ok };
}

/** Builds total summary line. */
function buildTotalLine(
  successCount: number,
  failureCount: number,
  totalFiles: number,
  totalMs: number
): string {
  return `Total: ${successCount} succeeded, ${failureCount} failed, ${totalFiles} files in ${totalMs}ms`;
}

/** Runs each check and aggregates results plus counts. */
async function collectResults(
  checks: Check[],
  root: string,
  context: RunContext | undefined
): Promise<{ results: ResultRow[]; counts: { success: number; failure: number; files: number } }> {
  const results: ResultRow[] = [];
  const counts = { failure: 0, files: 0, success: 0 };
  for (const check of checks) {
    const { name, ok, filesChecked, ms, errors } = await runOneCheck(check, root, context);
    results.push({ errors, filesChecked, ms, name, ok });
    if (ok) counts.success++;
    else counts.failure++;
    counts.files += filesChecked >= 0 ? filesChecked : 0;
  }
  return { counts, results };
}

/** Runs all checks; returns true if any failed. */
async function runChecks(checks: Check[], root: string, context?: RunContext): Promise<boolean> {
  const start = performance.now();
  const { results, counts } = await collectResults(checks, root, context);
  const out = counts.failure > 0 ? console.error : console.log;
  out(buildTable(results));
  const totalMs = Math.round(performance.now() - start);
  const totalLine = buildTotalLine(counts.success, counts.failure, counts.files, totalMs);
  out(counts.failure > 0 ? chalk.bold.red(totalLine) : chalk.bold.green(totalLine));
  return counts.failure > 0;
}

/** Runs fitness checks; exits with 1 on failure. */
export async function run(argv: string[] = process.argv): Promise<void> {
  const root = process.cwd();
  const { checks, spec, context } = await getChecks(argv, root);
  if (!checks.length) exitUnknown(spec);
  const failed = await runChecks(checks, root, context);
  process.exit(failed ? 1 : 0);
}
