# 0001 — Fix the work, not the limit: two totals guard the run's time

Two totals guard the run's time, and a slow check gets its work cut.

## Context

The `eslint` check timed out in CI — 5.4s against the 5s per-check budget — and the reflex was to raise the timeout. That hid the cause: the check built a full TypeScript type-checker program over the whole repo, yet no enabled rule reads type information. It paid ~5s for zero findings; parsing syntactically was ~4x faster with byte-identical output. The breach was waste, not work that needed more room.

## Decision

1. No check has its own time limit. The fast checks share `checkkit.FastTotal`; a breach names the slowest, so cut their work. (KISS)
2. A check joins the test stage, timed by its own total, only when it runs a repo's test suite or build.
3. A tool runs only the analysis its active rules need: ESLint parses syntactically; a type-checker program is added and removed with type-aware rules.
4. When two paths run the same analysis — the in-process check and the `fitness-shared lint` CLI — they share one config, so the change lands once.

## Consequences

1. "Slow but correct" is a bug in the check, not a knob to turn; new checks are written to fit the budget.
2. Every test-stage check names the run it waits on, so a reviewer can challenge any that does not.
3. Re-introducing a type-aware ESLint rule is a deliberate act with a known price (~4-5x lint time), overriding the syntactic default.
4. The shared lint config is one point of change and failure; a mistake hits both lint paths, caught by this repo checking itself.

## Check

- Fails a built-in check in the test stage with no entry in [config.json](config.json) naming the run it waits on.
- Fails an entry in `config.json` whose check is no longer in the test stage.
- Fails a `timeoutMs` key in `.fitnessrc.json`, a limit per check the run no longer has.
- Fix: cut the work so the check stays fast, or add its entry with the test run it waits on.
