### 2026.11.14.1949

- Feat: the shared hooks run `fitness-install --with-checks`, so they install the engine and each pinned check.
- Feat: the reusable CI workflow installs the pinned checks through the action's `with-checks` input, once the action release carries it.
- Test: requirement 0166 installs three pinned checks and matches their `--no-builtins` results to the built-in run.
