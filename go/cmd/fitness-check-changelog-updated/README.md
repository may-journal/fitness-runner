---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# changelog-updated

When the runner provides staged file context, it runs three checks. The added lines in the branch's [entry file](../fitness-check-changelog/README.md#entry-files) must share at least three words with the rest of the staged diff. A branch keeps one entry file, so later commits add bullets to the file its first commit wrote. Its added `### yyyy.mm.dd.HHMM` heading must use the check run time, or up to five minutes before it.

`CHANGELOG.md` is history that only the release PR writes. A commit that changes a committed one, such as a new or edited section, fails pointing to an entry file in `changelog/`. A commit may add the first one.

A branch's own commits are those no other `origin` branch holds, so a stacked branch starts where its base ends. On the default branch, or without `origin` refs, each commit is judged alone. `fitness pr-check` holds the whole pull request to one entry.

## Behavior

- Pass (replay in progress): mid-merge, mid-cherry-pick, or mid-revert (`MERGE_HEAD`/`CHERRY_PICK_HEAD`/`REVERT_HEAD` present) → skip, since a replay re-stages historical entries.
- Pass: no staged context → skip (ok).
- Pass: entry file additions share ≥ 3 words with the rest of the staged diff, under a current heading or the branch's first one.
- Fail: `CHANGELOG.md` missing on disk → prompt to add it.
- Fail: no staged entry file → "Stage a changelog/<name>.md entry file that mentions your staged changes".
- Fail: the branch already has its entry file and the commit adds another → error naming the branch's entry file and each new one.
- Fail: the added heading uses wrong date/time → error naming the expected `yyyy.mm.dd.HHMM` value.
- Fail: entry file additions share fewer than three words with rest of diff → error with count.
  - A second line lists up to 10 words from the staged diff (e.g. use words like: …) to help fix the entry.

Only the added parts of the entry file in the diff count, with words lowercased and at least 3 long. Time is taken at check run (e.g. pre-commit). A `package.json` version ending in `yyyy.mm.dd.HHMM` must match the heading exactly.

## Contributing

This README is the canonical description for this check. Extend this check here (e.g. configurable minimum overlap, or restrict to the latest changelog section) and keep the README and tests in sync.
