---
relatedConfigurations: ['../.fitnessrc.json']
---

# Check protocol

The runner invokes each check as `fitness-check-<name> --root <dir> [args…]` with cwd set to the repo root. Context arrives in `FITNESS_*` environment variables: staged files, changed files, enabled check names, and the commit message for the commit checks.

- stdout — one JSON result object: `{"ok": bool, "errors": [".."], "filesChecked": n, "info": [".."]}`, where `info` is optional
- stderr — human display output (banners, tool passthrough)
- exit code — 0 when the check passed, 1 when it found problems, 3 when it could not judge and says why in `info`
- `--describe` — prints check metadata (name, timeout budget, context-inline arg, blob flag) so the runner needs no registry

The runner runs the `--describe` handshakes side by side, then runs checks in a pool of one slot per core. Checks start by the timeout budget they declare in `--describe`, largest first, and ties keep catalog order. A handshake that runs past its budget among the others gets one more try alone.

A timeout kills the whole process group, so a hung check's child tree dies with it. Results render as a summary table in catalog order, and the run exits 1 when any check fails.

## Info

A check that cannot judge reports info. Causes include a panic, an error from `Run`, a start error, or output that is not JSON. An exit other than 0 or 1 counts too, while a timeout still fails. A failed `--describe` adds info, and the check runs with defaults.

Info rows show as `info` with an "N info" total. They never change the exit code, locally or in GitHub Actions. A printed result with `ok: false` still fails, whatever the exit code.

## Environment errors

Shared helpers in `go/internal` mark an environment error with `checkkit.Internal`, so a check can report it as info. `checkkit.Exec` tells a tool that could not start or was killed by a signal from one that ran. A language module's tool runner also reports sandbox text that `checkkit.SandboxText` spots, such as `dial tcp`, the same way.

`gitx.Repo` returns git errors and probes a clone for a shallow history, no origin, or a detached HEAD. A missing `gh`, an auth or network failure, or a read timeout is an environment error too.

## Blob input

A check can judge a GitHub blob, an Issue or PR body, instead of a file tree. Given `--body-file`, it reads that body and reports it as one file. Checks never call GitHub themselves.

The `issue-check`, `plan-check`, `pr-check`, `close-check`, and `comment-check` commands fetch the blob and run their declared checks on it. They then hand the findings to an after-effect in `go/internal/aftereffect`. It comments, labels, hides old verdicts, reopens, or replies to a comment and reacts. A GitHub read a rule needs, such as an epic's sub-issues, stays in the command: `issue-check` compares them with the list `epic-structure` judges.

### Judging a draft

A check that reads `--body-file` declares `"blob": true` in `--describe`. `fitness --blob <text>`, or `--blob -` for stdin, runs each such check on the text alone and lists the rest as skipped (needs files). It sets `FITNESS_BLOB`, so a body check passes inert on text of another kind and is listed as not applying. A check named in `--checks` always judges the text.

`--checks=name,name` narrows the set. It exits 0 when all pass and 1 on a finding. A missing or empty text, no blob check selected, `--all`, or a check name exits 2.

## Local branch metadata

`local-branch` runs PR and Plan checks locally. Full Fitness and pre-push include it; pre-commit excludes it. `fitness branch-check` is its alias. Actions keeps its existing metadata commands.

No PR skips; invalid or unreadable metadata fails, including unfinished draft checklists. Reads are bounded and never change GitHub state. See [local-branch](../go/cmd/fitness-check-local-branch/README.md) for discovery, forks, selection, and retry behavior.

## Applicability

Org mode runs every check, so each owes the runner one more rule. In a repo it does not apply to, it passes with `filesChecked: 0`. The contract test in `go/cmd/fitness/contract_test.go` builds every check and holds each to this and to a matching `--describe` name. The few checks that apply everywhere, such as `semantic-commit`, are listed there with a reason.

## Changed files

A scoped run lists changed files in `FITNESS_CHANGED_FILES`: the staged files locally, or a pull request's diff against its base. A push to `main` or `fitness --all` leaves it unset, so every file is checked. Checks that compare files, such as jscpd, keep a full walk.

## Config

Optional `.fitnessrc.json` at repo root:

```json
{
  "repeatedStringLiterals": { "allow": ["dist"] }
}
```

Org mode runs every check; each passes clean when it does not apply. `timeoutMs` replaces the 5-second default budget for checks that declare none. A leftover `checks` list is ignored with a warning in org mode.

External mode sets `"policy": "external"` and runs all checks by default; a nonempty `"checks"` list is an optional filter. `--policy=external --checks=name,name` can set both on the CLI; CLI selection replaces the entire config list. Unknown, empty, and duplicate names fail. See [external CI examples](ci.md) for complete setup.

## Every tracked file

File checks judge every tracked file through `walkfs`; outside git, everything but `.git`. A file `walkfs.ScanFiles` cannot read is skipped and named as info, and the other files are still judged. A config setting `ignore`, `skipTheseDirectories`, `disabledChecks`, or `proseBudget.exempt` fails.

