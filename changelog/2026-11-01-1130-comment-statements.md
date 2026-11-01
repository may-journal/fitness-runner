### 2026.11.01.1130

- Feat: `prose-budget` fails a code comment sentence that starts with a connector such as So, or holds three words or fewer, as `comment-fragment`.
- Feat: `prose-budget` fails a comment in a function body or after code that holds more than two sentences, as `comment-one-statement`.
- Feat: a doc comment on a declaration, file or package keeps the paragraph limits, and Markdown files are judged as before.
- Fix: a wrapped list item joins its item, a fence in a list closes, and a lone `?` ends no fragment.
- Test: requirement 0137 proves by real runs that a fragment, a "So" sentence and a split statement fail in Go, Swift and TypeScript.
