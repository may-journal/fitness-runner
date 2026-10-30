### 2026.10.30.1422
- Fix: `prettier` leaves a staged symbolic link out of a scoped run, so committing one no longer fails.
- Fix: the link's target is still checked as its own file, as unscoped runs already did.
- Docs: requirement 0052.5 now commits a file and a link to it through the real hooks.
