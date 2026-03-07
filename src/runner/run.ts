import { existsSync } from 'node:fs';
import { execSync } from 'node:child_process';
import { join, dirname, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { Worker } from 'node:worker_threads';
import chalk from 'chalk';
import Table from 'cli-table3';
import { registry } from '../checks/index.js';
import { CheckName } from '../types/check-name.js';
import { getColumns } from '../utils/terminal.js';
import type { Check, RunContext } from '../types/index.types.js';
import { loadConfig } from '../config/load.js';
import { getContextInlineFromArgs } from '../utils/contextInlineFromArgs.js';
import { interpolate } from '../utils/interpolate.js';
import { enUS } from './enUS.js';

/** Staged paths under node_modules are never passed to checks. */
function stripNodeModulesFromStaged(paths: string[]): string[] {
  return paths.filter((p) => !p.includes('node_modules'));
}

/** Returns staged file paths from git for context (always attempted); excludes node_modules. */
function getStagedContext(): RunContext | undefined {
  try {
    const out = execSync('git diff --cached --name-only', { encoding: 'utf8' });
    const raw = out.trim() ? out.trim().split('\n') : [];
    return { stagedFiles: stripNodeModulesFromStaged(raw) };
  } catch {
    return undefined;
  }
}

/** Returns args after the check spec when running a single check; strips contextInline arg when registered. */
function getPassthroughArgs(argsAfterSpec: string[], check: Check): string[] {
  const inline = check.contextInline;
  if (!inline) return argsAfterSpec;
  const parsed = getContextInlineFromArgs(argsAfterSpec, inline.argName);
  if (!parsed) return argsAfterSpec;
  return argsAfterSpec.filter((_, i) => !parsed.stripIndices.includes(i));
}

/** Resolves checks from config.checks list when present; else null. */
function checksFromConfigList(config: ReturnType<typeof loadConfig>): Check[] | null {
  if (!config?.checks?.length) return null;
  const byName = new Map(registry.map((c) => [c.name, c]));
  return config.checks.map((name) => byName.get(name)).filter((c): c is Check => c != null);
}

/** Resolves checks from registry, optionally excluding disabledChecks. */
function registryMinusDisabled(config: ReturnType<typeof loadConfig>): Check[] {
  const disabled = new Set(config?.disabledChecks ?? []);
  return disabled.size ? registry.filter((c) => !disabled.has(c.name)) : [...registry];
}

/** Resolves checks from config (if present) or full registry, in order. */
function resolveChecks(root: string): Check[] {
  const config = loadConfig(root);
  const fromList = checksFromConfigList(config);
  if (fromList != null) return fromList;
  return registryMinusDisabled(config);
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

const CHECK_TIMEOUT_MS = 5000;

/** Resolves check timeout ms; tests may pass _checkTimeoutMsForTesting for fast timeout tests. */
function getCheckTimeoutMs(context?: RunContext): number {
  return context?._checkTimeoutMsForTesting ?? CHECK_TIMEOUT_MS;
}

/** Rejects after ms; used for in-process path-based checks that may hang async. */
function timeoutAfter(ms: number): Promise<never> {
  return new Promise((_, reject) => setTimeout(() => reject(new Error('timeout')), ms));
}

type WorkerReply =
  | { ms: number; result: { errors: string[]; meta?: { filesChecked?: number }; ok: boolean } }
  | { error: string; ms: number };

/** Row shape returned by runOneCheck / worker. */
type ResultRowLike = {
  errors: string[];
  filesChecked: number;
  ms: number;
  name: string;
  ok: boolean;
};

/** Formats thrown value for in-process check (timeout vs other). */
function formatInProcessError(err: unknown, timeoutMs: number = CHECK_TIMEOUT_MS): string {
  if (err instanceof Error && err.message === 'timeout') {
    return interpolate(enUS.CheckTimeout, { seconds: timeoutMs / 1000 });
  }
  return err instanceof Error ? err.message : String(err);
}

/** Runs check in main thread (read-repo-first, vitest-coverage-full, path-based, or when VITEST). */
async function runOneCheckInProcess(
  check: Check,
  root: string,
  context?: RunContext
): Promise<ResultRowLike> {
  const start = performance.now();
  const timeoutMs = getCheckTimeoutMs(context);
  const runPromise =
    check.name === CheckName.ReadRepoFirst
      ? check.run(root, context)
      : Promise.race([
          Promise.resolve().then(() => check.run(root, context)),
          timeoutAfter(timeoutMs),
        ]);
  try {
    const result = await runPromise;
    const ms = Math.round(performance.now() - start);
    const filesChecked = result.meta?.filesChecked ?? -1;
    return { errors: result.errors, filesChecked, ms, name: check.name, ok: result.ok };
  } catch (err: unknown) {
    const ms = Math.round(performance.now() - start);
    return {
      errors: [formatInProcessError(err, timeoutMs)],
      filesChecked: -1,
      ms,
      name: check.name,
      ok: false,
    };
  }
}

/** Runs check in worker thread; main thread terminates worker after CHECK_TIMEOUT_MS. */
function runOneCheckInWorker(
  check: Check,
  root: string,
  context?: RunContext
): Promise<ResultRowLike> {
  const workerDir = dirname(fileURLToPath(import.meta.url));
  const workerNextToRun = resolve(workerDir, 'run-one-check-worker.js');
  const workerPath = existsSync(workerNextToRun)
    ? workerNextToRun
    : resolve(process.cwd(), 'dist/runner/run-one-check-worker.js');
  const worker = new Worker(workerPath, {
    type: 'module',
    workerData: { checkName: check.name, context, root },
  } as import('node:worker_threads').WorkerOptions);
  const timeoutMs = getCheckTimeoutMs(context);
  return new Promise((resolve) => {
    let settled = false;
    const timeoutMsg = interpolate(enUS.CheckTimeout, { seconds: timeoutMs / 1000 });
    const timeoutId = setTimeout(() => {
      /* v8 ignore start - defensive; timeout is cleared on message/error so settled is never true here */
      if (settled) return;
      /* v8 ignore stop */
      settled = true;
      worker.terminate();
      resolve({
        errors: [timeoutMsg],
        filesChecked: -1,
        ms: timeoutMs,
        name: check.name,
        ok: false,
      });
    }, timeoutMs);
    worker.on('message', (msg: WorkerReply) => {
      if (settled) return;
      settled = true;
      clearTimeout(timeoutId);
      if ('error' in msg) {
        resolve({ errors: [msg.error], filesChecked: -1, ms: msg.ms, name: check.name, ok: false });
      } else {
        const filesChecked = msg.result.meta?.filesChecked ?? -1;
        resolve({
          errors: msg.result.errors,
          filesChecked,
          ms: msg.ms,
          name: check.name,
          ok: msg.result.ok,
        });
      }
    });
    worker.on('error', (err: Error) => {
      if (settled) return;
      settled = true;
      clearTimeout(timeoutId);
      resolve({ errors: [err.message], filesChecked: -1, ms: 0, name: check.name, ok: false });
    });
  });
}

/** Runs a single check; registry checks use worker + 5s terminate; checks with runInProcess, path-loaded, and tests run in-process. */
async function runOneCheck(
  check: Check,
  root: string,
  context?: RunContext
): Promise<ResultRowLike> {
  const runInProcess =
    check.runInProcess === true || !registry.includes(check) || process.env.VITEST === 'true';
  return runInProcess
    ? runOneCheckInProcess(check, root, context)
    : runOneCheckInWorker(check, root, context);
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
export async function run(
  argv: string[] = process.argv,
  testOverrides?: Partial<RunContext>
): Promise<void> {
  const root = process.cwd();
  process.stderr.write(enUS.ResolvingChecks + '\n');
  const { checks, spec, context } = await getChecks(argv, root);
  if (!checks.length) exitUnknown(spec);
  const mergedContext = testOverrides ? { ...context, ...testOverrides } : context;
  const failed = await runChecks(checks, root, mergedContext);
  process.exit(failed ? 1 : 0);
}
