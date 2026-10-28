# 0007 — Release notes come from entry files

Release notes come only from `changelog/` entry files.

## Context

Entry files fed a shared CHANGELOG.md, which three checks guarded and the release PR folded. Releases only linked it, and a fork's upstream CHANGELOG.md failed every rule.

## Decision

1. Per-PR `changelog/` entry files are the only changelog source. They stay in the folder after a release. (KISS)
2. While a release is a draft, `fitness-release release-notes` writes its description from the entry files added since the previous tag.
3. An entry file a release tag holds is final; checks fail an edit to it.
4. Release Please sets `skip-changelog`, and no release step writes or folds CHANGELOG.md.
5. No check reads CHANGELOG.md: no dated headings, no release-only edits, and no package.json version stamp.
6. fitness-runner freezes its CHANGELOG.md at 1.14.0, pointing to GitHub Releases. Removing it would break links to its sections from older PRs.

## Consequences

1. Notes live where people read a release, written once by the PR that made each change.
2. A consumer may keep, edit, or drop its CHANGELOG.md, so an upstream fork passes as it is.
3. Retiring the version stamp also retires the package-lock.json match, which only that stamp needed.
4. The release PR body changes once, with no fold commit, which may shorten its race with Release Please.
5. The entry folder grows with every release; git history and tags decide which entries a release ships.

## Check

- Fails `CHANGELOG.md` when its newest release heading is other than 1.14.0, where it froze.
- Fails `release-please-config.json` without `"skip-changelog": true`.
- Fix: add an entry file under `changelog/` and leave `CHANGELOG.md` as it is.
