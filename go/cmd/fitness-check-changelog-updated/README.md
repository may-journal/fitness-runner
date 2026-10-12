---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# changelog-updated

When the runner provides staged file context, it runs three checks. The added lines in `CHANGELOG.md` must share at least three words with the rest of the staged diff. A branch keeps one dated `### yyyy.mm.dd.HHMM` entry, so later commits add bullets to the entry its first commit wrote. The newest added heading must use the check run time, or up to five minutes before it.

Older headings further down the diff are history, such as sections a changelog rewrite splits, and keep their times. A branch's own commits are those no other `origin` branch holds, so a stacked branch starts where its base ends. On the default branch, or without `origin` refs, each commit is judged alone. `fitness pr-check` holds the whole pull request to one entry.

## Behavior

- Pass (replay in progress): mid-merge, mid-cherry-pick, or mid-revert (`MERGE_HEAD`/`CHERRY_PICK_HEAD`/`REVERT_HEAD` present) → skip, since a replay re-stages historical entries.
- Pass: no staged context, or only `CHANGELOG.md` staged (nothing to compare) → skip (ok).
- Pass: Changelog additions share ≥ 3 words with the rest of the staged diff, under a current new heading or the branch's first entry.
- Fail: `CHANGELOG.md` missing on disk, or not in the staged diff → prompt to add or stage it ("Stage CHANGELOG.md and add an entry...").
- Fail: the branch already has its entry and the commit adds another dated section → error naming the branch's entry and each new heading.
- Fail: The newest added section heading uses wrong date/time → error naming the expected `yyyy.mm.dd.HHMM` value.
- Fail: Changelog additions share fewer than three words with rest of diff → error with count.
  - A second line lists up to 10 words from the staged diff (e.g. use words like: …) to help fix the entry.

Only the added parts of `CHANGELOG.md` in the diff count, with words lowercased and at least 3 long. Time is taken at check run (e.g. pre-commit). A `package.json` version ending in `yyyy.mm.dd.HHMM` must match the heading exactly.

## Contributing

This README is the canonical description for this check. Extend this check here (e.g. configurable minimum overlap, or restrict to the latest changelog section) and keep the README and tests in sync.
