---
relatedConfigurations: ['../../../package.json']
---

# repeated-string-literals

Fails when the same string literal appears 3+ times across source files — a signal it should be a single shared constant (and, when the repeats are a small closed set like `'active'` / `'pending'`, a union type or enum). Opt-in — not part of `defaultChecks`.

## Why not just ESLint / `jscpd`?

- `jscpd` finds duplicated multi-line blocks (`--min-lines 5`); a single repeated literal is one token on one line, so it never sees it.
- `sonarjs/no-duplicate-string` (ESLint) is close but, by design, only flags strings that are ≥10 chars and contain a separator (its `NO_SEPARATOR_REGEXP`/`MIN_LENGTH`). That deliberately excludes exactly the short, identifier-like tokens (`'active'`, `'GET'`, `'user_id'`) this check targets.

## Enable

```ts
// .fitnessrc.ts
export default { checks: ['repeated-string-literals'] };
```

## What passes

Distinct values, or a value repeated only twice:

```ts
// src/order.ts
const status = 'pending';
const other = 'shipped';
const again = 'pending'; // 2 uses — under the threshold
```

## What fails

A value used 3+ times, anywhere across the scanned files:

```ts
// src/a.ts
if (order.status === 'active') {
  /* ... */
}
setStatus('active');

// src/b.ts
const ACTIVE_ROWS = rows.filter((r) => r.status === 'active');
```

produces one error per duplicated value, most-repeated first:

```
"active" appears 3 times (src/a.ts:2, src/a.ts:3, src/b.ts:2) — extract a shared constant
```

## Behavior

- Scans `.ts`, `.tsx`, `.js`, `.mjs`, `.cjs`, `.mts`, `.cts` via `findFilesByExtension` (skips `node_modules`, `dist`, `coverage`, `.git`, …). Test/spec/bench files are excluded — fixtures repeat strings on purpose.
- A small hand lexer (no parser dependency, matching the other checks) extracts single- and double-quoted literals while ignoring line/block comments, regex literals, and template literals, and dropping module specifiers (the string after `import` / `require` / `from`).
- Counts identical values repo-wide; flags any at or above `MIN_OCCURRENCES` (3). Values shorter than `MIN_LENGTH` (3) are ignored as noise.
- Idiomatic tokens are never flagged (`IDIOMATIC_VALUES`): buffer encodings (`'utf8'`, `'base64'`, …), `child_process` stdio modes (`'inherit'`, `'pipe'`, …), `typeof` results (`'object'`, `'string'`, …), and language directives (`'use strict'`). For these closed sets the literal is the clearest spelling — a constant would hurt readability (or, for a directive prologue, is impossible).
- Error format: `"value" appears N times (file:line, …, +K more) — extract a shared constant`. Locations are capped at `MAX_LOCATIONS` (5) per value.
- `filesChecked` counts every scanned source file.

## Configuration

A project baseline for the rare repeat no constant can fix: a value that must live once per runtime island (TS sources vs. published plain-JS bin scripts vs. repo automation scripts — islands that cannot share one constants module), or one spelling shared by genuinely different things (a CLI binary name vs. a check name). Everything else is a real duplicate — extract it instead:

```js
// .fitnessrc.js
module.exports = {
  checks: ['repeated-string-literals'],
  repeatedStringLiterals: {
    allow: ['my-check-name', 'kindValue'],
  },
};
```

`allow` entries match exact string values and are never flagged. Keep the list a shrinking baseline — if a repeat is a genuine, justified constant, the fix is to centralize it, which is the point.

## Limitations (v1, KISS)

- Template literals are out of scope — only plain `'…'` / `"…"` literals are counted.
- Thresholds and the idiomatic-token set are fixed constants (exported from the module); there is no inline `// fitness:allow-literal` marker and no enum-candidate detection yet (both tracked in #44).
