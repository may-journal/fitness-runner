### 2026.11.15.2235

> A check declares which files reach it and what it requires, and the engine leaves out the checks a run cannot use.

- Feat: `checkkit.Describe` gains `files`, `exclude` and `needs` RE2 patterns, and `requires`, a list of typed `Requirement` values.
- Feat: a scoped run skips checks no file reaches, and `--blob` skips checks that require the repo, the diff or the history.
- Feat: a check whose `needs` matches no file git tracks is skipped, though a failed git call skips none.
- Feat: an unknown requirement skips its check and names the upgrade, and a pattern that won't compile stops the run.
- Test: requirement 0172 runs installed checks that declare each field.
