# semantic-commit

Validates that the repository’s **HEAD commit** message follows [Conventional Commits](https://www.conventionalcommits.org/): `type(scope): description`, or a merge commit (`Merge ...`).

## Allowed types

`feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`.

Scope is required (e.g. `feat(api): add endpoint`). Merge commits are always accepted.

## Behavior

- **Pass:** `git log -1 --pretty=%B` subject matches `type(scope): description` or `Merge ...`.
- **Fail:** Subject doesn’t match → error with suggested format and allowed types.
- **Pass (no repo / git error):** Treated as pass so the check doesn’t block in non-git contexts.

## Contributing

This check is a self-contained sub-project. `semantic.ts` holds the parsing logic; the check in `index.ts` reads HEAD and reports. To add types or relax rules, extend `semantic.ts` and the tests here and keep the README in sync.
