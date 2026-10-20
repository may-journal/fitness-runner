---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.20.1839

- Fix: release PR links resolve before the release, since they point at the PR's head commit.
- Fix: the release PR links each entry file added since the previous tag, not the whole `changelog/` folder.
- Fix: the release tool reads those entries at the head commit, fetching it when the checkout lacks it.
- Test: requirement 0069.21 checks every link the release tool writes resolves in a repo without the new tag.
- Docs: the CI guide and the release tool README describe the new links.
