import { execSync } from 'node:child_process';
import { registry } from '../checks/index.js';
import type { Check, RunContext } from '../types/index.js';

/** Returns staged file paths from git, or undefined if --staged not used. */
function getStagedContext(argv: string[]): RunContext | undefined {
  if (!argv.includes('--staged')) return undefined;
  try {
    const out = execSync('git diff --cached --name-only', { encoding: 'utf8' });
    return { stagedFiles: out.trim() ? out.trim().split('\n') : [] };
  } catch {
    return { stagedFiles: [] };
  }
}

/** Returns checks to run, optional check name, and optional context from argv. */
function getChecks(argv: string[]): {
  checks: Check[];
  checkName: string | undefined;
  context: RunContext | undefined;
} {
  const checkName = argv.find((a) => a.startsWith('--check='))?.slice(8);
  const checks = checkName ? registry.filter((c) => c.name === checkName) : registry;
  return { checks, checkName, context: getStagedContext(argv) };
}

/** Logs unknown check and exits 1. */
function exitUnknown(checkName: string | undefined): never {
  console.error(`Unknown check: ${checkName ?? '(none)'}`);
  process.exit(1);
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
  const { checks, checkName, context } = getChecks(argv);
  if (!checks.length) exitUnknown(checkName);
  const root = process.cwd();
  const failed = await runChecks(checks, root, context);
  process.exit(failed ? 1 : 0);
}
