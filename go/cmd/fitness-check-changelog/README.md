---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# changelog

Judges the `changelog/` entry files, the only changelog source. Each must hold one dated `### yyyy.mm.dd.HHMM` heading and its bullets, and its name must start with that heading's date. `CHANGELOG.md` is not judged, so a repo may keep, freeze, or drop it, and an upstream fork's `### Added` history passes as it is. The package version is not judged either.

## Behavior

- Pass: every entry file holds one dated heading and carries its date in its name.
- Pass with zero files: a repo with no entry files, with or without `CHANGELOG.md`.
- Fail: an entry file holding other than one dated `###` heading → `changelog/<file>.md must hold one ### yyyy.mm.dd.HHMM heading...`.
- Fail: an entry file whose name lacks its heading's date, or carries another → `changelog/<file>.md must be named changelog/<yyyy-mm-dd-HHMM>-<change>.md...`.

## Entry files

A pull request adds its entry as its own `changelog/<yyyy-mm-dd-HHMM>-<change>.md` file, with the change in kebab-case. So two open pull requests never edit the same lines. It holds one `### yyyy.mm.dd.HHMM` heading and its bullets, with optional front matter above. The name's date is that heading's, dots read as hyphens, so the folder lists in release order.

- A pull request adds exactly one entry file, and later commits add bullets to it.
- A repo with no `changelog/` folder needs no setting: its first pull request creates it.
- `changelog-updated`, `pr-check`, and `branch-check` accept only entry files, and fail an added one with a bad name.
- You write the heading's time by hand; no hook rewrites it.
- Entry files stay in the folder after a release.
- An entry file a release tag holds is final. Checks fail an edit or deletion of it, asking for a new entry file.
- Each release tag's GitHub description lists the entry files added since the previous tag, newest first, as `fitness-release release-notes` writes it.

## Contributing

This README is the canonical description for this check. This check is a self-contained sub-project (the `fitness-check-changelog` binary). To add options (e.g. custom folder, required format), extend the check and its tests here and keep the README in sync.
