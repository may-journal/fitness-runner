---
name: repeated-string-literals
description: Fix a failing repeated-string-literals check on TypeScript and JavaScript sources. Use when a string literal appears three or more times.
---

# Share a repeated string literal

A string literal of 3 or more characters appears at most twice across the scanned sources. [The check README](../../README.md) lists what is scanned and what is never flagged.

## Read the failure

- `"active" appears 3 times (src/a.ts:1, src/a.ts:1, src/b.ts:2) — extract a shared constant`: define the value once and import it where it is used. Locations are capped at 5, followed by `+K more`.
- When the repeats form a small closed set such as `'active'` and `'pending'`, use a union type or enum.

Tests count too. Share a test constant or fixture instead of retyping the value. Do not add the value to the `allow` list.

## Check it first

Run `fitness repeated-string-literals` from the repo root. Exit 0 means no value repeats three times.
