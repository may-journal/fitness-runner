### 2026.11.14.1949

- Feat: the commit-msg and pre-commit hooks set `FITNESS_WITH_CHECKS=true`, which the installer reads and an older one ignores.
- Feat: the reusable CI workflow installs the pinned checks through the action's `with-checks` input, once the action release carries it.
- Test: requirement 0166 installs three pinned checks and matches their `--no-builtins` results to the built-in run.
