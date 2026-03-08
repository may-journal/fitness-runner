import { existsSync } from 'node:fs';
import { execSync } from 'node:child_process';
import { join } from 'node:path';
import { buildExecCheckResult } from '../../utils/checkResult.js';
import { execSyncResult } from '../../utils/execSyncResult.js';
import { getExecSync, getStagedFiles } from '../../utils/runContext.js';
import type { ExecSyncFn } from '../../utils/runContext.js';
import { CheckName } from '../../types/index.types.js';
import type { Check } from '../../types/index.types.js';

export const ESLINT_CLI = 'npx eslint';
export const ESLINT_CLI_FORMAT = ' --format json 2>&1';
export const ESLINT_FALLBACK_MESSAGE = `ESLint reported issues. Run: ${ESLINT_CLI} .`;

interface ESLintJsonResult {
  errorCount: number;
  filePath: string;
  messages: Array<{
    column: number;
    line: number;
    message: string;
    ruleId: string | null;
    severity: number;
  }>;
  warningCount: number;
}

/** Run ESLint; returns stdout and exit code. */
export function runEslint(
  root: string,
  paths: string[],
  execSyncFn: ExecSyncFn = execSync
): { exitCode: number; output: string } {
  const args = paths.length > 0 ? paths.map((p) => `"${p.replace(/"/g, '\\"')}"`).join(' ') : '.';
  return execSyncResult(root, `${ESLINT_CLI} ${args}${ESLINT_CLI_FORMAT}`, execSyncFn);
}

/** Format one ESLint message as file:line:col - message (rule). */
export function formatMessage(
  filePath: string,
  msg: { column?: number; line?: number; message: string; ruleId: string | null }
): string {
  const line = msg.line ?? 0;
  const col = msg.column ?? 0;
  const rule = msg.ruleId ? ` (${msg.ruleId})` : '';
  return `${filePath}:${line}:${col} - ${msg.message}${rule}`;
}

/** Parse ESLint JSON output into error lines (file:line:col - message (rule)). */
function parseJsonResults(output: string): { errors: string[]; filesChecked: number } {
  try {
    const data = JSON.parse(output) as ESLintJsonResult[];
    if (!Array.isArray(data)) return { errors: [], filesChecked: 0 };
    const errors = data.flatMap((file) =>
      file.messages.map((msg) => formatMessage(file.filePath, msg))
    );
    return { errors, filesChecked: data.length };
  } catch {
    return { errors: [], filesChecked: 0 };
  }
}

const LINTABLE_EXT = /\.(cjs|js|mjs|tsx?)$/;
const IGNORED_BY_ESLINT = /\.(test|spec)\.(ts|tsx)$/;

/** Paths to lint: staged (existing, lintable, not ignored) under root, or "." when none. */
function getPaths(root: string, staged: string[]): string[] {
  if (staged.length === 0) return [];
  return staged.filter(
    (p) => LINTABLE_EXT.test(p) && !IGNORED_BY_ESLINT.test(p) && existsSync(join(root, p))
  );
}

/** Paths to pass to ESLint: staged existing paths or ['.']. */
function getPathsToLint(root: string, staged: string[]): string[] {
  const paths = getPaths(root, staged);
  return paths.length > 0 ? paths : ['.'];
}

/** Resolve paths and exec fn from root and context. */
function resolveInputs(
  root: string,
  context: Parameters<Check['run']>[1]
): { execFn: ExecSyncFn; paths: string[] } {
  const staged = getStagedFiles(context);
  return { execFn: getExecSync(context), paths: getPathsToLint(root, staged) };
}

/** ESLint check: runs eslint, reports errors from JSON formatter. */
export const eslintCheck: Check = {
  name: CheckName.Eslint,
  async run(root = process.cwd(), context) {
    const { paths, execFn } = resolveInputs(root, context);
    const { output, exitCode } = runEslint(root, paths, execFn);
    const { errors, filesChecked } = parseJsonResults(output);
    return buildExecCheckResult(exitCode, errors, filesChecked, ESLINT_FALLBACK_MESSAGE);
  },
};
