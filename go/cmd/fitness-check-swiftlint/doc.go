// Command fitness-check-swiftlint runs SwiftLint over the repo through the
// Swift language module, which owns the tool run and its judgment, via
// `swiftlint lint --strict`. [SwiftLint] is a system binary. It is on by
// default.
//
// # Behavior
//
//   - Runs: `swiftlint lint --strict --reporter json --quiet .` from repo root.
//   - Pass: no violations, reporting the Swift files it is asked to lint (a pull request counts only its changed files).
//   - Fail: any violation — reported as `file:line[:col] - reason (rule_id)`.
//   - No Swift files in the repo: passes clean (`filesChecked: 0`) instead of erroring, so it is safe in every repo.
//   - SwiftLint is not an npm package but a system binary (`brew install swiftlint`), resolved from PATH at runtime, never `node_modules`.
//   - A missing binary reports info with a one-line install hint rather than a stack trace.
//   - A run that could not start, was killed, or exits non-zero without a violation reports info quoting its output.
//
// # SwiftLint config
//
// It uses whatever `.swiftlint.yml` is present in the repo root — the same
// "consumer config wins" convention as the other checks. No shared-config
// fallback ships for this one yet.
//
// [SwiftLint]: https://github.com/realm/SwiftLint
package check
