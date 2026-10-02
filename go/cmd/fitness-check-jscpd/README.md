---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# jscpd

Detects duplicated code with a native Go clone-detection engine implementing [jscpd](https://github.com/kucherenko/jscpd) semantics. On by default; turn it off with `disabledChecks` in `.fitnessrc.json`.

## Behavior

- Scans every tracked file with min-lines 5, min-tokens 50, tests, Markdown, JSON, and lock files included.
- Binary files are skipped, since they have no lines to compare.
- Files the repo's `.gitattributes` marks `linguist-generated`, such as lock files, are left out; that mark is the only skip.
- Pass: Duplicated lines stay under 1% of the codebase (jscpd's threshold semantics).
- Fail: Over threshold — the check reports `ERROR: jscpd found too many duplicates (X%) over threshold (Y%)`.
- The engine is built into the check binary — no external jscpd tool to install.

## Escape hatch

For duplication that's a real, justified constraint (e.g. boilerplate a framework requires per type), wrap it in jscpd's inline-comment markers. Don't force a refactor:

```
// jscpd:ignore-start
...
// jscpd:ignore-end
```
