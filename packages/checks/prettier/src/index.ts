import { existsSync, readFileSync } from 'node:fs';
import { execSync } from 'node:child_process';
import { join } from 'node:path';
import {
  buildExecCheckResult,
  execSyncResult,
  getFitnessRunnerRoot,
  getExecSync,
  getStagedFiles,
  quoteForShell,
  resolveFitnessConfigPath,
} from '@mayjournal/fitness-shared';
import type { ExecSyncFn } from '@mayjournal/fitness-shared';
import { CheckName } from '@mayjournal/fitness';
import type { Check, RunContext } from '@mayjournal/fitness';

export const PRETTIER_CLI = 'npx prettier';
export const PRETTIER_FALLBACK_MESSAGE = `Prettier reported issues. Run: ${PRETTIER_CLI} . --write`;

const PRETTIER_CONFIG_NAMES = [
  '.prettierrc',
  '.prettierrc.json',
  '.prettierrc.yml',
  '.prettierrc.yaml',
  '.prettierrc.js',
  '.prettierrc.cjs',
  '.prettierrc.mts',
  '.prettierrc.cts',
  '.prettierrc.ts',
  'prettier.config.js',
  'prettier.config.cjs',
  'prettier.config.mjs',
  'prettier.config.mts',
  'prettier.config.cts',
  'prettier.config.ts',
];

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

/** Build args string from passthrough or paths. */
function buildPrettierArgs(passthroughArgs: string[] | undefined, paths: string[]): string {
  if ((passthroughArgs?.length ?? 0) > 0) return passthroughArgs!.map(quoteForShell).join(' ');
  return paths.length > 0 ? paths.map(quoteForShell).join(' ') : '.';
}

/** Build Prettier CLI command. */
function buildPrettierCmd(args: string, usePassthrough: boolean, configPath?: string): string {
  const mode = usePassthrough ? '' : ' --check';
  const config = configPath ? ` --config ${quoteForShell(configPath)}` : '';
  return `${PRETTIER_CLI}${config}${mode} ${args} 2>&1`;
}

/** Resolve Prettier config: consumer config when present, else shared prettier.config.cjs. */
function resolvePrettierConfigPath(root: string): string {
  if (hasPrettierConfig(root)) return '';
  return resolveFitnessConfigPath(root, 'prettier.config.cjs', getFitnessRunnerRoot());
}

/** Run Prettier with given args (or --check + paths by default); returns stdout+stderr and exit code. */
export function runPrettier(
  root: string,
  paths: string[],
  execSyncFn: ExecSyncFn = execSync,
  passthroughArgs?: string[],
  configPath?: string
): { exitCode: number; output: string } {
  const usePassthrough = (passthroughArgs?.length ?? 0) > 0;
  const args = buildPrettierArgs(passthroughArgs, paths);
  const cmd = buildPrettierCmd(args, usePassthrough, configPath);
  return execSyncResult(root, cmd, execSyncFn);
}

/** Parse Prettier output for [warn] file paths. */
export function parsePrettierOutput(output: string): string[] {
  return output
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line.startsWith('[warn] ') && !line.includes('Code style issues'))
    .map((line) => line.slice(7).trim());
}

/** Paths to skip when passing staged files to Prettier (no parser or ignore-file). */
const PRETTIER_SKIP_STAGED = new Set([
  '.gitignore',
  'githooks/commit-msg',
  '.npmrc',
  '.prettierignore',
  'LICENSE',
  'package-lock.json',
]);

/** Paths to check: staged (existing) under root, or ["."] when none; excludes skip list, scripts/, githooks. */
function getPathsToCheck(root: string, staged: string[]): string[] {
  if (staged.length === 0) return ['.'];
  return staged.filter(
    (p) =>
      existsSync(join(root, p)) &&
      !PRETTIER_SKIP_STAGED.has(p) &&
      !p.endsWith('.mdc') &&
      !p.includes('githooks/') &&
      !p.startsWith('scripts/')
  );
}

/** Resolve paths, exec fn, and passthrough from root and context. */
function resolveInputs(
  root: string,
  context: RunContext | undefined
): { execFn: ExecSyncFn; passthroughArgs?: string[]; paths: string[] } {
  const staged = getStagedFiles(context);
  const paths = getPathsToCheck(root, staged);
  return {
    execFn: getExecSync(context),
    passthroughArgs: context?.passthroughArgs,
    paths,
  };
}

/** Compute filesChecked from errors and paths. */
function getFilesChecked(errors: string[], paths: string[]): number {
  if (errors.length > 0) return errors.length;
  return paths.length === 1 && paths[0] === '.' ? 0 : paths.length;
}

/** Prettier check: runs prettier --check (or passthrough args) using shared config when consumer has none. */
const prettierCheck: Check = {
  name: CheckName.Prettier,
  async run(root = process.cwd(), context) {
    const configPath = resolvePrettierConfigPath(root);
    const { paths, execFn, passthroughArgs } = resolveInputs(root, context);
    const { output, exitCode } = runPrettier(root, paths, execFn, passthroughArgs, configPath);
    const errors = parsePrettierOutput(output);
    return buildExecCheckResult(
      exitCode,
      errors,
      getFilesChecked(errors, paths),
      PRETTIER_FALLBACK_MESSAGE
    );
  },
};

export { prettierCheck };
export default prettierCheck;
