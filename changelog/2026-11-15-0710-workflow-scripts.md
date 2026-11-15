### 2026.11.15.0710

> Every workflow and action step that did real work in shell now calls a checked-in script.

- Refactor: the multi-line `run:` steps of the workflows, reusable workflows and actions move into `scripts/` and each action's folder, with their text unchanged.
- Feat: `.github/actions/script` runs a script by name for reusable workflows, whose callers' checkouts do not hold it.
- Test: requirement 0008.7 proves every script a workflow or action names exists and is executable.
- Chore: CI runs shellcheck over the scripts.
- Test: requirement 0008.8 proves the script action declares every step output a script writes.
