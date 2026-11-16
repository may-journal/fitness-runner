### 2026.11.16.1500

> A run downloads only the checks its files reach, so a Markdown-only commit fetches a few checks, not all 69.

- Feat: a suite run matches its files against each pinned version's index fields and fetches only matching checks in parallel, each SHA-256 verified.
- Feat: a full run fetches every pinned check, and a reached check that cannot be fetched is an info row naming it.
- Feat: `fitness-install` leaves a suite run or hook to fetch on demand, and the commit-msg hook fetches only its checks.
- Fix: `fitness check --list`, which every new bundle runs, no longer downloads every pinned check, and the smoke test accepts skipped checks.
- Test: requirement 0173 runs single-language commits against a fake release index, and fetches newly published check versions once the index is stale.
