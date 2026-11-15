### 2026.11.15.2130

> The GitHub commands fetch the pinned checks, so a repo's tests that run `fitness pr-check` judge again.

- Fix: `pr-check`, `issue-check`, `plan-check`, `comment-check` and the other GitHub commands fetch missing pinned checks.
- Fix: a body judged with no checks installed no longer reports each check as "no result from the check".
- Test: requirement 0171 runs `fitness pr-check` from a pinned repo with none installed.
