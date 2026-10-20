---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.20.1516

- Fix: Auto-merge comments once on a stacked pull request saying how to merge it, and edits that comment once auto-merge is on.
- Fix: Auto-merge always runs when a pull request is reopened, since closing it turns auto-merge off.
- Fix: Auto-merge fails with an error naming GitHub's answer when it refuses auto-merge for any reason but stacking.
- Docs: the workflows README explains how to rerun a cancelled pr-check without closing the pull request.
- Test: requirement 0091 covers the comment, a refused comment, its update, the reopen, and the loud failure.
