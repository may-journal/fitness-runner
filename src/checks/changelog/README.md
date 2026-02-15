# changelog

Validates that the repo root has a `CHANGELOG.md` with at least one **dated section**: a heading `##` or `###` followed by `yyyy-mm-dd` (e.g. `### 2026-02-15` or `### 2026-02-15@10AM`).

## Behavior

- **Pass:** `CHANGELOG.md` exists and contains at least one line matching `## yyyy-mm-dd` or `### yyyy-mm-dd`.
- **Fail:** File missing → `missing root CHANGELOG.md`.
- **Fail:** File exists but no dated section → `CHANGELOG.md must have at least one dated section (## or ### yyyy-mm-dd)`.

## Contributing

This README is the canonical description for this check; `.cursor/rules/changelog.mdc` points Cursor here. This check is a self-contained sub-project. To add options (e.g. custom filename, required format), extend the check and its tests here and keep the README in sync.
