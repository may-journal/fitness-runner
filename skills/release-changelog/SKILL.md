---
name: release-changelog
description: Fix a failing release-changelog check on the Release Please manifest and version.txt. Use when the two versions disagree or a file is missing.
---

# Match the release manifest and version file

When `.release-please-manifest.json` exists, its root `.` version and `version.txt` hold the same `major.minor.patch`. [The check README](../../go/cmd/fitness-check-release-changelog/README.md) states the rules and why `CHANGELOG.md` is not involved.

## Read the failure

- `.release-please-manifest.json is invalid JSON`: repair the JSON. An empty object `{}` passes during bootstrap.
- `.release-please-manifest.json must contain a root "." release version`: add the `"."` key with the current version.
- `.release-please-manifest.json root version must be numeric major.minor.patch (got "1.2")`: write three numbers, as `1.2.0`.
- `version.txt is missing; it must match release 1.3.0`: create `version.txt` holding that version.
- `version.txt version "1.2.0" must match .release-please-manifest.json version "1.3.0"`: set `version.txt` to the manifest's version.

## Check it first

Run `fitness release-changelog` from the repo root. Exit 0 means the versions match.
