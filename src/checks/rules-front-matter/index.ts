import { readFileSync, statSync } from 'node:fs';
import { join, resolve } from 'node:path';
import type { Check } from '../../types/index.js';
import { findMd } from '../findMd.js';

const FRONTMATTER_RE = /^---\n([\s\S]*?)\n---/;
const ARRAY_RE = /(?:fitnessFunctions|relatedConfigurations):\s*\[([^\]]*)\]/g;

/** Parses paths from fitnessFunctions and relatedConfigurations in front matter. */
function getFrontMatterPaths(content: string): string[] {
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

/** Returns error message if path invalid, null if valid. */
function validatePath(file: string, path: string, root: string): string | null {
  if (isExternalOrAnchor(path)) return null;
  const target = resolve(root, path);
  if (!target.startsWith(root)) return `${file}: front matter path escapes repo: ${path}`;
  try {
    statSync(target);
    return null;
  } catch {
    return `${file}: front matter path missing: ${path}`;
  }
}

/** Validates a rule file's front matter paths; returns error messages. */
function validateFile(file: string, content: string, root: string): string[] {
  const errors: string[] = [];
  for (const path of getFrontMatterPaths(content)) {
    const err = validatePath(file, path, root);
    if (err) errors.push(err);
  }
  return errors;
}

/** Validates front matter: fitnessFunctions and relatedConfigurations paths must exist. */
export const rulesFrontMatterCheck: Check = {
  name: '30.03',
  async run(root = process.cwd()) {
    const errors: string[] = [];
    let filesChecked = 0;
    for (const file of findMd(root)) {
      filesChecked += 1;
      const content = readFileSync(join(root, file), 'utf8');
      errors.push(...validateFile(file, content, root));
    }
    return { ok: errors.length === 0, errors, meta: { filesChecked } };
  },
};
