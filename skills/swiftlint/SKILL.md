---
name: swiftlint
description: Fix a failing swiftlint check on Swift code. Use when SwiftLint reports a violation or is not installed.
---

# Fix SwiftLint violations

Swift code passes `swiftlint lint --strict`, which counts warnings as failures. [The check README](../../go/cmd/fitness-check-swiftlint/README.md) states the command and config.

## Read the failure

- `Sources/App.swift:12:5 - Line should be 120 characters or less (line_length)`: the form is `file:line[:col] - reason (rule_id)`. Change the code at that line to satisfy the named rule.
- `SwiftLint not installed: brew install swiftlint`: run the command shown.
- `swiftlint reported an error (run: swiftlint lint --strict)`: run that command and read its output.

The rules come from `.swiftlint.yml` at the repo root. Fix the code to meet them.

## Check it first

Run `fitness swiftlint` from the repo root. Exit 0 means no violations.
