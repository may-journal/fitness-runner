### 2026.11.15.0137

- Feat: a `*` pin in `checkVersions` gives every released check that pin, and a check named on its own keeps its own.
- Feat: the reusable workflows install the checks a repo pins, the same way the shared hooks do.
- Feat: on hosted runners the shared install action caches installed checks per pin set, so jobs stop downloading them each run.
- Test: requirement 0163.4 spreads a `*` pin over every released check.
