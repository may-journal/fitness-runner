---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.18.2345

- Perf: the pre-push gate reads each Plan's labels and comments in one `gh issue view`, once per push.
- Perf: `local-branch` judges the Plans the gate already read, instead of reading them from GitHub again.
- Perf: linked issues are read side by side, and `gh repo view` runs alongside the open PR read.
- Perf: the gate reads every pushed commit's subject with one `git log --no-walk`.
