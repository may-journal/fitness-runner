import chalk from 'chalk';
import type { Check, RunContext } from '../types/index.types.js';
import { interpolate } from '../utils/interpolate.js';
import { enUS } from './enUS.js';
import { runOneCheck } from './run-execute.js';
import { buildTable, buildTotalLine, type ResultRow } from './run-output.js';
import { getChecks } from './run-resolve.js';

/** Logs unknown check/path and exits 1. */
function exitUnknown(spec: string | undefined): never {
  console.error(chalk.red(enUS.UnknownCheckPrefix + (spec ?? enUS.UnknownCheckSpecNone)));
  process.exit(1);
  throw new Error('exit');
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
