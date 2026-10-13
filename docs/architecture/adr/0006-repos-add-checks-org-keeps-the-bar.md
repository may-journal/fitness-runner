---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# 0006 — Repos add local checks; only external policy replaces or disables a bundled one

## Context

A repo had no designed way to run its own rules through fitness. Org repos also cannot set `ignore` or turn a check off, since [ADR 0001](./0001-fix-the-work-not-the-limit.md) fixes the work, not the bar.

## Decision

1. `localChecks` in `.fitnessrc.json` names one directory inside the repo. Each executable `fitness-check-<name>` in it is a local check.
2. Lookup searches that directory before the bundled checks, so a local check with a bundled check's name replaces it.
3. A local check follows the [check protocol](../../check-protocol.md), like a bundled one.
4. A new name joins the suite after the bundled checks, under either policy.
5. Org policy refuses to replace a bundled check. A replacement could pass anything, which is the same as turning the check off.
6. External policy allows replacements, since that repo owns its own bar. Each run names the bundled checks it replaced.
7. A missing, empty, or outside directory, or a check that cannot run, fails the run before any check starts.
8. `disabledChecks` turns checks off under external policy only. Org policy refuses it, for the reason in item 5.

## Consequences

1. Repos grow their own rules without forking fitness, and hooks and CI run them alike.
2. An org repo that needs a stricter version of a bundled check adds it under a new name, or proposes the change upstream.
3. A local check is tracked code, so it is judged by the same checks as everything else.
4. GitHub blob commands, such as `pr-check`, still run only the bundled checks.
5. `disabledChecks` leaves the retired exclusion keys; `ignore` and `skipTheseDirectories` still fail everywhere.
