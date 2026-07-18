---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0001 — Fix the work, not the limit: checks earn their timeout budget

## Context

The `eslint` check timed out in CI — 5.4s against the 5s per-check budget — and the reflex was to
raise the timeout. That hid the cause: the check built a full TypeScript type-checker program over
the whole repo, yet not one enabled rule reads type information. It paid ~5s for zero findings.
Parsing syntactically instead was ~4x faster with byte-identical output. The breach was waste, not
work that needed more room.

## Decision

1. The default per-check timeout is a budget, not a suggestion. A check that breaches it is presumed
   to be doing needless work, and the fix is to cut the work, not widen the limit. (KISS)
2. Raise a check's `timeoutMs` only for irreducible external latency it cannot avoid — a network or
   registry round-trip (`dependency-currency`), or a subprocess with its own floor. Never for local
   computation.
3. A tool does the minimum analysis its active rules require. ESLint parses syntactically; a
   type-checker program (`project` / `projectService`) is turned on only when a type-aware rule
   needs it, and turned off when the last such rule goes.
4. When two paths run the same analysis — the in-process check and the `fitness-shared lint` CLI —
   they share one config, so a change like this lands once.

## Consequences

1. "Slow but correct" is treated as a bug in the check, not a knob to turn. New checks are written
   to fit the budget.
2. Every `timeoutMs` override now names an external cost, so a reviewer can challenge any that does
   not.
3. Re-introducing a type-aware ESLint rule is a deliberate act with a known price (~4-5x lint time);
   the syntactic default is the thing it must override, not an accident to discover.
4. The shared lint config is a single point of change and of failure. A mistake there hits both lint
   paths at once — caught by this repo running the full check suite on itself.
