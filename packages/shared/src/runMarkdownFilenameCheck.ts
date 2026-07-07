import { basename } from 'node:path';
import { checkResult } from './checkResult.js';
import { findFilesByExtension } from './findFilesByExtension.js';
import type { CheckResult } from './types/check-result.types.js';

/** Standard root filenames always allowed regardless of case (conventional OSS docs). */
export const ALLOWED_MARKDOWN_BASENAMES = new Set<string>([
  'CHANGELOG.md',
  'CODE_OF_CONDUCT.md',
  'CONTRIBUTING.md',
  'LICENSE.md',
  'README.md',
  'SECURITY.md',
]);

/** A single markdown filename convention: the pattern a basename must match, and a human label for errors. */
export interface FilenameConvention {
  /** Human label used in the error message, e.g. `kebab-case`. */
  label: string;
  /** Regex the full basename (including `.md`) must match to pass. */
  pattern: RegExp;
}

/** Returns an error for a non-conforming markdown path under `convention`, or empty when valid or exempt. */
export function validateMarkdownFilename(
  relPath: string,
  convention: FilenameConvention
): string[] {
  const base = basename(relPath);
  if (ALLOWED_MARKDOWN_BASENAMES.has(base)) return [];
  if (convention.pattern.test(base)) return [];
  return [`${relPath}: filename must be ${convention.label}`];
}

/**
 * Runs one markdown filename convention over every `.md` file under `root`
 * (standard root docs exempt). Shared by the kebab-case and camelCase filename
 * checks so each supplies only its own `convention` — same function, two flavors.
 */
export async function runMarkdownFilenameCheck(
  root: string,
  convention: FilenameConvention
): Promise<CheckResult> {
  const errors: string[] = [];
  const mdFiles = await findFilesByExtension(root, '.md');
  for (const file of mdFiles) errors.push(...validateMarkdownFilename(file, convention));
  return checkResult(errors.length === 0, errors, mdFiles.length);
}
