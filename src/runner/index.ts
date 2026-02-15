import { readFileSync } from 'node:fs';
import { execSync } from 'node:child_process';
import { registry } from '../checks/index.js';
import type { Check, RunContext } from '../types/index.js';
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

/** When running semantic-commit, first positional is the message file path (commit-msg hook). */
function getCommitMsgContext(argv: string[], checkName: string | undefined): RunContext | undefined {
  if (checkName !== 'semantic-commit') return undefined;
  const positionals = argv.slice(2).filter((a) => !a.startsWith('-'));
  const path = positionals[0];
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

/** Returns value of --check=<name> from argv if present. */
function getCheckNameFromArg(argv: string[]): string | undefined {
  return argv.find((a) => a.startsWith('--check='))?.slice(8);
}

/** Returns single positional as check name if it matches a registry check. */
function getPositionalCheckName(argv: string[]): string | undefined {
  const positionals = argv.slice(2).filter((a) => !a.startsWith('-'));
  const name = positionals.length === 1 ? positionals[0] : undefined;
  return name && registry.some((r) => r.name === name) ? name : undefined;
}

/** Resolves check name from --check= or single positional that matches a registry check. */
function resolveCheckName(argv: string[]): string | undefined {
  return getCheckNameFromArg(argv) ?? getPositionalCheckName(argv);
}

/** Returns checks to run for a given check name (or all from config when name is undefined). */
function resolveChecksByName(checkName: string | undefined, root: string): Check[] {
  if (!checkName) return resolveChecks(root);
  const one = registry.find((r) => r.name === checkName);
  return one ? [one] : [];
}

/** Returns checks to run, optional check name, and optional context from argv. */
function getChecks(argv: string[], root: string): {
  checks: Check[];
  checkName: string | undefined;
  context: RunContext | undefined;
} {
  const checkName = resolveCheckName(argv);
  const checks = resolveChecksByName(checkName, root);
  const staged = getStagedContext();
  const commitMsg = getCommitMsgContext(argv, checkName);
  const context = (staged ?? commitMsg) ? { ...(staged ?? {}), ...(commitMsg ?? {}) } : undefined;
  return { checks, checkName, context };
}

/** Logs unknown check and exits 1. */
function exitUnknown(checkName: string | undefined): never {
  console.error(`Unknown check: ${checkName ?? '(none)'}`);
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
  const { checks, checkName, context } = getChecks(argv, root);
  if (!checks.length) exitUnknown(checkName);
  const failed = await runChecks(checks, root, context);
  process.exit(failed ? 1 : 0);
}
