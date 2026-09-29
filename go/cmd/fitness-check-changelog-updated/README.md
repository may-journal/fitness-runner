---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# changelog-updated

When the runner provides staged file context, it runs two checks. First, the added lines in `CHANGELOG.md` (in the staged diff) must share at least three words with the rest of the staged diff. Second, the newest added `### yyyy.mm.dd.HHMM` heading must use the check run time, or up to five minutes before it. Older headings further down the diff are history, such as sections a changelog rewrite splits, and keep their times.

## Behavior

- Pass (replay in progress): mid-merge, mid-cherry-pick, or mid-revert (`MERGE_HEAD`/`CHERRY_PICK_HEAD`/`REVERT_HEAD` present) → skip, since a replay re-stages historical entries.
- Pass: no staged context, or only `CHANGELOG.md` staged (nothing to compare) → skip (ok).
- Pass: Changelog additions share ≥ 3 words with the rest of the staged diff, and the newest added heading is current.
- Fail: `CHANGELOG.md` missing on disk → prompt to add it and mention changes.
- Fail: `CHANGELOG.md` not in the staged diff (no additions) → "Stage CHANGELOG.md and add an entry...".
- Fail: The newest added section heading uses wrong date/time → error naming the expected `yyyy.mm.dd.HHMM` value.
- Fail: Changelog additions share fewer than three words with rest of diff → error with count.
  - A second line lists up to 10 words from the staged diff (e.g. use words like: …) to help fix the entry.

Only the modified (added) parts of `CHANGELOG.md` in the diff are considered, not the whole file. Words are lowercased and length ≥ 3. Time is taken at check run (e.g. pre-commit). A `package.json` version ending in `yyyy.mm.dd.HHMM` must match the heading exactly.

## Contributing

This README is the canonical description for this check. Extend this check here (e.g. configurable minimum overlap, or restrict to the latest changelog section) and keep the README and tests in sync.
