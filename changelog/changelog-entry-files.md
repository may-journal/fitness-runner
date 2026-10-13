---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.13.1837

- Feat: changelog: a pull request may add its one entry as a `changelog/<name>.md` file, so open pull requests never conflict.
- Feat: release-pr: the release tool commits pending entry files into the release PR's changelog below the version section and deletes them.
- Feat: changelog, changelog-bullets, changelog-updated, the stamp, pr-check, and branch-check accept entry files and sections alike, one per pull request.
- Docs: the changelog README, AGENTS.md, and the PR template point to entry files first.
- Test: requirements 0017.14 to 0017.16, 0018.17, 0019.10 to 0019.12, 0063.5, and 0069.9 to 0069.13 prove both styles.
