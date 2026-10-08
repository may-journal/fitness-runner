---
fitnessFunctions: ['requirements']
relatedConfigurations: ['../../../.fitnessrc.json']
---

# requirements

In a Go repo, each file in `docs/requirements` is one requirement. It says why the requirement matters, how to measure it, and which acceptances prove it. Each acceptance is owned by exactly one Go test named for it, and each test owns exactly one acceptance. A test that proves no requirement is either a requirement nobody wrote down or a duplicate to delete.

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
<!-- Exactly one Go test is named for each ID: func Test0001_1(t *testing.T) owns 0001.1. Delete the test when you delete the ID. -->
- 0001.1
    - Given no network
        - When I edit an entry
            - Then it is saved locally
```

## Rules

- IDs: the file name's `NNNN` matches the title, and no two docs share it.
- Acceptance IDs: the title ID plus `.N`, unique in its doc.
- Reuse: git history records every ID a commit removed. A doc or acceptance that brings one back fails.
- Sections: exactly `## Why`, `## Measurement`, `## Requirements`, in that order, none empty.
- Measurement: numerator, `-`, denominator, then `Source:` citing one `path:line` that exists or one `#123` issue.
- Acceptance: one Given, one When, and one Then, each nested under the line above. A branch is a new ID.
- Ownership: exactly one `func TestNNNN_N` owns each acceptance, and a test named for an undefined acceptance fails.
- Every test: each top-level test in a judged file must be named for an acceptance. `TestMain` and helpers are not tests.

Tests are read with `go/ast`, so only real top-level test functions count.

## Behavior

- No Go: a repo without Go files passes with `filesChecked: 0`.
- Missing docs: a Go repo with none in `docs/requirements` fails.
- Scoped runs skip when no doc or test file changed.
- Otherwise a scoped run judges every doc, but only changed test files must be fully named.
- Full runs judge every test file, so an unmapped repo stays red until every test owns a requirement.

## Contributing

This README is the canonical description for this check, a self-contained `fitness-check-requirements` binary. Extend the check and its tests here and keep this README in sync.
