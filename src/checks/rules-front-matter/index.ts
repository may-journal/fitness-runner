import { readFileSync, statSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { checkResult } from '../../utils/checkResult.js';
import { CheckName } from '../../types/index.types.js';
import type { Check, RunContext } from '../../types/index.types.js';
import { findFilesByExtension } from '../../utils/findFilesByExtension.js';

const FRONTMATTER_RE = /^---\n([\s\S]*?)\n---/;
const ARRAY_RE = /(?:fitnessFunctions|relatedConfigurations):\s*\[([^\]]*)\]/g;
const EMPTY_ARRAY_RE = /(fitnessFunctions|relatedConfigurations):\s*\[\s*\]/g;

/** True if content has front matter with at least one of fitnessFunctions or relatedConfigurations. Exported for tests. */
export function hasRequiredFrontMatter(content: string): boolean {
  const fm = content.match(FRONTMATTER_RE);
  if (!fm) return false;
  const inner = fm[1];
  return inner.includes('fitnessFunctions') || inner.includes('relatedConfigurations');
}

/** Parses paths from fitnessFunctions and relatedConfigurations in front matter. Exported for tests. */
export function getFrontMatterPaths(content: string): string[] {
  const fm = content.match(FRONTMATTER_RE);
  if (!fm) return [];
  const paths: string[] = [];
  for (const m of fm[1].matchAll(ARRAY_RE)) {
    const inner = m[1].trim();
    if (!inner) continue;
    for (const p of inner.split(',')) paths.push(p.trim().replace(/^['"]|['"]$/g, ''));
  }
  return paths;
}

/** Returns true for http, #, or mailto paths (skipped from validation). */
function isExternalOrAnchor(path: string): boolean {
  return path.startsWith('http') || path.startsWith('#') || path.startsWith('mailto:');
}

/** Returns errors for empty fitnessFunctions or relatedConfigurations arrays in front matter inner. */
function getEmptyArrayErrors(file: string, fmInner: string): string[] {
  const errors: string[] = [];
  for (const m of fmInner.matchAll(EMPTY_ARRAY_RE))
    errors.push(`${file}: ${m[1]} must not be an empty array`);
  return errors;
}

/** Returns error message if path invalid, null if valid. Path is resolved relative to the md file's directory. */
function validatePath(
  file: string,
  path: string,
  root: string,
  mdFileDir: string,
  registeredCheckNames: string[]
): string | null {
  if (isExternalOrAnchor(path)) return null;
  if (registeredCheckNames.includes(path)) return null;
  const target = resolve(mdFileDir, path);
  if (!target.startsWith(root)) return `${file}: front matter path escapes repo: ${path}`;
  try {
    statSync(target);
    return null;
  } catch {
    return `${file}: front matter path missing: ${path}`;
  }
}

/** Validates paths in front matter and returns error messages. */
function getPathErrors(
  file: string,
  content: string,
  root: string,
  mdFileDir: string,
  registeredCheckNames: string[]
): string[] {
  const errors: string[] = [];
  for (const path of getFrontMatterPaths(content)) {
    const err = validatePath(file, path, root, mdFileDir, registeredCheckNames);
    if (err) errors.push(err);
  }
  return errors;
}

/** Validates a rule file has required front matter and paths; returns error messages. Paths are relative to the md file. */
function validateFile(
  file: string,
  content: string,
  root: string,
  mdFileDir: string,
  registeredCheckNames: string[]
): string[] {
  const fm = content.match(FRONTMATTER_RE);
  if (!fm || !hasRequiredFrontMatter(content)) {
    return [`${file}: missing front matter with fitnessFunctions or relatedConfigurations`];
  }
  const emptyErrors = getEmptyArrayErrors(file, fm[1]);
  if (emptyErrors.length > 0) return emptyErrors;
  return getPathErrors(file, content, root, mdFileDir, registeredCheckNames);
}

/** Validates front matter: fitnessFunctions and relatedConfigurations paths must exist; entries may be registered check names. */
export const rulesFrontMatterCheck: Check = {
  folder: 'rules-front-matter',
  name: CheckName.MarkdownFrontMatter,
  async run(root = process.cwd(), context?: RunContext) {
    const registeredCheckNames = context?.registeredCheckNames ?? [];
    const errors: string[] = [];
    let filesChecked = 0;
    for (const file of await findFilesByExtension(root, '.md')) {
      filesChecked += 1;
      const content = readFileSync(join(root, file), 'utf8');
      const mdFileDir = join(root, dirname(file));
      errors.push(...validateFile(file, content, root, mdFileDir, registeredCheckNames));
    }
    return checkResult(errors.length === 0, errors, filesChecked);
  },
  runInProcess: true,
};
