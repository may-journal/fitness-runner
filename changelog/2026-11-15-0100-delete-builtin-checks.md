### 2026.11.15.0100

> The engine holds no check code: it runs the checks a repo installs from fitness-checks, and judges `local-branch` itself.

- Refactor: the 68 moved check packages, their check-only helpers, and the 79 requirement docs and tests that now live in fitness-checks, are deleted.
- Feat: only `local-branch` stays built in; an unset `checkVersions` pins each released check, and a failed check install fails the run.
- Refactor: `fitness-release checks` no longer writes check READMEs; fitness-checks generates them, and the check index lists only `local-branch`.
- Test: the requirement tests, smoke run, CI and repo hooks install every check first; gh stand-ins probe a local API host.
- Docs: links to a moved check's README point at fitness-checks.
