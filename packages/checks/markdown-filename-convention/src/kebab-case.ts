import { runMarkdownFilenameCheck } from '@mayjournal/fitness-shared';
import type { Check, CheckName } from '@mayjournal/fitness';

/** Kebab-case: lowercase alphanumeric segments joined by single hyphens, e.g. api-design.md. */
export const KEBAB_RE = /^[a-z0-9]+(-[a-z0-9]+)*\.md$/;

/** Ensures every `.md` basename is kebab-case (standard root docs exempt). */
const markdownFilenameKebabCaseCheck = {
  // Not in the CheckName enum: opt-in-only, never joins defaultChecks (see check-name.ts).
  name: 'markdown-filename-kebab-case' as CheckName,
  run: (root = process.cwd()) =>
    runMarkdownFilenameCheck(root, { label: 'kebab-case', pattern: KEBAB_RE }),
  runInProcess: true,
} satisfies Check;

export default markdownFilenameKebabCaseCheck;
