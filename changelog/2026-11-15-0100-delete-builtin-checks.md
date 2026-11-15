### 2026.11.15.0100

> The engine holds no check code: it runs the checks a repo installs from fitness-checks, and judges `local-branch` itself.

- Refactor: the 68 moved check packages, their check-only helpers, and the 79 requirement docs and tests that now live in fitness-checks, are deleted.
- Feat: the built-ins map holds only `local-branch`, and an unset `checkVersions` installs every check the fitness-checks release index lists, at its current major.
- Refactor: `fitness-release checks` no longer writes check READMEs; fitness-checks generates them, and the check index lists only `local-branch`.
- Test: the requirement tests, through `tests/install-checks.sh`, the release smoke run and this repo's hooks install every check first.
- Docs: links to a moved check's README point at fitness-checks.
