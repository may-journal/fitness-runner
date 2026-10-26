---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.26.2005
- Feat: the new `requirements-changed` check fails a commit or pull request that changes product code without a changed test and requirement doc.
- Feat: docs, config, changelog, renames, whitespace edits, replays, and pure deletions still pass; a deletion that leaves its requirement doc behind fails.
- Feat: a `Requirements: none, <reason>` commit trailer is the one exception, so the commit-msg hook now judges requirements-changed.
