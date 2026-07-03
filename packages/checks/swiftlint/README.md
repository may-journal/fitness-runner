---
relatedConfigurations: ['../../../package.json']
---

# swiftlint

Runs [SwiftLint](https://github.com/realm/SwiftLint) via `swiftlint lint --strict`. Opt-in — not part of `defaultChecks`. Add `'swiftlint'` to `.fitnessrc` `checks` to enable it.

## Behavior

- Runs: `swiftlint lint --strict --reporter json --quiet .` from repo root.
- Pass: No violations.
- Fail: Any violation — reported as `file:line[:col] - reason (rule_id)`.
- No Swift files in the repo: passes clean (`filesChecked: 0`) instead of erroring — safe to enable in a mostly-JS/TS repo.
- SwiftLint is not an npm package — it's a system binary (`brew install swiftlint`). Unlike every other check, it is not bundled as a dependency; a missing binary fails clearly with an install hint rather than a stack trace.

## Config

Uses whatever `.swiftlint.yml` is present in the repo root — same "consumer config wins" convention as the other checks, but there's no fitness-shared fallback shipped for this one yet.
