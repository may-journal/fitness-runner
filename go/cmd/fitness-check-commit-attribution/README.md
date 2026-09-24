---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# commit-attribution

Validates that a commit message discloses the AI tooling used to produce it, via two git trailers after the subject line: `AI-Tools:` and `AI-Models:`. Each must be present with a non-empty value on its own line. Opt-in, and not enabled on this repo itself — every historical commit predates the convention, so it is meant for repos that adopt it going forward.

## Enable

Add the name to `checks` in `.fitnessrc.json`, then wire it into a commit-msg hook so the proposed message is validated before the commit lands:

```json
{ "checks": ["commit-attribution"] }
```

```sh
# .git/hooks/commit-msg
fitness --check=commit-attribution --message="$(cat "$1")"
```

## What passes

Both trailers, each on its own line with a non-empty value, appearing anywhere in the body in any order (other trailers may sit alongside them):

```
feat(api): add pagination to the search endpoint

Support cursor-based paging so large result sets stay fast.

AI-Tools: Claude Code
AI-Models: Opus 4.8
Co-Authored-By: A. Dev <a@dev.io>
```

## What fails

One error per missing trailer. This body has neither:

```
feat(api): add pagination

Just a body, no attribution.
```

```
commit message missing "AI-Models:" trailer
commit message missing "AI-Tools:" trailer
```

A trailer with an empty value does not count as present. An empty message (no subject) fails with one guidance error naming both trailers.

## Advanced

Merge and revert commits are exempt — the subject prefix is matched literally:

```
Merge branch 'feature' into main    → exempt
Revert "feat(api): add endpoint"    → exempt
```

Detection is per-line via `^<Key>:[ \t]*(\S.*)$`, so an `AI-Tools:` mid-sentence is not mistaken for a trailer. To change the required keys, edit the check and its tests here.

## Behavior

- Declares `--message` as its context-inline argument. The runner forwards a proposed message (e.g. from a commit-msg hook) for validation; otherwise the check reads the last commit via `git log`.
  - Mirrors `semantic-commit`.
- Pass: both trailers present with non-empty values, or a `Merge `/`Revert ` subject.
- Fail: one error per missing trailer; on no repo, git error, or empty message, the empty-message guidance error.
- `filesChecked` is always 1.

## Contributing

This README is the canonical description for this check, a self-contained `fitness-check-commit-attribution` binary. To change the rules, extend the check and tests here and keep this README in sync.
