---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# no-eslint-disable

Fails when any source file contains an ESLint disable directive, keeping lint hygiene on the fix-the-rule path instead of accumulating suppressions. Opt-in — not in the runner's default list.

## Enable

Add it to the `checks` list in `.fitnessrc.json`:

```json
{ "checks": ["no-eslint-disable"] }
```

## What passes

A source file with no disable directives:

```ts
// src/slug.ts
export function toSlug(input: string): string {
  return input.trim().toLowerCase().replace(/\s+/g, '-');
}
```

## What fails

Any of the directive forms — file-level, block, line, or next-line:

```ts
// src/legacy.ts
export const x = 1;
// eslint-disable-next-line no-console
console.log(x);
const y = 2; // eslint-disable-line
/* eslint-disable */
```

produces one error per hit, with the matched directive and its 1-based line number:

```
src/legacy.ts:3: eslint-disable-next-line
src/legacy.ts:5: eslint-disable-line
src/legacy.ts:6: eslint-disable
```

## Advanced

Detection is a plain line scan (grep-like), so the directive text is flagged even inside a string or unrelated comment:

```ts
// Flagged, even though it only describes the directive:
const hint = 'prefix a line with eslint-disable-next-line to skip a rule';
```

Reword the rare case rather than suppressing it. Scanned extensions are `.ts`, `.tsx`, `.js`, `.mjs`, `.cjs`, `.mts`, `.cts`, and test/spec files are included. The policy is zero-config and strict, with no allowlist.

## Behavior

- Walks the repo for files with each extension, combining and de-duplicating results; standard skip dirs (`node_modules`, `dist`, `coverage`, `.git`, and others) are excluded.
- Matches `eslint-disable(-next-line|-line)?` — file-level `eslint-disable`, block `eslint-disable ... eslint-enable`, `eslint-disable-line`, and `eslint-disable-next-line`.
- Pass: no scanned file contains a directive.
- Fail: `path/to/file.ts:42: <directive>` per hit.
- `filesChecked` counts every scanned source file (after de-duplication).
