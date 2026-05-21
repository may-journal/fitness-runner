---
fitnessFunctions: ['markdown-no-bold-italic']
---

# markdown-no-bold-italic

## Behavior

- Pass: No .md files, or no markdown file contains bold (double asterisk or double underscore) or italic (single asterisk or single underscore) emphasis.
- Fail: Any .md file contains disallowed emphasis; error per occurrence with file path and snippet.

Scans all `.md` files under root (same as other markdown checks; excludes node_modules, dist, coverage, .git, githooks). Use bold/italic only when explicitly required. Ignores emphasis inside fenced code, inline code, and markdown link blocks `[text](url)` (so underscores in URLs or link text are not flagged).
