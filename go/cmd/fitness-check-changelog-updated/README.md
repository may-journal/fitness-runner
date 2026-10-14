---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# changelog-updated

When the runner provides staged file context, it runs these checks. The added lines in the branch's [entry file](../fitness-check-changelog/README.md#entry-files) must share at least three words with the rest of the staged diff. A branch keeps one entry file, so later commits add bullets to the file its first commit wrote. You write its `### yyyy.mm.dd.HHMM` heading; the [changelog](../fitness-check-changelog/README.md) check judges its format.

`CHANGELOG.md` is history that only the release PR writes. A commit that changes the branch's starting copy, such as with a new or edited section, fails pointing to `changelog/`. A branch may add and edit the first one.

A branch's own commits are those no other `origin` branch holds, so a stacked branch starts where its base ends. On the default branch, or without `origin` refs, each commit is judged alone. `fitness pr-check` holds the whole pull request to one entry.

## Behavior

- Pass (replay in progress): mid-merge, mid-cherry-pick, or mid-revert (`MERGE_HEAD`/`CHERRY_PICK_HEAD`/`REVERT_HEAD` present) → skip, since a replay re-stages historical entries.
- Pass: no staged context → skip (ok).
- Pass: entry file additions share ≥ 3 words with the rest of the staged diff.
- Fail: `CHANGELOG.md` missing on disk → prompt to add it.
- Fail: no staged entry file → "Stage a changelog/<yyyy-mm-dd-HHMM>-<change>.md entry file that mentions your staged changes".
- Fail: the branch already has its entry file and the commit adds another → error naming the branch's entry file and each new one.
- Fail: a staged new entry file whose name lacks its heading's date → error naming the file it should be.
- Fail: entry file additions share fewer than three words with rest of diff → error with count.
  - A second line lists up to 10 words from the staged diff (e.g. use words like: …) to help fix the entry.

Only the added parts of the entry file in the diff count, with words lowercased and at least 3 long.

## Contributing

This README is the canonical description for this check. Extend this check here (e.g. configurable minimum overlap, or restrict to the latest changelog section) and keep the README and tests in sync.
