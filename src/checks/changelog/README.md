# changelog

Validates that the repo root has a `CHANGELOG.md` and that **every** `###` heading is a dated section matching the package version style: `### yyyy.mm.dd.HHMM` (e.g. `### 2026.02.15.1100`).

## Behavior

- **Pass:** `CHANGELOG.md` exists and every `###` line matches `### yyyy.mm.dd.HHMM`.
- **Fail:** File missing → `missing root CHANGELOG.md`.
- **Fail:** No `###` heading → must have at least one `### yyyy.mm.dd.HHMM` section.
- **Fail:** Any `###` heading not in that format → `every ### heading must be ### yyyy.mm.dd.HHMM (invalid: "...")`.

## Contributing

This README is the canonical description for this check; `.cursor/rules/changelog.mdc` points Cursor here. This check is a self-contained sub-project. To add options (e.g. custom filename, required format), extend the check and its tests here and keep the README in sync.
