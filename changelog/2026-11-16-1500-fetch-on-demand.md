### 2026.11.16.1500

> A run downloads only the checks its files reach, so a Markdown-only commit fetches a few checks, not all 69.

- Feat: a suite run matches its files against each pinned version's index fields and fetches only matching checks in parallel, each SHA-256 verified.
- Feat: a full run fetches every pinned check, and a reached check that cannot be fetched is an info row naming it.
- Feat: `fitness-install` installs checks up front only for `--install-only`, the action and `--no-builtins`; the commit-msg hook fetches only its checks.
- Fix: `fitness check --list`, which every new bundle runs, no longer downloads every pinned check.
- Test: requirement 0173 runs Markdown-only, Go-only and Swift-only commits against a fake release index, and the smoke test accepts skipped checks.
