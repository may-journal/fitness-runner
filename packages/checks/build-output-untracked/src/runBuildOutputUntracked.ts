import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import {
  checkResult,
  execSyncResult,
  findFilesByExtension,
  getExecSync,
} from '@mayjournal/fitness-shared';
import type { ExecSyncFn } from '@mayjournal/fitness-shared';
import type { Check, CheckName, RunContext } from '@mayjournal/fitness';
import { enUS } from './enUS.js';

/** Source extensions scanned for imports of build output. */
const SOURCE_EXTENSIONS = ['.ts', '.mts', '.cts'];

/** Matches the specifier of `from '…'`, `import('…')`, and `require('…')`. */
const IMPORT_SPECIFIER_RE = /\b(?:from|import|require)\s*\(?\s*['"]([^'"]+)['"]/g;

/** True when a specifier reaches into a `dist/` build-output path. */
const DIST_SPECIFIER_RE = /(^|\/)dist\/|\.\.\/dist/;

/** Non-empty, trimmed lines of a git command's output. */
function splitLines(output: string): string[] {
  return output
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean);
}

/** True when `git check-ignore -q dist` exits 0 (dist is git-ignored). */
function isDistIgnored(root: string, execSyncFn: ExecSyncFn): boolean {
  return execSyncResult(root, 'git check-ignore -q dist', execSyncFn).exitCode === 0;
}

/** Git-tracked files under dist (root-level `dist` and any nested `**\/dist\/**`), deduped. */
function trackedDistFiles(root: string, execSyncFn: ExecSyncFn): string[] {
  const direct = execSyncResult(root, 'git ls-files -- dist', execSyncFn).output;
  const nested = execSyncResult(root, "git ls-files -- '**/dist/**'", execSyncFn).output;
  return [...new Set([...splitLines(direct), ...splitLines(nested)])].sort();
}

/** Errors for rule A: dist must be git-ignored and have no tracked files. */
export function distTrackingErrors(root: string, execSyncFn: ExecSyncFn): string[] {
  const errors: string[] = [];
  if (!isDistIgnored(root, execSyncFn)) errors.push(enUS.NotIgnored);
  const tracked = trackedDistFiles(root, execSyncFn);
  if (tracked.length > 0) {
    errors.push(`${enUS.RemovePrefix} ${tracked.join(', ')} (git rm --cached)`);
  }
  return errors;
}

/** Specifiers on one line that import from a dist/ build-output path. */
function distSpecifiersInLine(line: string): string[] {
  const out: string[] = [];
  IMPORT_SPECIFIER_RE.lastIndex = 0;
  let match: RegExpExecArray | null;
  while ((match = IMPORT_SPECIFIER_RE.exec(line)) !== null) {
    if (DIST_SPECIFIER_RE.test(match[1])) out.push(match[1]);
  }
  return out;
}

/** Errors for rule B: `path:line` for each source line importing build output. */
export function scanFileForDistImports(relPath: string, content: string): string[] {
  const errors: string[] = [];
  content.split('\n').forEach((line, index) => {
    for (const spec of distSpecifiersInLine(line)) {
      errors.push(`${relPath}:${index + 1} imports build output: ${JSON.stringify(spec)}`);
    }
  });
  return errors;
}

/** Sorted, deduped source files (.ts/.mts/.cts) under root. */
export async function collectSourceFiles(root: string): Promise<string[]> {
  const lists = await Promise.all(SOURCE_EXTENSIONS.map((ext) => findFilesByExtension(root, ext)));
  return [...new Set(lists.flat())].sort();
}

/** Fails when dist is tracked/un-ignored or a source file imports from build output. */
export const buildOutputUntrackedCheck = {
  name: 'build-output-untracked' as CheckName,
  async run(root = process.cwd(), context?: RunContext) {
    const errors = distTrackingErrors(root, getExecSync(context));
    const files = await collectSourceFiles(root);
    for (const file of files) {
      errors.push(...scanFileForDistImports(file, readFileSync(join(root, file), 'utf8')));
    }
    return checkResult(errors.length === 0, errors, files.length + 1);
  },
  runInProcess: true,
} satisfies Check;

export default buildOutputUntrackedCheck;
