---
fitnessFunctions: ['markdown-no-bold-italic']
---

# markdown-no-bold-italic

## Behavior

- Pass: No .md files, or no markdown file contains bold (double asterisk or double underscore) or italic (single asterisk or single underscore) emphasis.
- Fail: Any .md file contains disallowed emphasis; error per occurrence with file path and snippet.

Scans all `.md` files under root (same as other markdown checks; excludes node_modules, dist, coverage, .git, .husky). Use bold/italic only when explicitly required.
