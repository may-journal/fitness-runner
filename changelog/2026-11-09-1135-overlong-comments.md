### 2026.11.09.1135

- Feat: `prose-budget` fails a function body as `overlong` when its two or more comments hold over three words per code token.
- Feat: each `overlong` error names the body's opening line, its comment words, comments and code tokens, and says to keep the why.
- Feat: words in a fenced or indented example inside a comment do not count, and one comment alone never fails.
- Test: requirement 0157 proves the rule by real runs in Go, Swift and TypeScript, and that Markdown is unaffected.
