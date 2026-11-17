### 2026.11.16.2040

> Every input fitness judges is a document with a path, so one `files` pattern decides which checks apply.

- Feat: commit messages, branch history, and PR, issue and comment bodies get virtual paths that `checkkit` exports.
- Feat: `scope` and `blob` are deprecated aliases the engine maps onto `files`, `exclude` and `requires`; `fitness explain` shows the mapped fields.
- Feat: `fitness --blob` judges text at `.fitness/blob.md` or the `--as` path, and the Claude Code hook judges drafts at GitHub's paths.
- Feat: a run reaching `local-branch` fetches only the checks the branch's PR and Plan need, not every check.
- Test: requirement 0176 covers each mapping, `--as`, the mapped fields, the narrowed fetch and GitHub body paths.
