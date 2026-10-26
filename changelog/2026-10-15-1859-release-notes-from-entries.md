### 2026.10.15.1859

- Feat: `fitness-release release-notes` writes each draft release's description from the entry files added since the previous tag, before GoReleaser publishes it.
- Feat: `release-pr` links the entry files and compare view and no longer folds them into `CHANGELOG.md`; Release Please skips the file.
- Fix: `changelog`, `changelog-bullets`, `changelog-updated`, `release-changelog`, and `pr-check` ask nothing of `CHANGELOG.md` or the package version stamp.
- Feat: commit and PR checks fail an edit to a released entry file, and a shallow clone fails the notes.
- Docs: ADR 0007 freezes `CHANGELOG.md` at 1.14.0; requirements 0017, 0018, 0019, 0054, and 0069 gain real-run acceptances for each rule.
