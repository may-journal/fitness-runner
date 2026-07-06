---
relatedConfigurations: ['../../../package.json']
---

# no-eslint-disable

Fails when any source file contains an ESLint disable directive, keeping lint hygiene on the fix-the-rule path instead of accumulating suppressions. Opt-in — not part of `defaultChecks`. Add `'no-eslint-disable'` to `.fitnessrc` `checks` to enable it.

## Behavior

- Scans source files with these extensions: `.ts`, `.tsx`, `.js`, `.mjs`, `.cjs`, `.mts`, `.cts`. Each extension is discovered separately via `findFilesByExtension` and the results are combined and de-duplicated. Test and spec files ARE included in the scan — this is a zero-config strict policy with no allowlist.
- Detects every disable directive form via the regex `eslint-disable(-next-line|-line)?`: file-level `eslint-disable`, block `eslint-disable ... eslint-enable`, `eslint-disable-line`, and `eslint-disable-next-line`.
- Pass: no scanned file contains a disable directive.
- Fail: one error per matching line, formatted `path/to/file.ts:42: eslint-disable-next-line` with a 1-based line number.

## Notes

- `filesChecked` counts every scanned source file across all extensions (after de-duplication), excluding standard skip dirs (`node_modules`, `dist`, `coverage`, `.git`, and the other runner skip dirs).
- Detection is a plain line scan, so the directive text inside a string literal is also flagged; v1 is intentionally grep-like. Reword or restructure such a rare case rather than suppressing it.
