---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# jscpd

Detects duplicated code with a native Go clone-detection engine implementing [jscpd](https://github.com/kucherenko/jscpd) semantics. Opt-in — not part of the runner's default list. Add `"jscpd"` to the `checks` array in `.fitnessrc.json` to enable it.

## Behavior

- Scans every file under root with min-lines 5 and min-tokens 50. Ignores `**/*.md`, `**/*.json`, `**/*.lock`, `**/*.test.*`, `**/*.spec.*`, and `**/*_test.go`, plus gitignored and binary files.
- Test and spec files — including Go's `_test.go` convention — are ignored: repeated mock setup and fixtures there read as false-positive duplication, not production code to refactor.
- Pass: Duplicated lines stay under 1% of the codebase (jscpd's threshold semantics).
- Fail: Over threshold — the check reports `ERROR: jscpd found too many duplicates (X%) over threshold (Y%)`.
- The engine is built into the check binary — no external jscpd tool to install.

## Escape hatch

For duplication that's a real, justified constraint (e.g. boilerplate a framework requires per type), wrap it in jscpd's inline-comment markers instead of trying to force a refactor:

```
// jscpd:ignore-start
...
// jscpd:ignore-end
```
