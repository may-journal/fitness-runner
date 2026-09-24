---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# changelog-updated

When the runner provides staged file context, it runs two checks. First, the added lines in `CHANGELOG.md` (in the staged diff) must share at least three words with the rest of the staged diff. Second, any new `### yyyy.mm.dd.HHMM` section heading must use the current date and time (hour and minute) at check run time—not a guessed time.

## Behavior

- Pass (no staged context): Empty staged list → skip (ok).
- Pass: Only `CHANGELOG.md` is staged (no other files to compare) → ok.
- Pass: Changelog additions share ≥ 3 words with the rest of the staged diff, and any new `### yyyy.mm.dd.HHMM` heading matches current date and time.
- Fail: `CHANGELOG.md` missing on disk → prompt to add it and mention changes.
- Fail: `CHANGELOG.md` not in the staged diff (no additions) → "Stage CHANGELOG.md and add an entry...".
- Fail: New section heading uses wrong date/time → "CHANGELOG.md new section heading must use current date and time (yyyy.mm.dd.HHMM), not a guessed time" with expected value.
- Fail: Changelog additions share fewer than three words with rest of diff → error with count.
  - A second line lists up to 10 words from the staged diff (e.g. use words like: …) to help fix the entry.

Only the modified (added) parts of `CHANGELOG.md` in the diff are considered, not the whole file. Words are lowercased and length ≥ 3. Time is taken at check run (e.g. pre-commit); use that exact `yyyy.mm.dd.HHMM` for new section headings.

## Contributing

This README is the canonical description for this check. Extend this check here (e.g. configurable minimum overlap, or restrict to the latest changelog section) and keep the README and tests in sync.
