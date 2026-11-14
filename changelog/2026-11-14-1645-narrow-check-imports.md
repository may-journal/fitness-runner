### 2026.11.14.1645

- Refactor: `comment-structure` and `workflow-ref-exists` import `MarkerPrefix` and `GHErr` from `checkkit`, not from `aftereffect`.
- Refactor: the engine hands `local-branch` its judgment from `metadata`, so the check imports neither package.
- Refactor: no check imports `metadata`, `aftereffect`, `dupe` or `report`, and no behavior changes.
- Test: requirement 0160 proves that no check, tests included, depends on those four packages.
