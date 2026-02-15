import { existsSync } from 'node:fs';
import { execSync } from 'node:child_process';
import { join } from 'node:path';
import type { Check } from '../../types/index.js';

const CSPELL_ISSUE_RE = /^(.+):(\d+):(\d+)\s+-\s+(.+)$/m;
const FILES_CHECKED_RE = /Files checked:\s*(\d+)/;

const EXEC_OPTS = { encoding: 'utf8' as const, cwd: '', maxBuffer: 1024 * 1024 };
type ExecSyncFn = (cmd: string, opts: typeof EXEC_OPTS) => string;

/** Run cspell; output is stdout + stderr (2>&1) so we can parse issues and filesChecked. */
export function runCspell(
  root: string,
  paths: string[],
  execSyncFn: ExecSyncFn = execSync,
): { output: string; exitCode: number } {
  if (paths.length === 0) {
    return { output: '', exitCode: 0 };
  }
  const list = paths.map((p) => `"${p.replace(/"/g, '\\"')}"`).join(' ');
  try {
    const out = execSyncFn(`npx cspell --no-progress ${list} 2>&1`, { ...EXEC_OPTS, cwd: root });
    return { output: out, exitCode: 0 };
  } catch (e: unknown) {
    const err = e as { stdout?: string; stderr?: string; status?: number };
    const out = [err.stdout,
      err.stderr].filter(Boolean).join('\n');
    return { output: out, exitCode: typeof err.status === 'number' ? err.status : 1 };
  }
}

/** Parse cspell output into error lines (file:line:col - message). */
function parseIssues(output: string): string[] {
  const lines = output.trim().split('\n').filter(Boolean);
  return lines.filter((line) => CSPELL_ISSUE_RE.test(line));
}

/** Extract "Files checked: N" from cspell output. */
function parseFilesChecked(output: string): number | undefined {
  const m = output.match(FILES_CHECKED_RE);
  return m ? parseInt(m[1], 10) : undefined;
}

/** Run cspell with a glob; output is stdout + stderr. */
function runCspellGlob(
  root: string,
  glob: string,
  execSyncFn: ExecSyncFn = execSync,
): { output: string; exitCode: number } {
  try {
    const out = execSyncFn(`npx cspell --no-progress "${glob.replace(/"/g, '\\"')}" 2>&1`, {
      ...EXEC_OPTS,
      cwd: root,
    });
    return { output: out, exitCode: 0 };
  } catch (e: unknown) {
    const err = e as { stdout?: string; stderr?: string; status?: number };
    const out = [err.stdout,
      err.stderr].filter(Boolean).join('\n');
    return { output: out, exitCode: typeof err.status === 'number' ? err.status : 1 };
  }
}

/** Run cspell on staged paths; returns empty output if no valid paths. */
function runCspellStaged(
  root: string,
  stagedFiles: string[],
  execSyncFn: ExecSyncFn,
): { output: string; exitCode: number } {
  const paths = stagedFiles.filter((p) => existsSync(join(root, p)));
  if (paths.length === 0) return { output: '', exitCode: 0 };
  return runCspell(root, paths.map((p) => join(root, p)), execSyncFn);
}

/** Builds check result from cspell output and exit code. */
function buildCspellResult(
  output: string,
  exitCode: number,
): { ok: boolean; errors: string[]; meta: { filesChecked: number } } {
  const issues = parseIssues(output);
  const filesChecked = parseFilesChecked(output) ?? 0;
  const ok = exitCode === 0 && issues.length === 0;
  const errors =
    issues.length > 0 ? issues : !ok ? ['cspell reported issues (run: npx cspell <files>)'] : [];
  return { ok, errors, meta: { filesChecked } };
}

/** Resolves exec function and staged list from context. */
function getContextExecAndStaged(context: { stagedFiles?: string[]; _execSync?: ExecSyncFn } | undefined): {
  execSyncFn: ExecSyncFn;
  staged: string[];
} {
  const ctx = context ?? {};
  return { execSyncFn: ctx._execSync ?? execSync, staged: ctx.stagedFiles ?? [] };
}

/** Runs cspell for context (staged paths or glob); returns output and exit code. */
function getCspellRunResult(
  root: string,
  staged: string[],
  execSyncFn: ExecSyncFn,
): { output: string; exitCode: number } {
  if (staged.length > 0) return runCspellStaged(root, staged, execSyncFn);
  return runCspellGlob(root, '**/*.md', execSyncFn);
}

/** Spell-check via cspell; with --staged runs on staged files only, else on markdown glob. */
export const cspellCheck: Check = {
  name: 'cspell',
  async run(root = process.cwd(), context) {
    const configPath = join(root, 'cspell.json');
    if (!existsSync(configPath)) return { ok: true, errors: [], meta: { filesChecked: 0 } };
    const { execSyncFn, staged } = getContextExecAndStaged(context);
    const { output, exitCode } = getCspellRunResult(root, staged, execSyncFn);
    return buildCspellResult(output, exitCode);
  },
};
