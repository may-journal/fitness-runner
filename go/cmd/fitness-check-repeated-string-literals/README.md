---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# repeated-string-literals

Fails when the same string literal appears 3+ times across source files — a signal it should be a shared constant. When the repeats are a small closed set like `'active'` / `'pending'`, a union type or enum fits better. Opt-in.

## Why not just ESLint / `jscpd`?

- `jscpd` finds duplicated multi-line blocks (`--min-lines 5`); a single repeated literal is one token on one line, so it never sees it.
- `sonarjs/no-duplicate-string` (ESLint) only flags strings that are 10+ chars and contain a separator, which excludes exactly the short, identifier-like tokens (`'active'`, `'GET'`) this check targets.

## Enable

```json
{ "checks": ["repeated-string-literals"] }
```

## What passes / fails

A value repeated only twice passes; a value used 3+ times anywhere across the scanned files fails, one error per value, most-repeated first:

```ts
// src/a.ts
if (order.status === 'active') setStatus('active');
// src/b.ts
const rows = all.filter((r) => r.status === 'active');
```

```
"active" appears 3 times (src/a.ts:1, src/a.ts:1, src/b.ts:2) — extract a shared constant
```

## Behavior

- Scans `.ts`, `.tsx`, `.js`, `.mjs`, `.cjs`, `.mts`, `.cts` (skipping `node_modules`, `dist`, `coverage`, `.git`). Test, spec, and bench files are excluded — fixtures repeat strings on purpose.
- A hand lexer extracts single- and double-quoted literals while ignoring comments, regex literals, and template literals, and dropping module specifiers (the string after `import` / `require` / `from`).
- Counts identical values repo-wide; flags any at or above the threshold (3). Values under the length floor (3) are ignored.
- Idiomatic tokens are never flagged: buffer encodings (`'utf8'`), stdio modes (`'inherit'`), `typeof` results (`'object'`), and directives (`'use strict'`) — a constant would hurt readability there.
- Error format: `"value" appears N times (file:line, …, +K more) — extract a shared constant`, locations capped at 5.
- `filesChecked` counts every scanned source file.

## Configuration

Use `allow` for the rare repeat no constant can fix. That means a value that must live once per runtime island (TS sources vs. published bin scripts vs. automation scripts), or one spelling shared by genuinely different things:

```json
{
  "checks": ["repeated-string-literals"],
  "repeatedStringLiterals": { "allow": ["my-check-name", "kindValue"] }
}
```

`allow` entries match exact values and are never flagged. Keep the list a shrinking baseline — the fix for a genuine constant is to centralize it.

## Limitations (v1)

- Template literals are out of scope — only plain `'…'` / `"…"` literals count.
- Thresholds and the idiomatic-token set are fixed in the binary; there is no inline marker and no enum-candidate detection yet (both tracked in #44).
