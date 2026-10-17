---
fitnessFunctions: ['requirements']
relatedConfigurations: ['../../../.fitnessrc.json']
---

# requirements

In a repo written in any language of the [module registry](../../internal/lang/langs/langs.go), each file in `docs/requirements` is one requirement. It says why the requirement matters, how to measure it, and which acceptances prove it. Each acceptance is owned by exactly one test named for it, and each test owns exactly one acceptance. A test that proves no requirement is either a requirement nobody wrote down or a duplicate to delete.

## Template

Copy this into `docs/requirements/NNNN-kebab-title.md`. The comments explain each rule.

```markdown
<!-- File name is NNNN-kebab-title.md. NNNN matches the title ID, is unique in docs/requirements, and is never reused after deletion. -->
# 0001 Offline Sync

## Why
<!-- One sentence: what a person can do once this holds. -->

I can keep writing when signal drops or is lost

## Measurement
<!-- Numerator line, a "-" line, denominator line. Then "Source:" citing either a line of code that exists (path:line) or the #issue that will implement it. -->

users who save locally
-
users who have lost connection

Source: #123

## Requirements
<!-- Top-level bullets are IDs: title ID + ".N", unique, never reused. -->
<!-- Each ID nests exactly one Given, one When, one Then, one level deeper each, each line starting with its keyword. A second Given, When, or Then is a new ID. -->
<!-- A Given starts with an actor the system context declares, by name or Mermaid ID; see the architecture-actors check. -->
<!-- Exactly one test is named for each ID, the way its language module names tests; the check's error names the test it expects. Delete the test when you delete the ID. -->
- 0001.1
    - Given Writer has no network
        - When I edit an entry
            - Then it is saved locally
```

## Rules

- IDs: the file name's `NNNN` matches the title, and no two docs share it or one area, per [ADR 0004](../../../docs/architecture/adr/0004-one-requirement-doc-per-area.md).
- Acceptance IDs: the title ID plus `.N`, unique in its doc.
- Reuse: git history records every ID a commit removed. A doc or acceptance that brings one back fails.
- Sections: exactly `## Why`, `## Measurement`, `## Requirements`, in that order, none empty.
- Measurement: numerator, `-`, denominator, then `Source:` citing one `path:line` that exists or one `#123` issue.
- Acceptance: one Given, one When, and one Then, nested in order; the Given starts with an [actor](../fitness-check-architecture-actors/README.md).
- Ownership: exactly one test named for each acceptance owns it. A test named for an undefined acceptance fails.
- Every test: each top-level test in the repository must be named for an acceptance. Setup functions and helpers are not tests.

Each language module finds its tests and names the test that owns an ID, as its package documents. A comment or string never counts as a test.

## Behavior

- No code: a repo with no file in a module's language passes with `filesChecked: 0`.
- Not code: a file a module names a tool config, or a file `.gitattributes` marks `linguist-generated`.
- Missing docs: a repo with such code and none in `docs/requirements` fails.
- Full and scoped runs judge every doc and test file, including unchanged files in pull requests that change only unrelated files.
- Ownership is a repository-wide invariant: an unmapped repo stays red until every test owns a requirement.

## Pairs

A requirement test and its doc change together, so neither drifts from the other. Changed behavior extends its existing doc, as [ADR 0004](../../../docs/architecture/adr/0004-one-requirement-doc-per-area.md) sets out.

- Pairing: a test file pairs with each doc whose number its name or its tests carry, one or several.
- Tests: a changed test file fails unless one of its docs changed too. The error names both files and the missing doc.
- Docs: a changed doc fails unless one of its test files changed too. The error names both files and the missing test.
- Helpers: a test file that names and holds no requirement pairs with nothing.
- Exemptions: a commit trailer `Unpaired-Requirement: NNNN, NNNN <reason>` exempts those numbers. A pull request fails until its body lists each such trailer.

Which changes are judged depends on the run:

- Commits: the `commit-msg` hook judges the staged files, since only it can read the message. The `pre-commit` run leaves pairs to it.
- Amends: an amended commit is judged on everything it changes since its parent.
- Pull requests: CI judges the diff against the current merge base with the base branch, so merged base changes are not judged.
- Bodies: CI reads the pull request body from its event, so push or reopen after listing a trailer.
- Replays: a local merge, cherry-pick, or revert, or an amended merge, is not judged again. Its pull request diff still is.
- Full scans: a run with nothing staged, or a push to the default branch, judges no pairs.

## Contributing

This README is the canonical description for this check, a self-contained `fitness-check-requirements` binary. Extend the check and its tests here and keep this README in sync.
