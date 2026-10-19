---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.19.1505

- Fix: Auto-merge passes with a notice on a stacked pull request, since GitHub has no auto-merge for stacks.
- Fix: once the base merges and the pull request targets the default branch, its next push turns auto-merge on.
- Test: requirements 0091.1 and 0091.2 prove both cases.
- Docs: the workflows README explains stacked pull requests.
