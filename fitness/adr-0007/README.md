# ADR 0007 check

Keeps [ADR 0007](../../docs/architecture/adr/0007-release-notes-from-entry-files.md): release notes come only from `changelog/` entry files.

- Fails `CHANGELOG.md` when its newest release heading is other than 1.14.0, where it froze.
- Fails `release-please-config.json` without `"skip-changelog": true`.
- Fix: add an entry file under `changelog/` and leave `CHANGELOG.md` as it is.
