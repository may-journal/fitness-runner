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
import { enUS } from './enUS.js';

const CSPELL_ISSUE_RE = /^(.+):(\d+):(\d+)\s+-\s+(.+)$/m;
const FILES_CHECKED_RE = /Files checked:\s*(\d+)/;

const EXEC_OPTS = { cwd: '', encoding: 'utf8' as const, maxBuffer: 1024 * 1024 };

/** Runs npx cspell with cmdPart; returns exit code and combined stdout+stderr. */
function execCspell(
  root: string,
  cmdPart: string,
  execSyncFn: ExecSyncFn
): { exitCode: number; output: string } {
  try {
    const out = execSyncFn(`npx cspell --no-progress ${cmdPart} 2>&1`, { ...EXEC_OPTS, cwd: root });
    return { exitCode: 0, output: out };
  } catch (e: unknown) {
    const err = e as { status?: number; stderr?: string; stdout?: string };
    return {
      exitCode: typeof err.status === 'number' ? err.status : 1,
      output: [err.stdout, err.stderr].filter(Boolean).join('\n'),
    };
  }
}

/** Run cspell; output is stdout + stderr (2>&1) so we can parse issues and filesChecked. */
export function runCspell(
  root: string,
  paths: string[],
  execSyncFn: ExecSyncFn = execSync
): { exitCode: number; output: string } {
  if (paths.length === 0) return { exitCode: 0, output: '' };
  const list = paths.map((p) => `"${p.replace(/"/g, '\\"')}"`).join(' ');
  return execCspell(root, list, execSyncFn);
}

/** Parses cspell CLI output into lines matching file:line:col - message. */
function parseIssues(output: string): string[] {
  return output
    .trim()
    .split('\n')
    .filter(Boolean)
    .filter((line) => CSPELL_ISSUE_RE.test(line));
}

/** Extracts "Files checked: N" from cspell output. */
function parseFilesChecked(output: string): number | undefined {
  const m = output.match(FILES_CHECKED_RE);
  return m ? parseInt(m[1], 10) : undefined;
}

/** Runs cspell with a glob pattern (e.g. all .md files). */
function runCspellGlob(
  root: string,
  glob: string,
  execSyncFn: ExecSyncFn
): { exitCode: number; output: string } {
  return execCspell(root, `"${glob.replace(/"/g, '\\"')}"`, execSyncFn);
}

/** Runs cspell on staged files that exist under root. */
function runCspellStaged(
  root: string,
  stagedFiles: string[],
  execSyncFn: ExecSyncFn
): { exitCode: number; output: string } {
  const paths = stagedFiles.filter((p) => existsSync(join(root, p))).map((p) => join(root, p));
  return paths.length === 0 ? { exitCode: 0, output: '' } : runCspell(root, paths, execSyncFn);
}

/** Builds CheckResult from exec output, parsed issues, and files-checked count. */
function buildCspellResult(output: string, exitCode: number) {
  const issues = parseIssues(output);
  const ok = exitCode === 0 && issues.length === 0;
  const errors = issues.length ? issues : ok ? [] : [enUS.FallbackRunHint];
  return checkResult(ok, errors, parseFilesChecked(output) ?? 0);
}

/** Runs cspell via CLI: staged files if any, else all .md files. */
function getCspellRunResult(
  root: string,
  staged: string[],
  execSyncFn: ExecSyncFn
): { exitCode: number; output: string } {
  return staged.length > 0
    ? runCspellStaged(root, staged, execSyncFn)
    : runCspellGlob(root, '**/*.md', execSyncFn);
}

/** Converts character offset in text to 1-based line and column. */
function offsetToLineCol(text: string, offset: number): { col: number; line: number } {
  let line = 1,
    col = 1;
  for (let i = 0; i < offset && i < text.length; i++) {
    if (text[i] === '\n') {
      line++;
      col = 1;
    } else {
      col++;
    }
  }
  return { col, line };
}

/** Formats one cspell-lib issue as file:line:col - message. */
function formatLibIssue(
  filePath: string,
  issue: { line?: { offset?: number }; message?: string; text?: string },
  text: string
): string {
  const off = typeof issue.line?.offset === 'number' ? issue.line.offset : 0;
  const { col, line } = offsetToLineCol(text, off);
  const msg = `${issue.message ?? ''}${issue.text ? `: ${issue.text}` : ''}`;
  return `${filePath}:${line}:${col} - ${msg}`;
}

/** Spell-checks one file via cspell-lib; returns error lines or FallbackRunHint on throw. */
async function checkOneFileWithLib(
  filePath: string,
  opts: { noConfigSearch: true },
  config: Awaited<ReturnType<typeof readConfigFile>>
): Promise<string[]> {
  try {
    const result = await spellCheckFile(filePath, opts, config);
    const text = (result.document as { text?: string }).text ?? '';
    return result.issues.map((issue) => formatLibIssue(filePath, issue, text));
  } catch {
    return [`${filePath}: ${enUS.FallbackRunHint}`];
  }
}

/** Absolute paths to check: staged existing paths under root, or .md files when no staged. */
function getPathsToCheck(root: string, staged: string[]): string[] {
  const rel =
    staged.length > 0
      ? staged.filter((p) => existsSync(join(root, p)))
      : findFilesByExtension(root, '.md');
  return rel.map((p) => join(root, p));
}

/** Runs cspell via CLI (execSync) and returns CheckResult. */
function runViaExec(
  root: string,
  staged: string[],
  execSyncFn: ExecSyncFn
): ReturnType<typeof checkResult> {
  const { output, exitCode } = getCspellRunResult(root, staged, execSyncFn);
  return buildCspellResult(output, exitCode);
}

/** Runs cspell via cspell-lib and returns CheckResult. */
async function runViaLib(root: string, staged: string[]): Promise<ReturnType<typeof checkResult>> {
  const paths = getPathsToCheck(root, staged);
  if (paths.length === 0) return checkResult(true, [], 0);
  const configPath = join(root, 'cspell.json');
  const config = await readConfigFile(configPath, root);
  const opts = { noConfigSearch: true as const };
  const allErrors: string[] = [];
  for (const filePath of paths) {
    allErrors.push(...(await checkOneFileWithLib(filePath, opts, config)));
  }
  return checkResult(allErrors.length === 0, allErrors, paths.length);
}

/** Cspell check: uses CLI when context provides execSync, else cspell-lib. */
export const cspellCheck: Check = {
  name: CheckName.Cspell,
  async run(root = process.cwd(), context) {
    const configPath = join(root, 'cspell.json');
    if (!existsSync(configPath)) return checkResult(true, [], 0);
    const staged = getStagedFiles(context);
    const execSyncFn = getExecSync(context);
    if (context?._execSync) return runViaExec(root, staged, execSyncFn);
    return runViaLib(root, staged);
  },
};
