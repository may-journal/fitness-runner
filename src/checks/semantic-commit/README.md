---
relatedConfigurations: ["../../../package.json"]
---

# semantic-commit

Validates that the repository’s HEAD commit message follows [Conventional Commits](https://www.conventionalcommits.org/): `type(scope): description`, or a merge commit (`Merge ...`).

## Allowed types

From [conventional-commit-types](https://github.com/commitizen/conventional-commit-types).

Scope is required (e.g. `feat(api): add endpoint`). Merge commits are always accepted.

## Behavior

- Pass: `git log -1 --pretty=%B` subject matches `type(scope): description` or `Merge ...`.
- Fail: Subject doesn’t match → error with suggested format and allowed types.
- Pass (no repo / git error): Treated as pass so the check doesn’t block in non-git contexts.

From a Git commit-msg hook, run `fitness --check=semantic-commit "$1"` so the check validates the proposed message in that file instead of HEAD.

## Contributing

This README is the canonical description for this check; `.cursor/rules/semantic-commit.mdc` points Cursor here. This check is a self-contained sub-project. To add types or relax rules, extend the check and tests here and keep the README in sync.
