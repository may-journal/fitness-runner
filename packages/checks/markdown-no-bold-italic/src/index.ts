import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { CHANGELOG_MD, checkResult, findFilesByExtension } from '@mayjournal/fitness-shared';
import { CheckName, type Check } from '@mayjournal/fitness';

/** Remove fenced code blocks (```...```) and inline code (`...`) so emphasis inside code is not flagged. */
function stripCodeForEmphasisCheck(content: string): string {
  let s = content.replace(/```[\s\S]*?```/g, '\n');
  return s.replace(/`[^`]*`/g, ' ');
}

/** Remove markdown links [text](url) so underscores in URLs or link text are not flagged as italic. */
function stripLinkBlocks(content: string): string {
  return content.replace(/\[[^\]]*\]\([^)]*\)/g, ' ');
}

/** Matches **bold** (asterisk). */
const BOLD_ASTERISK_RE = /\*\*[^*]*\*\*/g;
/** Matches __bold__ (underscore). */
const BOLD_UNDERSCORE_RE = /__[^_]*__/g;
/** Matches *italic* (single asterisk, not part of **); excludes list markers by not spanning newlines. */
const ITALIC_ASTERISK_RE = /(?<!\*)\*[^*\n]+\*(?!\*)/g;
/** Matches _italic_ (single underscore, not part of __). */
const ITALIC_UNDERSCORE_RE = /(?<!_)_[^_]+_(?!_)/g;

const EMPHASIS_RULES: [RegExp, string][] = [
  [BOLD_ASTERISK_RE, '**bold**'],
  [BOLD_UNDERSCORE_RE, '__bold__'],
  [ITALIC_ASTERISK_RE, '*italic*'],
  [ITALIC_UNDERSCORE_RE, '_italic_'],
];

/** Returns all disallowed markdown emphasis matches in content (bold/italic); ignores content inside code and links. */
export function findDisallowedEmphasis(content: string): { kind: string; match: string }[] {
  let stripped = stripCodeForEmphasisCheck(content);
  stripped = stripLinkBlocks(stripped);
  const out: { kind: string; match: string }[] = [];
  for (const [re, kind] of EMPHASIS_RULES) {
    re.lastIndex = 0;
    let m: RegExpExecArray | null;
    while ((m = re.exec(stripped)) !== null) out.push({ kind, match: m[0] });
  }
  return out;
}

/** Validates one markdown file; returns error messages for disallowed bold/italic. */
function validateFile(relPath: string, content: string): string[] {
  const hits = findDisallowedEmphasis(content);
  return hits.map(
    ({ kind, match }) =>
      `${relPath}: disallowed ${kind} (use only when explicitly required): ${JSON.stringify(match)}`
  );
}

/** Ensures markdown files do not use **bold**, __bold__, *italic*, or _italic_ per AI-generated markdown convention. */
export const markdownNoBoldItalicCheck: Check = {
  name: CheckName.MarkdownNoBoldItalic,
  async run(root = process.cwd()) {
    const errors: string[] = [];
    let filesChecked = 0;
    const mdFiles = (await findFilesByExtension(root, '.md')).filter((f) => f !== CHANGELOG_MD);
    for (const file of mdFiles) {
      filesChecked += 1;
      const content = readFileSync(join(root, file), 'utf8');
      errors.push(...validateFile(file, content));
    }
    return checkResult(errors.length === 0, errors, filesChecked);
  },
  runInProcess: true,
};

export default markdownNoBoldItalicCheck;
