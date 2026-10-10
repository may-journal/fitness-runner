---
fitnessFunctions: ['requirements']
relatedConfigurations: ['../../../.fitnessrc.json']
---

# requirements

In a Go or Swift repo, each file in `docs/requirements` is one requirement. It says why the requirement matters, how to measure it, and which acceptances prove it. Each acceptance is owned by exactly one test named for it, and each test owns exactly one acceptance. A test that proves no requirement is either a requirement nobody wrote down or a duplicate to delete.

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
<!-- Exactly one test is named for each ID: Go func Test0001_1(t *testing.T) or Swift func test0001_1() owns 0001.1. Delete the test when you delete the ID. -->
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
- Ownership: exactly one Go `func TestNNNN_N` or Swift `func testNNNN_N` owns each acceptance. A test named for an undefined acceptance fails.
- Every test: each top-level test in a judged file must be named for an acceptance. `TestMain` and helpers are not tests.

Go tests are read with `go/ast`, so only real top-level test functions count.

A Swift test is an XCTest method named `test…` with no parameters, in a class that inherits `XCTestCase`, or any function marked `@Test`. Comments and string literals are skipped, so a name inside them is no test.

## Behavior

- No code: a repo without Go or Swift files passes with `filesChecked: 0`.
- Missing docs: a Go or Swift repo with none in `docs/requirements` fails.
- Scoped runs skip when no doc, Go test file, or Swift file changed.
- Otherwise a scoped run judges every doc, but only changed test files must be fully named.
- Full runs judge every test file, so an unmapped repo stays red until every test owns a requirement.

## Contributing

This README is the canonical description for this check, a self-contained `fitness-check-requirements` binary. Extend the check and its tests here and keep this README in sync.
