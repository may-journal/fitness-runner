import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import {
  SOURCE_FILE_EXTENSIONS,
  checkResult,
  findFilesByExtension,
} from '@mayjournal/fitness-shared';
import type { Check, CheckName } from '@mayjournal/fitness';

/** Source file extensions scanned for eslint disable directives (test files included). */
export const SOURCE_EXTENSIONS = SOURCE_FILE_EXTENSIONS;

/** Matches any eslint disable directive form (file, block, `-line`, and `-next-line`). */
export const ESLINT_DISABLE_RE = /eslint-disable(-next-line|-line)?/;

/** Returns the sorted, de-duplicated relative paths of all scanned source files under root. */
export async function findSourceFiles(root: string): Promise<string[]> {
  const groups = await Promise.all(SOURCE_EXTENSIONS.map((ext) => findFilesByExtension(root, ext)));
  return [...new Set(groups.flat())].sort();
}

/** Returns one error message per line in content that contains an eslint disable directive. */
export function scanContent(relPath: string, content: string): string[] {
  const errors: string[] = [];
  const lines = content.split('\n');
  for (let i = 0; i < lines.length; i += 1) {
    const match = lines[i].match(ESLINT_DISABLE_RE);
    if (match) errors.push(`${relPath}:${i + 1}: ${match[0]}`);
  }
  return errors;
}

/** Fails when any scanned source file contains an eslint disable directive. */
export const noEslintDisableCheck = {
  name: 'no-eslint-disable' as CheckName,
  async run(root = process.cwd()) {
    const files = await findSourceFiles(root);
    const errors: string[] = [];
    for (const file of files) {
      errors.push(...scanContent(file, readFileSync(join(root, file), 'utf8')));
    }
    return checkResult(errors.length === 0, errors, files.length);
  },
  runInProcess: true,
} satisfies Check;

export default noEslintDisableCheck;
