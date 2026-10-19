---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.19.1500

- Perf: issue-check and plan-check skip a title-only edit and a label that cannot change the verdict, at job level, billing nothing.
- Perf: comment-check skips known bots' comments at job level, and close-check skips a not planned or duplicate close.
- Perf: Auto-merge skips repeat pushes. Native stacks report success without asking GitHub for unsupported auto-merge.
- Perf: close-check, pr-check and Auto-merge cancel a superseded run for the same issue or PR.
- Fix: the release smoke test commits unsigned, so a signing setup the user's git config names cannot fail it.
