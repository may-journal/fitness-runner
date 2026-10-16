---
fitnessFunctions: ['semantic-commit']
relatedConfigurations: ['../../../.fitnessrc.json']
---

# semantic-commit

Validates that a commit message follows [Conventional Commits](https://www.conventionalcommits.org/): `type(scope): description`, or a merge commit (`Merge ...`).

## Allowed types

From [conventional-commit-types](https://github.com/commitizen/conventional-commit-types), inlined in the check binary.

Scope is required (e.g. `feat(api): add endpoint`). Merge commits are always accepted.

## Behavior

- Pass: the subject matches `type(scope): description` or `Merge ...`. Only the first line is judged.
- Fail: the subject doesn’t match → error quoting it, with the suggested format and allowed types.
- Fail (empty message, or no repo to read): a no-message error.

## Which message it judges

- commit-msg hook: the message file being written, passed as `--message`.
- `pr-check` workflow: the PR title, passed the same way. A `--message` argument always wins.
- pre-commit hook with a commit-msg hook beside it: none. The new message does not exist yet, and HEAD is only its parent.
- pre-commit hook alone, such as `fitness-install --install-hook`: the branch’s earlier commits, as a full run judges them. A bad subject fails the next commit.
- Full run, such as `fitness --all`: each of the branch’s own commits, never history inherited from its base.

A full run finds the base the way the `local-branch` check does. A pull request run in GitHub Actions starts from the fetched `origin/$GITHUB_BASE_REF`. Locally, a branch counts the commits no other origin branch holds, whatever the default branch is called.

The default branch is `origin/HEAD`, else a fetched `origin/main` or `origin/master`. On it, on a detached HEAD, or while the default is unknown, only commits origin lacks count. With no commits of its own, the check passes with 0 checked.

A repo with no origin refs inherits nothing, so a full run judges the HEAD commit subject.

## Contributing

This README is the canonical description for this check. This check is a self-contained binary (`fitness-check-semantic-commit`). To add types or relax rules, extend the check and tests here and keep the README in sync.
