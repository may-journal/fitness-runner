---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.15.1859

- Feat: `fitness-release release-notes` writes each tagged release's description from the entry files added since the previous tag.
- Feat: `release-pr` links the entry files and compare view and no longer folds them into `CHANGELOG.md`; Release Please skips the file.
- Fix: `changelog`, `changelog-bullets`, `changelog-updated`, `release-changelog`, and `pr-check` ask nothing of `CHANGELOG.md` or the package version stamp.
- Docs: ADR 0007 freezes `CHANGELOG.md` at 1.14.0 with a pointer to GitHub Releases.
- Test: requirements 0017.22-24, 0018.18-19, 0019.19-20, 0054.11-12, and 0069.20-22 replace the retired `CHANGELOG.md` acceptances.
