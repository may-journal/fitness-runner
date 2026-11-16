### 2026.11.16.1740

> The issue, comment and PR bots fetch only the checks their bodies run, not every pinned check.

- Feat: `pr-check`, `issue-check`, `comment-check` and the other GitHub commands fetch each pinned check the first time they run it.
- Feat: `fitness-command` installs on demand for events, and a manual sweep still installs every pinned check up front.
- Feat: the checks cache keys on the release index hash and job, falls back to older sets, and saves once a set changes.
- Test: requirement 0171 runs `pr-check` against a fake release index, and 0174 reads the cache key and install modes.
