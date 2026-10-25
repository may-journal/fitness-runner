---
name: no-eslint-disable
description: Fix a failing no-eslint-disable check on source files. Use when a file contains an eslint-disable directive.
---

# Remove ESLint disable directives

No source file carries an ESLint disable directive. [The check README](../../go/cmd/fitness-check-no-eslint-disable/README.md) shows what passes and fails, and the [fix-the-work ADR](../../docs/architecture/adr/0001-fix-the-work-not-the-limit.md) explains why.

## Read the failure

- `src/legacy.ts:3: eslint-disable-next-line`: delete the directive, then run eslint and fix the code the rule flags.
- `src/legacy.ts:5: eslint-disable-line` and `src/legacy.ts:6: eslint-disable` get the same fix.

The scan is a plain text match, so a string or comment that merely mentions a directive also fails. Reword that text. Files with `.ts`, `.tsx`, `.js`, `.mjs`, `.cjs`, `.mts` and `.cts` extensions are scanned, tests included.

## Check it first

Run `fitness no-eslint-disable` from the repo root. Exit 0 means no directive remains. Then run [eslint](../eslint/SKILL.md) to see the findings the directive hid.
