import { runMarkdownFilenameCheck } from '@mayjournal/fitness-shared';
import type { Check, CheckName } from '@mayjournal/fitness';

/** camelCase: starts lowercase, then letters/digits, e.g. releaseNotes.md, adr001.md. */
export const CAMEL_RE = /^[a-z][a-zA-Z0-9]*\.md$/;

/** Ensures every `.md` basename is camelCase (standard root docs exempt). */
const markdownFilenameCamelCaseCheck = {
  // Not in the CheckName enum: opt-in-only, never joins defaultChecks (see check-name.ts).
  name: 'markdown-filename-camel-case' as CheckName,
  run: (root = process.cwd()) =>
    runMarkdownFilenameCheck(root, { label: 'camelCase', pattern: CAMEL_RE }),
  runInProcess: true,
} satisfies Check;

export default markdownFilenameCamelCaseCheck;
