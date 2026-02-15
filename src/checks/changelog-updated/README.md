# changelog-updated

When run with `--staged`, validates that `CHANGELOG.md` contains at least three distinct words that also appear in the staged diff (so the changelog was updated to reflect the commit).

## Behavior

- **Pass (no staged context):** No `--staged` or empty staged list → skip (ok).
- **Pass:** Staged diff has no significant words → ok (e.g. only binary or punctuation).
- **Pass:** At least three words from the staged diff appear in `CHANGELOG.md`.
- **Fail:** `CHANGELOG.md` missing with staged changes → prompt to add it and mention changes.
- **Fail:** Fewer than three overlapping words → error with count and examples.

Words are lowercased and length ≥ 3.

## Contributing

Extend this check here (e.g. configurable minimum overlap, or restrict to the latest changelog section) and keep the README and tests in sync.
