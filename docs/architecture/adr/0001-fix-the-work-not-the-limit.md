---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# 0001 — Fix the work, not the limit: checks earn their timeout budget

## Context

The `eslint` check timed out in CI — 5.4s against the 5s per-check budget — and the reflex was to raise the timeout. That hid the cause: the check built a full TypeScript type-checker program over the whole repo, yet no enabled rule reads type information. It paid ~5s for zero findings; parsing syntactically was ~4x faster with byte-identical output. The breach was waste, not work that needed more room.

## Decision

1. The per-check timeout is a budget, not a suggestion; a breach means wasted work, so cut the work, not the limit. (KISS)
2. Raise a check's `timeoutMs` only for irreducible external latency — a network or registry round-trip, or a subprocess floor, never local computation.
3. A tool runs only the analysis its active rules need: ESLint parses syntactically; a type-checker program is added and removed with type-aware rules.
4. When two paths run the same analysis — the in-process check and the `fitness-shared lint` CLI — they share one config, so the change lands once.

## Consequences

1. "Slow but correct" is a bug in the check, not a knob to turn; new checks are written to fit the budget.
2. Every `timeoutMs` override now names an external cost, so a reviewer can challenge any that does not.
3. Re-introducing a type-aware ESLint rule is a deliberate act with a known price (~4-5x lint time), overriding the syntactic default.
4. The shared lint config is one point of change and failure; a mistake hits both lint paths, caught by this repo checking itself.
