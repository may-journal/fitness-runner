import { basename } from 'node:path';
import { checkResult, findFilesByExtension } from '@mayjournal/fitness-shared';
import type { Check, CheckName } from '@mayjournal/fitness';

/** Standard root filenames always allowed regardless of case (conventional OSS docs). */
export const ALLOWED_BASENAMES = new Set<string>([
  'CHANGELOG.md',
  'CODE_OF_CONDUCT.md',
  'CONTRIBUTING.md',
  'LICENSE.md',
  'README.md',
  'SECURITY.md',
]);

/** Kebab-case: lowercase alphanumeric segments joined by single hyphens, e.g. api-design.md. */
const KEBAB_RE = /^[a-z0-9]+(-[a-z0-9]+)*\.md$/;
/** camelCase: starts lowercase, then letters/digits, e.g. releaseNotes.md, adr001.md. */
const CAMEL_RE = /^[a-z][a-zA-Z0-9]*\.md$/;

/** True when the basename is an allowed exception or matches kebab-case or camelCase. */
export function isValidMarkdownBasename(base: string): boolean {
  if (ALLOWED_BASENAMES.has(base)) return true;
  return KEBAB_RE.test(base) || CAMEL_RE.test(base);
}

/** Returns an error message for a non-conforming markdown path, or empty array when valid. */
export function validateMarkdownFile(relPath: string): string[] {
  if (isValidMarkdownBasename(basename(relPath))) return [];
  return [`${relPath}: filename must be kebab-case or camelCase`];
}

/** Ensures every .md basename is kebab-case or camelCase (standard root docs exempt). */
const markdownFilenameConventionCheck = {
  name: 'markdown-filename-convention' as CheckName,
  async run(root = process.cwd()) {
    const errors: string[] = [];
    const mdFiles = await findFilesByExtension(root, '.md');
    for (const file of mdFiles) errors.push(...validateMarkdownFile(file));
    return checkResult(errors.length === 0, errors, mdFiles.length);
  },
  runInProcess: true,
} satisfies Check;

export default markdownFilenameConventionCheck;
