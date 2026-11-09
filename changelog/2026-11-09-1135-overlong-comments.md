### 2026.11.09.1135

- Feat: `prose-budget` fails a function body as `overlong` when its two or more comments hold over three words per code token.
- Feat: `prose.comments.maxWordsPerToken` (3) and `prose.comments.minBodyComments` (2) set the thresholds.
- Feat: words in a fenced or indented example inside a comment do not count, and one comment alone never fails.
- Test: requirement 0157 proves the rule by real runs in Go, Swift and TypeScript, and that Markdown is unaffected.
- Fix: shell and YAML comments keep the body placement they had before.
