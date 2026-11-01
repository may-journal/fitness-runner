### 2026.11.01.1130

- Feat: `prose-budget` fails a code comment sentence that starts with a connector such as So, or holds three words or fewer, as `comment-fragment`.
- Feat: `prose-budget` fails any comment of more than two sentences, doc comments included, as `comment-one-statement`.
- Feat: only a Go package comment keeps the paragraph limits, and Markdown files are judged as before.
- Fix: a wrapped list item joins its item, a fence in a list closes, and a lone `?` ends no fragment.
- Test: requirement 0137 proves by real runs that a fragment, a "So" sentence and a split statement fail in Go, Swift and TypeScript.
