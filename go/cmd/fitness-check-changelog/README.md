---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# changelog

The repo root must have a `CHANGELOG.md`. Every `###` heading must be a dated section matching the package version style: `### yyyy.mm.dd.HHMM` (e.g. `### 2026.02.15.1100`). When `package.json` exists, the first (latest) CHANGELOG heading timestamp must match its version suffix (e.g. `0.1.0-2026.02.15.1100`). The `package-lock.json` version must match the `package.json` version.

## Behavior

- Pass: `CHANGELOG.md` exists, every `###` line matches `### yyyy.mm.dd.HHMM`, and (if present) package.json version suffix and package-lock version match the first heading.
- Fail: File missing → `missing root CHANGELOG.md`.
- Fail: No `###` heading → must have at least one `### yyyy.mm.dd.HHMM` section.
- Fail: Any `###` heading not in that format → `every ### heading must be ### yyyy.mm.dd.HHMM (invalid: "...")`.
- Fail: package.json version suffix ≠ first CHANGELOG heading → `package.json version suffix must match CHANGELOG.md first ### heading (yyyy.mm.dd.HHMM)`.
- Fail: package-lock.json version ≠ package.json version → `package-lock.json version must match package.json version`.

## Entry files

Every pull request that inserts its entry at the top of `CHANGELOG.md` conflicts with every other open one. So a pull request adds its entry as its own `changelog/<name>.md` file, named for the change in kebab-case. It holds one `### yyyy.mm.dd.HHMM` heading and its bullets, with optional front matter above.

- A pull request adds exactly one entry file. Adding or editing a `CHANGELOG.md` section fails, pointing to `changelog/`.
- A repo with no `changelog/` folder needs no setting: its first pull request creates it.
- This check folds every entry file into `CHANGELOG.md`, newest first, and judges the result, so the newest entry is the first heading.
- Fail: an entry file holding other than one dated `###` heading → `changelog/<name>.md must hold one ### yyyy.mm.dd.HHMM heading...`.
- `changelog-updated`, the pre-commit stamp, `pr-check`, and `branch-check` accept only entry files.
- On the release PR, `fitness-release release-pr` commits the entry files into `CHANGELOG.md` below the Release Please version section and deletes them.

Dated sections already in `CHANGELOG.md` stay valid history, which this check and `changelog-bullets` still judge.

## Contributing

This README is the canonical description for this check. This check is a self-contained sub-project (the `fitness-check-changelog` binary). To add options (e.g. custom filename, required format), extend the check and its tests here and keep the README in sync.
