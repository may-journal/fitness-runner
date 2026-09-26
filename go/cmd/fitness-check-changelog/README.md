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

## Contributing

This README is the canonical description for this check. This check is a self-contained sub-project (the `fitness-check-changelog` binary). To add options (e.g. custom filename, required format), extend the check and its tests here and keep the README in sync.
