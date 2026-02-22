import { existsSync } from 'node:fs';
import { execSync } from 'node:child_process';
import { join } from 'node:path';
import { readConfigFile, spellCheckFile } from 'cspell-lib';
import { checkResult } from '../../utils/checkResult.js';
import { findFilesByExtension } from '../../utils/findFilesByExtension.js';
import { getExecSync, getStagedFiles } from '../../utils/runContext.js';
import type { ExecSyncFn } from '../../utils/runContext.js';
import { CheckName } from '../../types/index.types.js';
import type { Check } from '../../types/index.types.js';

const CSPELL_ISSUE_RE = /^(.+):(\d+):(\d+)\s+-\s+(.+)$/m;
const FILES_CHECKED_RE = /Files checked:\s*(\d+)/;

const EXEC_OPTS = { cwd: '', encoding: 'utf8' as const, maxBuffer: 1024 * 1024 };

/** Run cspell; output is stdout + stderr (2>&1) so we can parse issues and filesChecked. */
export function runCspell(
  root: string,
  paths: string[],
  execSyncFn: ExecSyncFn = execSync
): { exitCode: number; output: string } {
  if (paths.length === 0) {
    return { exitCode: 0, output: '' };
  }
  const list = paths.map((p) => `"${p.replace(/"/g, '\\"')}"`).join(' ');
  try {
    const out = execSyncFn(`npx cspell --no-progress ${list} 2>&1`, { ...EXEC_OPTS, cwd: root });
    return { exitCode: 0, output: out };
  } catch (e: unknown) {
    const err = e as { status?: number; stderr?: string; stdout?: string };
    const out = [err.stdout, err.stderr].filter(Boolean).join('\n');
    return { exitCode: typeof err.status === 'number' ? err.status : 1, output: out };
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
  execSyncFn: ExecSyncFn = execSync
): { exitCode: number; output: string } {
  try {
    const out = execSyncFn(`npx cspell --no-progress "${glob.replace(/"/g, '\\"')}" 2>&1`, {
      ...EXEC_OPTS,
      cwd: root,
    });
    return { exitCode: 0, output: out };
  } catch (e: unknown) {
    const err = e as { status?: number; stderr?: string; stdout?: string };
    const out = [err.stdout, err.stderr].filter(Boolean).join('\n');
    return { exitCode: typeof err.status === 'number' ? err.status : 1, output: out };
  }
}

/** Run cspell on staged paths; returns empty output if no valid paths. */
function runCspellStaged(
  root: string,
  stagedFiles: string[],
  execSyncFn: ExecSyncFn
): { exitCode: number; output: string } {
  const paths = stagedFiles.filter((p) => existsSync(join(root, p)));
  if (paths.length === 0) return { exitCode: 0, output: '' };
  return runCspell(
    root,
    paths.map((p) => join(root, p)),
    execSyncFn
  );
}

/** Builds check result from cspell output and exit code. */
function buildCspellResult(output: string, exitCode: number) {
  const issues = parseIssues(output);
  const filesChecked = parseFilesChecked(output) ?? 0;
  const ok = exitCode === 0 && issues.length === 0;
  const errors =
    issues.length > 0 ? issues : !ok ? ['cspell reported issues (run: npx cspell <files>)'] : [];
  return checkResult(ok, errors, filesChecked);
}

/** Runs cspell for context (staged paths or glob); returns output and exit code. */
function getCspellRunResult(
  root: string,
  staged: string[],
  execSyncFn: ExecSyncFn
): { exitCode: number; output: string } {
  if (staged.length > 0) return runCspellStaged(root, staged, execSyncFn);
  return runCspellGlob(root, '**/*.md', execSyncFn);
}

/** 1-based line and column from document text and character offset. */
function offsetToLineCol(text: string, offset: number): { col: number; line: number } {
  let line = 1;
  let col = 1;
  for (let i = 0; i < offset && i < text.length; i++) {
    if (text[i] === '\n') {
      line++;
      col = 1;
    } else col++;
  }
  return { col, line };
}

/** Format one cspell-lib issue as "file:line:col - message: word". */
function formatLibIssue(
  filePath: string,
  issue: { line?: { offset?: number }; message?: string },
  text: string
): string {
  const off = typeof issue.line?.offset === 'number' ? issue.line.offset : 0;
  const { col, line } = offsetToLineCol(text, off);
  const msg = issue.message ?? 'Unknown word';
  const word = (issue as { text?: string }).text;
  return `${filePath}:${line}:${col} - ${msg}${word ? `: ${word}` : ''}`;
}

/** Spell-check one file with cspell-lib; returns error lines for that file. */
async function checkOneFileWithLib(
  filePath: string,
  opts: { noConfigSearch: true },
  config: Awaited<ReturnType<typeof readConfigFile>>
): Promise<{ errors: string[] }> {
  try {
    const result = await spellCheckFile(filePath, opts, config);
    const doc = result.document as { text?: string };
    const text = doc.text ?? '';
    const errors = result.issues.map((issue) => formatLibIssue(filePath, issue, text));
    return { errors };
  } catch {
    return { errors: [`${filePath}: cspell reported issues (run: npx cspell <files>)`] };
  }
}

/** Spell-check in-process via cspell-lib (faster than CLI); returns errors and filesChecked. */
async function runCspellWithLib(
  root: string,
  configPath: string,
  paths: string[]
): Promise<{ errors: string[]; filesChecked: number; ok: boolean }> {
  if (paths.length === 0) return { errors: [], filesChecked: 0, ok: true };
  const config = await readConfigFile(configPath, root);
  const opts = { noConfigSearch: true as const };
  const allErrors: string[] = [];
  for (const filePath of paths) {
    const { errors } = await checkOneFileWithLib(filePath, opts, config);
    allErrors.push(...errors);
  }
  return { errors: allErrors, filesChecked: paths.length, ok: allErrors.length === 0 };
}

/** Paths to check: staged (existing) or all .md under root. */
function getPathsToCheck(root: string, staged: string[]): string[] {
  if (staged.length > 0)
    return staged.filter((p) => existsSync(join(root, p))).map((p) => join(root, p));
  return findFilesByExtension(root, '.md').map((p) => join(root, p));
}

/** Spell-check via cspell; when context has stagedFiles runs on those paths only, else on markdown glob. */
export const cspellCheck: Check = {
  name: CheckName.Cspell,
  async run(root = process.cwd(), context) {
    const configPath = join(root, 'cspell.json');
    if (!existsSync(configPath)) return checkResult(true, [], 0);
    const staged = getStagedFiles(context);
    const execSyncFn = getExecSync(context);
    const useCli = !!context?._execSync;
    if (useCli) {
      const { output, exitCode } = getCspellRunResult(root, staged, execSyncFn);
      return buildCspellResult(output, exitCode);
    }
    const paths = getPathsToCheck(root, staged);
    const lib = await runCspellWithLib(root, configPath, paths);
    return checkResult(lib.ok, lib.errors, lib.filesChecked);
  },
};
