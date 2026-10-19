---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.19.2100

- Feat: repos that opt in to releases call one reusable Release Please workflow, keeping their own config, manifest and version file.
- Test: requirements 0092.1 to 0092.4 prove the notes shaping, the tracking issue link, a release with no pull request, and the major tag move.
- Fix: the tracking issue lookup searches by title and stops on a failed call.
- Fix: the major tag step fails on a lookup error other than a missing tag.
- Docs: the workflows README explains the release caller.
