import { createRequire } from 'node:module';
import { existsSync } from 'node:fs';
import { join } from 'node:path';
import {
  buildExecCheckResult,
  getFitnessRunnerRoot,
  getStagedFiles,
  resolveLintTsconfig,
} from '@mayjournal/fitness-shared';
import type { Check, CheckName } from '@mayjournal/fitness';

const require = createRequire(import.meta.url);

export const ESLINT_CLI = 'npx eslint';
export const ESLINT_FALLBACK_MESSAGE = `ESLint reported issues. Run: ${ESLINT_CLI} .`;

interface ESLintJsonResult {
  errorCount: number;
  filePath: string;
  messages: Array<{
    column?: number;
    line?: number;
    message: string;
    ruleId: string | null;
    severity?: number;
  }>;
  warningCount: number;
}

/** Run ESLint via Node API using this package's config; cwd is root (parent project). */
export async function runEslintViaAPI(
  root: string,
  paths: string[],
  fitnessRunnerRoot?: string
): Promise<{ errors: string[]; exitCode: number; filesChecked: number }> {
  const frRoot = fitnessRunnerRoot ?? getFitnessRunnerRoot();
  const { createEslintConfig } = require(join(frRoot, 'eslint.base.cjs')) as {
    createEslintConfig: (parserOptions: Record<string, unknown>) => unknown[];
  };
  const { ESLint } = require('eslint') as {
    ESLint: new (opts: Record<string, unknown>) => {
      lintFiles: (p: string[]) => Promise<ESLintJsonResult[]>;
    };
  };
  const tsconfigPath = resolveLintTsconfig(root, frRoot);
  const eslint = new ESLint({
    cwd: root,
    errorOnUnmatchedPattern: false,
    overrideConfig: createEslintConfig({
      project: tsconfigPath,
      tsconfigRootDir: root,
    }),
    overrideConfigFile: true,
  });
  const patterns = paths.length > 0 ? paths : ['.'];
  const results = await eslint.lintFiles(patterns);
  const errors = results.flatMap((r) => r.messages.map((msg) => formatMessage(r.filePath, msg)));
  const exitCode = results.some((r) => r.errorCount > 0) ? 1 : 0;
  return { errors, exitCode, filesChecked: results.length };
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

/** Resolve paths to lint from root and context. */
function resolvePaths(root: string, context: Parameters<Check['run']>[1]): string[] {
  const staged = getStagedFiles(context);
  return getPathsToLint(root, staged);
}

/** Build failed result from thrown value (Error or other). */
function eslintRunCatchResult(err: unknown): {
  errors: string[];
  exitCode: number;
  filesChecked: number;
} {
  const snippet = err instanceof Error ? err.message : String(err);
  return {
    errors: [ESLINT_FALLBACK_MESSAGE, `Output: ${snippet}`],
    exitCode: 1,
    filesChecked: 0,
  };
}

/** Runs test-injected or default ESLint; on throw returns fallback result. */
async function runEslintWithFallback(
  root: string,
  paths: string[],
  context: Parameters<Check['run']>[1]
): Promise<{ errors: string[]; exitCode: number; filesChecked: number }> {
  const fitnessRunnerRoot = context?._fitnessRunnerRootForTesting;
  const run = context?._eslintRunForTesting ?? runEslintViaAPI;
  try {
    return await run(root, paths, fitnessRunnerRoot);
  } catch (err) {
    return eslintRunCatchResult(err);
  }
}

/** ESLint check: runs this package's ESLint config against root (parent project) via Node API. */
export const eslintCheck: Check = {
  name: 'eslint' as CheckName,
  async run(root = process.cwd(), context) {
    const paths = resolvePaths(root, context);
    const { errors, exitCode, filesChecked } = await runEslintWithFallback(root, paths, context);
    return buildExecCheckResult(exitCode, errors, filesChecked, ESLINT_FALLBACK_MESSAGE);
  },
  runInProcess: true,
};

export default eslintCheck;
