import { existsSync, readFileSync } from 'node:fs';
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

/** Returns true if package.json has a "prettier" field (string or object). */
function hasPrettierInPackageJson(root: string): boolean {
  const pkgPath = join(root, 'package.json');
  if (!existsSync(pkgPath)) return false;
  try {
    const pkg = JSON.parse(readFileSync(pkgPath, 'utf8')) as { prettier?: unknown };
    return pkg.prettier != null;
  } catch {
    return false;
  }
}

/** Returns true if root has a Prettier config file or package.json "prettier" field. */
export function hasPrettierConfig(root: string): boolean {
  return (
    PRETTIER_CONFIG_NAMES.some((name) => existsSync(join(root, name))) ||
    hasPrettierInPackageJson(root)
  );
}

/** Quote a single arg for shell. */
function quoteArg(p: string): string {
  return `"${p.replace(/"/g, '\\"')}"`;
}

/** Build args string from passthrough or paths. */
function buildPrettierArgs(passthroughArgs: string[] | undefined, paths: string[]): string {
  if ((passthroughArgs?.length ?? 0) > 0) return passthroughArgs!.map(quoteArg).join(' ');
  return paths.length > 0 ? paths.map(quoteArg).join(' ') : '.';
}

/** Extract output and exitCode from exec error. */
function parseExecError(e: unknown): { output: string; exitCode: number } {
  const err = e as { stdout?: string; stderr?: string; status?: number };
  const output = [err.stdout, err.stderr].filter(Boolean).join('\n');
  const exitCode = typeof err.status === 'number' ? err.status : 1;
  return { exitCode, output };
}

/** Build Prettier CLI command. */
function buildPrettierCmd(args: string, usePassthrough: boolean): string {
  const mode = usePassthrough ? '' : ' --check';
  return `${PRETTIER_CLI}${mode} ${args} 2>&1`;
}

/** Run Prettier with given args (or --check + paths by default); returns stdout+stderr and exit code. */
export function runPrettier(
  root: string,
  paths: string[],
  execSyncFn: ExecSyncFn = execSync,
  passthroughArgs?: string[]
): { output: string; exitCode: number } {
  const usePassthrough = (passthroughArgs?.length ?? 0) > 0;
  const args = buildPrettierArgs(passthroughArgs, paths);
  const cmd = buildPrettierCmd(args, usePassthrough);
  try {
    const output = execSyncFn(cmd, { ...EXEC_OPTS, cwd: root });
    return { exitCode: 0, output };
  } catch (e: unknown) {
    return parseExecError(e);
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

type PrettierContext = {
  stagedFiles?: string[];
  passthroughArgs?: string[];
  _execSync?: ExecSyncFn;
};

/** Resolve staged from context. */
function getStaged(context: PrettierContext | undefined): string[] {
  return context?.stagedFiles ?? [];
}

/** Resolve paths, exec fn, and passthrough from root and context. */
function resolveInputs(
  root: string,
  context: PrettierContext | undefined
): { paths: string[]; execFn: ExecSyncFn; passthroughArgs?: string[] } {
  const staged = getStaged(context);
  const paths = getPathsToCheck(root, staged);
  const execFn = (context && context._execSync) ?? execSync;
  const passthroughArgs = context?.passthroughArgs;
  return { execFn, passthroughArgs, paths };
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

/** Prettier check: runs prettier --check (or passthrough args); skips when no config. */
export const prettierCheck: Check = {
  name: 'prettier',
  async run(root = process.cwd(), context) {
    if (!hasPrettierConfig(root)) return { errors: [], meta: { filesChecked: 0 }, ok: true };
    const { paths, execFn, passthroughArgs } = resolveInputs(root, context);
    const { output, exitCode } = runPrettier(root, paths, execFn, passthroughArgs);
    const errors = parsePrettierOutput(output);
    return buildResult(exitCode, errors, getFilesChecked(errors, paths));
  },
};
