---
name: workflow-ref-exists
description: Fix a failing workflow-ref-exists check, where a workflow calls a may-journal workflow or action at a tag or sha that lacks it. Use before merging a workflow that pins a new ref.
---

# Pin a ref that has the file

A workflow that calls `may-journal/<repo>/<path>@<ref>` fails every event if the path does not exist at that ref. [The check README](../../go/cmd/fitness-check-workflow-ref-exists/README.md) states the rules, what it ignores, and what happens when GitHub cannot answer.

## Read the failure

- `.github/workflows/ci.yml: line 5: …/comment-check-reusable.yml@v1: .github/workflows/comment-check-reusable.yml does not exist at v1`: the tag predates the file. Release first and push the tag, then pin it. Never move the tag.
- `line 5: may-journal/fitness-runner@v9: v9 is not a tag, a branch, or a full commit sha`: write an existing tag or a full 40-character sha.
- `no commit 000… exists`: the sha is wrong. Copy it from the repo.
- A note about calls left unverified: GitHub could not answer. Run `gh auth status`, then rerun.

Push the tag before you merge the workflow that uses it.

## Check it first

Run `fitness workflow-ref-exists` in your work tree. It reads changed workflow files in a PR or staged commit, and all of them otherwise.
