---
fitnessFunctions: ['semantic-commit']
relatedConfigurations: ['../../../.fitnessrc.json']
---

# semantic-commit

Validates that the repository’s HEAD commit message follows [Conventional Commits](https://www.conventionalcommits.org/): `type(scope): description`, or a merge commit (`Merge ...`).

## Allowed types

From [conventional-commit-types](https://github.com/commitizen/conventional-commit-types), inlined in the check binary.

Scope is required (e.g. `feat(api): add endpoint`). Merge commits are always accepted.

## Behavior

- Pass: `git log -1 --pretty=%B` subject matches `type(scope): description` or `Merge ...`.
- Fail: Subject doesn’t match → error with suggested format and allowed types.
- Fail (no repo / git error / empty message): Returns a no-message error so commit-msg hook and explicit `--message` runs get a clear signal.

This check declares a context-inline `--message` argument in its `--describe` metadata. From a Git commit-msg hook, pass the message string: `fitness --check=semantic-commit --message="$(cat "$1")"` so the check validates the proposed message instead of HEAD.

## Contributing

This README is the canonical description for this check. This check is a self-contained binary (`fitness-check-semantic-commit`). To add types or relax rules, extend the check and tests here and keep the README in sync.
