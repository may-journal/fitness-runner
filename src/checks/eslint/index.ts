import { existsSync } from 'node:fs';
import { execSync } from 'node:child_process';
import { join } from 'node:path';
import { buildExecCheckResult } from '../../utils/checkResult.js';
import { execSyncResult } from '../../utils/execSyncResult.js';
import { quoteForShell } from '../../utils/shellQuote.js';
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
  const args = paths.length > 0 ? paths.map(quoteForShell).join(' ') : '.';
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

/** Parse string as JSON; return array if valid, else null. */
export function tryParseJsonArray(str: string): ESLintJsonResult[] | null {
  try {
    const data = JSON.parse(str) as unknown;
    return Array.isArray(data) ? data : null;
  } catch {
    return null;
  }
}

/** Try to parse a JSON array from output; tolerates leading/trailing text (e.g. stderr). */
function extractJsonArray(output: string): ESLintJsonResult[] | null {
  const start = output.indexOf('[');
  if (start === -1) return null;
  const end = output.lastIndexOf(']');
  if (end < start) return null;
  return tryParseJsonArray(output.slice(start, end + 1));
}

/** Parse ESLint JSON output into error lines (file:line:col - message (rule)). */
function parseJsonResults(output: string): { errors: string[]; filesChecked: number } {
  let data: ESLintJsonResult[] | null = null;
  try {
    const parsed = JSON.parse(output) as unknown;
    data = Array.isArray(parsed) ? parsed : null;
  } catch {
    data = extractJsonArray(output);
  }
  if (!data) return { errors: [], filesChecked: 0 };
  const errors = data.flatMap((file) =>
    file.messages.map((msg) => formatMessage(file.filePath, msg))
  );
  return { errors, filesChecked: data.length };
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

const FALLBACK_OUTPUT_MAX_LINES = 15;

/** ESLint check: runs eslint, reports errors from JSON formatter. */
export const eslintCheck: Check = {
  name: CheckName.Eslint,
  async run(root = process.cwd(), context) {
    const { paths, execFn } = resolveInputs(root, context);
    const { output, exitCode } = runEslint(root, paths, execFn);
    const { errors, filesChecked } = parseJsonResults(output);
    const useFallback = errors.length === 0 && exitCode !== 0 && output.trim().length > 0;
    const fallbackErrors = useFallback
      ? (() => {
          const lines = output.trim().split(/\r?\n/).slice(0, FALLBACK_OUTPUT_MAX_LINES);
          const snippet = lines.length > 1 ? `Output:\n${lines.join('\n')}` : `Output: ${lines[0]}`;
          return [ESLINT_FALLBACK_MESSAGE, snippet];
        })()
      : errors;
    return buildExecCheckResult(exitCode, fallbackErrors, filesChecked, ESLINT_FALLBACK_MESSAGE);
  },
};
