# 0007 — Release notes come from entry files

Release notes come only from `changelog/` entry files.

https://github.com/may-journal/fitness-runner/issues/859

## Check

- Fails `CHANGELOG.md` when its newest release heading is other than 1.14.0, where it froze.
- Fails `release-please-config.json` without `"skip-changelog": true`.
- Fix: add an entry file under `changelog/` and leave `CHANGELOG.md` as it is.
