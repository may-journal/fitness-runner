### 2026.11.15.1658

> close-check installs the checks a repo pins again, so it judges a closed issue instead of reopening it with no result.

- Fix: the close-check reusable workflow checks out the repo first, so `with-checks: pinned` finds its `.fitnessrc.json`.
- Test: requirement 0167 fails when a reusable job runs a fitness command without a checkout first.
- Chore: the other reusable workflows that run a fitness command already check out first.
