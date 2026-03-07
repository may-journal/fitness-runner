import { existsSync } from 'node:fs';
import { execSync } from 'node:child_process';
import { resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import chalk from 'chalk';
import Table from 'cli-table3';
import { registry } from '../checks/index.js';
import { getColumns } from '../utils/terminal.js';
import type { Check, RunContext } from '../types/index.types.js';
import type { CheckName } from '../types/check-name.js';
import { loadConfig } from '../config/load.js';
import { interpolate } from '../utils/interpolate.js';
import { enUS } from './enUS.js';

/** Returns staged file paths from git for context (always attempted). */
function getStagedContext(): RunContext | undefined {
  try {
    const out = execSync('git diff --cached --name-only', { encoding: 'utf8' });
    return { stagedFiles: out.trim() ? out.trim().split('\n') : [] };
  } catch {
    return undefined;
  }
}

/** Parses args for a contextInline arg (--arg=value or --arg value); returns value and indices to strip. */
function getContextInlineFromArgs(
  args: string[],
  argName: string
): { stripIndices: number[]; value: string } | null {
  const eq = argName + '=';
  for (let i = 0; i < args.length; i++) {
    const hit = tryExactArg(args, argName, i) ?? tryEqualsArg(args, eq, i);
    if (hit) return hit;
  }
  return null;
}

/** Handles --arg value form. */
function tryExactArg(
  args: string[],
  argName: string,
  i: number
): { stripIndices: number[]; value: string } | null {
  if (args[i] !== argName) return null;
  const next = args[i + 1] ?? '';
  const value = next.startsWith('-') ? '' : next;
  const stripIndices = value === next ? [i, i + 1] : [i];
  return { stripIndices, value };
}

/** Handles --arg=value form. */
function tryEqualsArg(
  args: string[],
  eq: string,
  i: number
): { stripIndices: number[]; value: string } | null {
  if (!args[i].startsWith(eq)) return null;
  return { stripIndices: [i], value: args[i].slice(eq.length) };
}

/** Returns args after the check spec when running a single check; strips contextInline arg when registered. */
function getPassthroughArgs(argsAfterSpec: string[], check: Check): string[] {
  const inline = check.contextInline;
  if (!inline) return argsAfterSpec;
  const parsed = getContextInlineFromArgs(argsAfterSpec, inline.argName);
  if (!parsed) return argsAfterSpec;
  return argsAfterSpec.filter((_, i) => !parsed.stripIndices.includes(i));
}

/** Resolves checks from config (if present) or full registry, in order. */
function resolveChecks(root: string): Check[] {
  const config = loadConfig(root);
  if (config?.checks?.length) {
    const byName = new Map(registry.map((c) => [c.name, c]));
    return config.checks
      .map((name) => byName.get(name as CheckName))
      .filter((c): c is Check => c != null);
  }
  return [...registry];
}

/** Returns value of --check=<name-or-path> from argv if present. */
function getCheckSpecFromArg(argv: string[]): string | undefined {
  return argv.find((a) => a.startsWith('--check='))?.slice(8);
}

/** Returns first positional as check name or path. */
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
function getCheckFromModule(mod: { [k: string]: unknown; default?: unknown }): Check | null {
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
    const mod = (await import(url)) as { [k: string]: unknown; default?: unknown };
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

/** Builds context fragment from check.contextInline and args after spec; null if none. */
function getInlineContextFragment(argsAfterSpec: string[], check: Check): RunContext | undefined {
  const inline = check.contextInline;
  if (!inline) return undefined;
  const parsed = getContextInlineFromArgs(argsAfterSpec, inline.argName);
  if (!parsed) return undefined;
  return { [inline.contextKey]: parsed.value } as RunContext;
}

/** Builds context with registeredCheckNames, enabledCheckNames, and optional staged/inline/passthrough data. */
function buildContext(
  staged: RunContext | undefined,
  inlineFragment: RunContext | undefined,
  enabledChecks: Check[],
  passthroughArgs: string[]
): RunContext {
  return {
    enabledCheckNames: enabledChecks.map((c) => c.name),
    registeredCheckNames: registry.map((c) => c.name),
    ...(staged ?? {}),
    ...(inlineFragment ?? {}),
    ...(passthroughArgs.length > 0 ? { passthroughArgs } : {}),
  };
}

/** Returns args after the check spec (for single-check run). */
function getArgsAfterSpec(argv: string[], spec: string | undefined): string[] {
  const specFromPositional = spec !== undefined && getPositionalSpec(argv) === spec;
  return specFromPositional
    ? argv.slice(3)
    : argv.slice(2).filter((a) => !a.startsWith('--check='));
}

/** Returns checks to run, spec for error display, and optional context from argv. */
async function getChecks(
  argv: string[],
  root: string
): Promise<{
  checks: Check[];
  context: RunContext | undefined;
  spec: string | undefined;
}> {
  const spec = resolveCheckSpec(argv);
  const checks = await resolveChecksBySpec(spec, root);
  const staged = getStagedContext();
  const argsAfterSpec = checks.length === 1 ? getArgsAfterSpec(argv, spec) : [];
  const inlineFragment =
    checks.length === 1 ? getInlineContextFragment(argsAfterSpec, checks[0]) : undefined;
  const passthrough = checks.length === 1 ? getPassthroughArgs(argsAfterSpec, checks[0]) : [];
  return { checks, context: buildContext(staged, inlineFragment, checks, passthrough), spec };
}

/** Logs unknown check/path and exits 1. */
function exitUnknown(spec: string | undefined): never {
  console.error(chalk.red(enUS.UnknownCheckPrefix + (spec ?? enUS.UnknownCheckSpecNone)));
  process.exit(1);
  throw new Error('exit');
}

type ResultRow = { errors?: string[]; filesChecked: number; ms: number; name: string; ok: boolean };

/** Formats error block for colspan row. */
function formatErrorsBlock(checkName: string, errors: string[]): string {
  const header = chalk.red(`[${checkName}] ${enUS.PleaseFix}`);
  const bullets = errors.map((e) => chalk.red(enUS.ErrorBullet + e)).join('\n');
  return header + '\n' + bullets;
}

/** Returns green "passed" or red "failed" string. */
function formatStatus(ok: boolean): string {
  return ok ? chalk.green(enUS.StatusPassed) : chalk.red(enUS.StatusFailed);
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
      chalk.bold.white(enUS.TableCheck),
      chalk.bold.white(enUS.TableStatus),
      chalk.bold.white(enUS.TableFiles),
      chalk.bold.white(enUS.TableTime),
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
): Promise<{ errors: string[]; filesChecked: number; ms: number; name: string; ok: boolean }> {
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
  return interpolate(enUS.TotalLine, {
    failure: failureCount,
    files: totalFiles,
    ms: totalMs,
    success: successCount,
  });
}

/** Runs each check and aggregates results plus counts. */
async function collectResults(
  checks: Check[],
  root: string,
  context: RunContext | undefined
): Promise<{ counts: { failure: number; files: number; success: number }; results: ResultRow[] }> {
  const results: ResultRow[] = [];
  const counts = { failure: 0, files: 0, success: 0 };
  for (const check of checks) {
    process.stderr.write(interpolate(enUS.RunningCheck, { name: check.name }) + '\n');
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
  process.stderr.write(enUS.RunningChecks + '\n');
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
  process.stderr.write(enUS.ResolvingChecks + '\n');
  const { checks, spec, context } = await getChecks(argv, root);
  if (!checks.length) exitUnknown(spec);
  const failed = await runChecks(checks, root, context);
  process.exit(failed ? 1 : 0);
}
