---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# commit-attribution

Validates that a commit message discloses the AI tooling used to produce it, via two git trailers after the subject line: `AI-Tools:` and `AI-Models:`. Each must be present with a non-empty value on its own line. Opt-in — not part of `defaultChecks`, and unlike the other opt-in checks it is not enabled on this repo itself: every historical commit predates the trailer convention, so turning it on here would fail the whole suite. It is meant for repos that adopt the convention going forward.

## Enable

```ts
// .fitnessrc.ts
export default { checks: ['commit-attribution'] };
```

Wire it into a commit-msg hook so the proposed message is validated before the commit lands:

```sh
# .git/hooks/commit-msg
fitness --check=commit-attribution --message="$(cat "$1")"
```

## What passes

Both trailers present, each on its own line with a non-empty value, after the subject:

```
feat(api): add pagination to the search endpoint

Support cursor-based paging so large result sets stay fast.

AI-Tools: Claude Code
AI-Models: Opus 4.8
```

The trailers may carry any non-empty value and appear anywhere in the body — order between them does not matter, and other trailers (`Co-Authored-By:`, `Closes:`) can sit alongside them.

## What fails

A message missing either trailer reports one error per missing trailer. This body has neither:

```
feat(api): add pagination to the search endpoint

Just a body, no attribution.
```

produces:

```
commit message missing "AI-Models:" trailer
commit message missing "AI-Tools:" trailer
```

A trailer with no value (`AI-Tools:` followed by nothing, or only whitespace) does not count as present — the value after the colon must be non-empty. An empty commit message (no subject) fails with a single guidance error:

```
No commit message to validate; add "AI-Tools:" and "AI-Models:" trailers to disclose AI usage
```

## Advanced

Merge and revert commits are exempt (they pass without trailers) because they are not authored content — the subject-line prefix is matched literally:

```
Merge branch 'feature' into main    → exempt
Revert "feat(api): add endpoint"    → exempt
```

Trailer detection is per-line via `^<Key>:[ \t]*(\S.*)$` (multiline), so an `AI-Tools:` string that appears mid-sentence in the body prose is not mistaken for a trailer — it must start its own line. Only the two keys in `REQUIRED_TRAILERS` (`AI-Models`, `AI-Tools`) are required; to change or extend the required set, edit the check and its tests here and keep this README in sync.

## Behavior

- Registers `contextInline: { argName: '--message', contextKey: 'proposedCommitMessage' }`. When `proposedCommitMessage` is set (e.g. from a commit-msg hook), that message is validated; otherwise the check reads the last commit with `git log -1 --pretty=%B`. This mirrors `semantic-commit`'s resolution, so both flow the same proposed message from the same hook wiring.
- Pass: the message contains both an `AI-Tools:` and an `AI-Models:` trailer, each with a non-empty value — or the subject is a `Merge `/`Revert ` commit.
- Fail: one error per missing required trailer.
- Fail (no repo / git error / empty message): returns the `MSG_EMPTY` guidance error so hook and explicit `--message` runs get a clear signal.
- `filesChecked` is always 1 (the single commit message under validation).

## Contributing

This README is the canonical description for this check. This check is a self-contained package (`@mayjournal/fitness-check-commit-attribution`). To change the required trailers or relax rules, extend the check and tests here and keep the README in sync.
