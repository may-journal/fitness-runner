---
relatedConfigurations: ['../../../package.json']
---

# no-eslint-disable

Fails when any source file contains an ESLint disable directive, keeping lint hygiene on the fix-the-rule path instead of accumulating suppressions. Opt-in — not part of `defaultChecks`.

## Enable

```ts
// .fitnessrc.ts
export default { checks: ['no-eslint-disable'] };
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

Detection is a plain line scan (grep-like), so the directive text is flagged even when it appears inside a string or an unrelated comment:

```ts
// Flagged, even though it only describes the directive:
const hint = 'prefix a line with eslint-disable-next-line to skip a rule';
```

Reword the rare case rather than suppressing it. Scanned extensions are `.ts`, `.tsx`, `.js`, `.mjs`, `.cjs`, `.mts`, `.cts`, and test/spec files are included — the policy is zero-config and strict, with no allowlist.

## Behavior

- Discovers each extension via `findFilesByExtension`, combining and de-duplicating the results; standard skip dirs (`node_modules`, `dist`, `coverage`, `.git`, and the other runner skip dirs) are excluded.
- Matches `eslint-disable(-next-line|-line)?` — file-level `eslint-disable`, block `eslint-disable ... eslint-enable`, `eslint-disable-line`, and `eslint-disable-next-line`.
- Pass: no scanned file contains a directive.
- Fail: `path/to/file.ts:42: <directive>` per hit.
- `filesChecked` counts every scanned source file (after de-duplication).
