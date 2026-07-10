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

- Scans `.ts`, `.tsx`, `.js`, `.mjs`, `.cjs`, `.mts`, `.cts` via `findFilesByExtension` (skips `node_modules`, `dist`, `coverage`, `.git`, …). Test/spec files are excluded — fixtures repeat strings on purpose.
- A small hand lexer (no parser dependency, matching the other checks) extracts single- and double-quoted literals while ignoring line/block comments, regex literals, and template literals, and dropping module specifiers (the string after `import` / `require` / `from`).
- Counts identical values repo-wide; flags any at or above `MIN_OCCURRENCES` (3). Values shorter than `MIN_LENGTH` (3) are ignored as noise.
- Idiomatic tokens are never flagged (`IDIOMATIC_VALUES`): buffer encodings (`'utf8'`, `'base64'`, …), `child_process` stdio modes (`'inherit'`, `'pipe'`, …), and `typeof` results (`'object'`, `'string'`, …). For these closed sets the literal is the clearest spelling — a constant would hurt readability.
- Error format: `"value" appears N times (file:line, …, +K more) — extract a shared constant`. Locations are capped at `MAX_LOCATIONS` (5) per value.
- `filesChecked` counts every scanned source file.

## Limitations (v1, KISS)

- Template literals are out of scope — only plain `'…'` / `"…"` literals are counted.
- Thresholds and the idiomatic-token set are fixed constants (exported from the module); there is no `.fitnessrc` config or per-repo allowlist yet. If a repeat is a genuine, justified constant, the fix is to centralize it — which is the point.
