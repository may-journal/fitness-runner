import { existsSync } from 'node:fs';
import { execSync } from 'node:child_process';
import { join } from 'node:path';
import type { Check } from '../../types/index.types.js';

export const PRETTIER_CLI = 'npx prettier';
export const PRETTIER_FALLBACK_MESSAGE = `Prettier reported issues. Run: ${PRETTIER_CLI} . --write`;

const PRETTIER_CONFIG_NAMES = [
  '.prettierrc',
  '.prettierrc.json',
  '.prettierrc.yml',
  '.prettierrc.yaml',
  '.prettierrc.js',
  '.prettierrc.cjs',
  'prettier.config.js',
  'prettier.config.cjs',
  'prettier.config.mjs',
];

const EXEC_OPTS = { encoding: 'utf8' as const, maxBuffer: 1024 * 1024 };
type ExecSyncFn = (
  cmd: string,
  opts: { encoding: 'utf8'; cwd: string; maxBuffer: number }
) => string;

/** Returns true if root has a Prettier config file. */
export function hasPrettierConfig(root: string): boolean {
  return PRETTIER_CONFIG_NAMES.some((name) => existsSync(join(root, name)));
}

/** Run Prettier --check; returns stdout+stderr and exit code. */
export function runPrettierCheck(
  root: string,
  paths: string[],
  execSyncFn: ExecSyncFn = execSync
): { output: string; exitCode: number } {
  const args = paths.length > 0 ? paths.map((p) => `"${p.replace(/"/g, '\\"')}"`).join(' ') : '.';
  try {
    const out = execSyncFn(`${PRETTIER_CLI} --check ${args} 2>&1`, { ...EXEC_OPTS, cwd: root });
    return { exitCode: 0, output: out };
  } catch (e: unknown) {
    const err = e as { stdout?: string; stderr?: string; status?: number };
    const out = [err.stdout, err.stderr].filter(Boolean).join('\n');
    return { exitCode: typeof err.status === 'number' ? err.status : 1, output: out };
  }
}

/** Parse Prettier output for [warn] file paths. */
export function parsePrettierOutput(output: string): string[] {
  return output
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line.startsWith('[warn] ') && !line.includes('Code style issues'))
    .map((line) => line.slice(7).trim());
}

/** Paths to check: staged (existing) under root, or ["."] when none. */
function getPathsToCheck(root: string, staged: string[]): string[] {
  if (staged.length === 0) return ['.'];
  return staged.filter((p) => existsSync(join(root, p)));
}

/** Resolve paths and exec fn from root and context. */
function resolveInputs(
  root: string,
  context: { stagedFiles?: string[]; _execSync?: ExecSyncFn } | undefined
): { paths: string[]; execFn: ExecSyncFn } {
  const staged = context?.stagedFiles ?? [];
  return { execFn: context?._execSync ?? execSync, paths: getPathsToCheck(root, staged) };
}

/** Compute filesChecked from errors and paths. */
function getFilesChecked(errors: string[], paths: string[]): number {
  if (errors.length > 0) return errors.length;
  return paths.length === 1 && paths[0] === '.' ? 0 : paths.length;
}

/** Build CheckResult from exit code, parsed errors, and filesChecked. */
function buildResult(
  exitCode: number,
  errors: string[],
  filesChecked: number
): { ok: boolean; errors: string[]; meta: { filesChecked: number } } {
  const ok = exitCode === 0 && errors.length === 0;
  const fallback = !ok && errors.length === 0 ? [PRETTIER_FALLBACK_MESSAGE] : [];
  return { errors: errors.length > 0 ? errors : fallback, meta: { filesChecked }, ok };
}

/** Prettier check: runs prettier --check; skips when no config. */
export const prettierCheck: Check = {
  name: 'prettier',
  async run(root = process.cwd(), context) {
    if (!hasPrettierConfig(root)) return { errors: [], meta: { filesChecked: 0 }, ok: true };
    const { paths, execFn } = resolveInputs(root, context);
    const { output, exitCode } = runPrettierCheck(root, paths, execFn);
    const errors = parsePrettierOutput(output);
    return buildResult(exitCode, errors, getFilesChecked(errors, paths));
  },
};
