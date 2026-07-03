---
relatedConfigurations: ['../../../package.json']
---

# jscpd

Runs [jscpd](https://github.com/kucherenko/jscpd) to detect duplicated code. Opt-in — not part of `defaultChecks`. Add `'jscpd'` to `.fitnessrc` `checks` to enable it.

## Behavior

- Runs: `npx jscpd --min-lines 5 --min-tokens 50 --threshold 1 --reporters console .` from repo root (respects `.gitignore` by default).
- Pass: Duplicated lines stay under 1% of the codebase (jscpd's own `--threshold`).
- Fail: Over threshold — jscpd exits non-zero; the check reports jscpd's own `ERROR: jscpd found too many duplicates (X%) over threshold (Y%)` line.
- `jscpd` is bundled as a dependency of `@mayjournal/fitness-shared` — consumers do not install it separately.

## Escape hatch

For duplication that's a real, justified constraint (e.g. boilerplate a framework requires per type), wrap it in jscpd's inline-comment markers instead of trying to force a refactor:

```
// jscpd:ignore-start
...
// jscpd:ignore-end
```
