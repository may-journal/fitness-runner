---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# changelog

The repo root must have a `CHANGELOG.md`. Every `###` heading must be a dated section matching the package version style: `### yyyy.mm.dd.HHMM` (e.g. `### 2026.02.15.1100`). When `package.json` exists, its version must end in the first (latest) heading's timestamp. The `package-lock.json` version must match the `package.json` version.

The timestamp follows `+` as build metadata (`0.1.0+2026.02.15.1100`) or `-` as a prerelease (`0.1.0-2026.02.15.1100`). Prefer `+`. Semver forbids leading zeros in prerelease numbers, so npm rejects `0.1.0-2026.02.15.0900` but accepts the build metadata form.

## Behavior

- Pass: `CHANGELOG.md` exists, every `###` line matches `### yyyy.mm.dd.HHMM`, and (if present) package.json version suffix and package-lock version match the first heading.
- Fail: File missing → `missing root CHANGELOG.md`.
- Fail: No `###` heading → must have at least one `### yyyy.mm.dd.HHMM` section.
- Fail: Any `###` heading not in that format → `every ### heading must be ### yyyy.mm.dd.HHMM (invalid: "...")`.
- Fail: package.json version suffix ≠ first CHANGELOG heading → `package.json version suffix must match CHANGELOG.md first ### heading (yyyy.mm.dd.HHMM)`.
- Fail: package-lock.json version ≠ package.json version → `package-lock.json version must match package.json version`.

## Contributing

This README is the canonical description for this check. This check is a self-contained sub-project (the `fitness-check-changelog` binary). To add options (e.g. custom filename, required format), extend the check and its tests here and keep the README in sync.
