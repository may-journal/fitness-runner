# changelog-updated

When run with `--staged`, validates that the **added lines** in `CHANGELOG.md` (in the staged diff) share at least three words with the rest of the staged diff—so you actually updated the changelog for this commit.

## Behavior

- **Pass (no staged context):** No `--staged` or empty staged list → skip (ok).
- **Pass:** Only `CHANGELOG.md` is staged (no other files to compare) → ok.
- **Pass:** Changelog additions share ≥ 3 words with the rest of the staged diff.
- **Fail:** `CHANGELOG.md` missing on disk → prompt to add it and mention changes.
- **Fail:** `CHANGELOG.md` not in the staged diff (no additions) → "Stage CHANGELOG.md and add an entry...".
- **Fail:** Changelog additions share fewer than three words with rest of diff → error with count and examples.

Only the **modified (added) parts** of `CHANGELOG.md` in the diff are considered, not the whole file. Words are lowercased and length ≥ 3.

## Contributing

Extend this check here (e.g. configurable minimum overlap, or restrict to the latest changelog section) and keep the README and tests in sync.
