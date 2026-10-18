---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# workflow-ref-exists

Fails a workflow that calls a may-journal workflow or action at a tag or sha before that path exists there ([#485](https://github.com/may-journal/fitness-runner/issues/485)). Such a caller fails every event in its repo with a missing-workflow error, as fitness-runner#249 did.

## Behavior

- It reads each `uses:` line in the workflow files directly under `.github/workflows`.
- A `may-journal/<repo>/<path>@<ref>` line names a reusable workflow file or an action folder. A bare `may-journal/<repo>@<ref>` names the repo's root action.
- A ref is a tag or a full 40-character sha, and the path must exist at it.
- An action folder holds an `action.yml` or `action.yaml` file at the ref.
- Fail: the path is missing at the ref, naming the file, line, path, and ref.
- Fail: the ref is neither a tag nor a branch.
- Ignored: local `./` actions, other owners, branch refs, and refs built from `${{ }}` expressions.

In a pull request, or a staged commit, it reads only the changed workflow files.

## GitHub access

The check asks GitHub through `gh api`, so it uses `gh`'s login or `GH_TOKEN`. The org's `fitness-suite` and CI workflows pass `GH_TOKEN` to the suite for it.

When GitHub cannot answer, it skips with an info note and never passes or fails a ref it could not see. That covers no `gh`, no network, a bad token, and a rate limit. A repo that answers 404 is one the token cannot read, so its lines are skipped with a note as well.
