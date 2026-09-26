---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# swiftlint

Runs [SwiftLint](https://github.com/realm/SwiftLint) via `swiftlint lint --strict`. Opt-in — not in the runner's default list. Add `"swiftlint"` to the `checks` array in `.fitnessrc.json` to enable it.

## Behavior

- Runs: `swiftlint lint --strict --reporter json --quiet .` from repo root.
- Pass: No violations.
- Fail: Any violation — reported as `file:line[:col] - reason (rule_id)`.
- No Swift files in the repo: passes clean (`filesChecked: 0`) instead of erroring — safe to enable in a mostly-JS/TS repo.
- SwiftLint is not an npm package but a system binary (`brew install swiftlint`), resolved from PATH at runtime, never `node_modules`.
- A missing binary fails clearly with a one-line install hint rather than a stack trace.

## Config

Uses whatever `.swiftlint.yml` is present in the repo root — the same "consumer config wins" convention as the other checks. No shared-config fallback ships for this one yet.
