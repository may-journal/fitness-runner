---
relatedConfigurations: ['../../../package.json']
---

# gitignore-why

Requires every `.gitignore` ignore pattern to be immediately preceded by a `#` comment explaining why it exists, so ignores are never unexplained and stay easy to audit and onboard. Opt-in — not part of `defaultChecks`. Add `'gitignore-why'` to `.fitnessrc` `checks` to enable it.

## Behavior

- Reads `<root>/.gitignore`. When the file is absent, passes with `filesChecked` 0.
- Classifies each line as blank, a `#` comment, or a pattern (any other non-blank line, including `!` negations and `dir/` rules).
- Pass: every pattern line is directly preceded by a non-empty explanatory `#` comment.
- Fail: a pattern line whose line above is blank, another pattern, a bare `#`, or nothing at all (first line). Reports one error per violation: `.gitignore:<lineNumber>: pattern "<text>" has no explanatory # comment on the line above`.
- `filesChecked` is 1 whenever `.gitignore` is present.

## Notes

- Strict rule: one WHY comment per pattern line, on the line directly above it. A single comment does not cover multiple patterns.
