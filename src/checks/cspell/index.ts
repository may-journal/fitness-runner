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
    const out = [err.stdout, err.stderr].filter(Boolean).join('\n');
    return { output: out, exitCode: typeof err.status === 'number' ? err.status : 1 };
  }
}

/** Parse cspell output into error lines (file:line:col - message). */
function parseIssues(output: string): string[] {
  const lines = output.trim().split('\n').filter(Boolean);
  return lines.filter((line) => CSPELL_ISSUE_RE.test(line));
}

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
    const out = [err.stdout, err.stderr].filter(Boolean).join('\n');
    return { output: out, exitCode: typeof err.status === 'number' ? err.status : 1 };
  }
}

/** Spell-check via cspell; with --staged runs on staged files only, else on markdown glob. */
export const cspellCheck: Check = {
  name: 'cspell',
  async run(root = process.cwd(), context) {
    const configPath = join(root, 'cspell.json');
    if (!existsSync(configPath)) {
      return { ok: true, errors: [], meta: { filesChecked: 0 } };
    }
    const execSyncFn = context?._execSync ?? execSync;
    let output: string;
    let exitCode: number;
    if (context?.stagedFiles?.length) {
      const paths = context.stagedFiles.filter((p) => existsSync(join(root, p)));
      if (paths.length === 0) {
        return { ok: true, errors: [], meta: { filesChecked: 0 } };
      }
      const absPaths = paths.map((p) => join(root, p));
      const result = runCspell(root, absPaths, execSyncFn);
      ({ output, exitCode } = result);
    } else {
      const result = runCspellGlob(root, '**/*.md', execSyncFn);
      ({ output, exitCode } = result);
    }
    const issues = parseIssues(output);
    const filesChecked = parseFilesChecked(output);
    const ok = exitCode === 0 && issues.length === 0;
    const errors =
      issues.length > 0 ? issues : !ok ? ['cspell reported issues (run: npx cspell <files>)'] : [];
    return {
      ok,
      errors,
      meta: { filesChecked: filesChecked ?? 0 },
    };
  },
};
