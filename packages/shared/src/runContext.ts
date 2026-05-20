import { execSync } from 'node:child_process';
import type { RunContext } from './types/run-context.types.js';

/** Resolved execSync-style function used by CLI checks (testable via context._execSync). */
export type ExecSyncFn = (
  command: string,
  options: { cwd: string; encoding: 'utf8'; maxBuffer: number }
) => string;

/** Staged file paths from context, or [] when absent. */
export function getStagedFiles(context: RunContext | undefined): string[] {
  return context?.stagedFiles ?? [];
}

/** Exec function from context for CLI invocations, or node execSync when absent. */
export function getExecSync(context: RunContext | undefined): ExecSyncFn {
  return context?._execSync ?? execSync;
}
