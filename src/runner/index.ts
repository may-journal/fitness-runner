import { readFileSync, existsSync } from 'node:fs';
import { execSync } from 'node:child_process';
import { resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { registry } from '../checks/index.js';
import type { Check, RunContext } from '../types/index.types.js';
import { loadConfig } from '../config/load.js';

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

/** When running semantic-commit, positional(s) may include the message file path (commit-msg hook). */
function getCommitMsgContext(argv: string[], checkName: string | undefined, specFromPositional: boolean): RunContext | undefined {
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
    const byName = new Map(registry.map((c) => [c.name,
      c]));
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
    const mod = await import(url) as { default?: unknown; [k: string]: unknown };
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

/** Builds context with registeredCheckNames and optional staged/commit-msg data. */
function buildContext(staged: RunContext | undefined, commitMsg: RunContext | undefined): RunContext {
  return {
    registeredCheckNames: registry.map((c) => c.name),
    ...(staged ?? {}),
    ...(commitMsg ?? {}),
  };
}

/** Returns checks to run, spec for error display, and optional context from argv. */
async function getChecks(argv: string[], root: string): Promise<{
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
  return { checks, spec, context: buildContext(staged, commitMsg) };
}

/** Logs unknown check/path and exits 1. */
function exitUnknown(spec: string | undefined): never {
  console.error(`Unknown check: ${spec ?? '(none)'}`);
  process.exit(1);
  throw new Error('exit');
}

/** Formats check result meta for logging. */
function formatMeta(result: { meta?: { filesChecked?: number } }, ms: number): string {
  return result.meta?.filesChecked != null
    ? `checked ${result.meta.filesChecked} files in ${ms}ms`
    : `${ms}ms`;
}

/** Formats and logs check errors with check name prefix. */
function displayErrors(checkName: string, errors: string[]): void {
  for (const err of errors) {
    console.error(`✖ [${checkName}] ${err}`);
  }
}

/** Runs a single check; returns true if failed. */
async function runOneCheck(
  check: Check,
  root: string,
  context?: RunContext,
): Promise<boolean> {
  const start = performance.now();
  const result = await check.run(root, context);
  const ms = Math.round(performance.now() - start);
  console.log(`${check.name}: ${formatMeta(result, ms)}`);
  if (!result.ok) {
    displayErrors(check.name, result.errors);
    return true;
  }
  return false;
}

/** Runs all checks; returns true if any failed. */
async function runChecks(
  checks: Check[],
  root: string,
  context?: RunContext,
): Promise<boolean> {
  let failed = false;
  for (const check of checks) {
    if (await runOneCheck(check, root, context)) failed = true;
  }
  return failed;
}

/** Runs fitness checks; exits with 1 on failure. */
export async function run(argv: string[] = process.argv): Promise<void> {
  const root = process.cwd();
  const { checks, spec, context } = await getChecks(argv, root);
  if (!checks.length) exitUnknown(spec);
  const failed = await runChecks(checks, root, context);
  process.exit(failed ? 1 : 0);
}
