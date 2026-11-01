# Check protocol

The runner invokes each check as `fitness-check-<name> --root <dir> [args…]` with cwd set to the repo root. Context arrives in `FITNESS_*` environment variables: staged files, changed files, tracked files, enabled check names, and the commit message for the commit checks.

- stdout — one JSON result object, whose fields are documented in the code: `go -C go doc ./internal/checkkit Result` and `go -C go doc ./internal/checkkit Finding` print them
- stderr — human display output (banners, tool passthrough)
- exit code — 0 when the check passed, 1 when it found problems, 3 when it could not judge and says why in `info`
- `--describe` — prints check metadata (name, timeout budget, context-inline arg, blob flag) so the runner needs no registry

The runner runs the `--describe` handshakes side by side, then runs checks in a pool of one slot per core. Checks start by the timeout budget they declare in `--describe`, largest first, and ties keep catalog order. A compiled check whose handshake runs past its budget among the others gets one more try. A script never does, so its work never runs twice.

A timeout kills the whole process group, so a hung check's child tree dies with it. Each git call the runner makes, such as `git ls-files`, gets ten seconds. A stalled call ends the run as a setup error naming it. Results render as a summary table in catalog order, and the run exits 1 when any check fails.

## Progress

The runner prints `→ <name>` as each check starts. On a terminal, or with `FITNESS_PROGRESS=1`, it prints a heartbeat every five seconds. The heartbeat has one line per running check, such as `… 12s  running: go-test (budget 15m)`. Off a terminal, as in Actions, a check prints that line once when it passes half its budget.

The runner holds each check's stderr until the run ends. When only one check is still running, its stderr streams live, and the summary does not print it again. A timed-out row adds the last line the check wrote to stderr, which shows where it stopped.

## Findings

A check may add structured findings beside its error lines, so an agent need not parse prose. `fitness --format=json` prints every check's result and findings to stdout and moves the table to stderr. A check without findings gets one per error line. The doc comments on `runReport` in `go/cmd/fitness/format.go` give the output's fields.

## Info

A check that cannot judge reports info. Causes include a panic, an error from `Run`, a start error, or output that is not JSON. An exit other than 0 or 1 counts too, while a timeout still fails. A failed `--describe` adds info, and the check runs with defaults.

Info rows show as `info` with an "N info" total. They never change the exit code, locally or in GitHub Actions. A printed result with `ok: false` still fails, whatever the exit code.

## Environment errors

Shared helpers in `go/internal` mark an environment error with `checkkit.Internal`, so a check can report it as info. `checkkit.Exec` tells a tool that could not start or was killed by a signal from one that ran. A language module's tool runner also reports sandbox text that `checkkit.SandboxText` spots, such as `dial tcp`, the same way.

A tool-wrapper check reports info, never a failure, when its tool is missing, could not start, or was killed. It does the same when the tool exits non-zero without a finding it can parse, and `checkkit.NoVerdict` quotes the tool's last lines. Parsed findings and threshold misses still fail.

`gitx.Repo` returns git errors and probes a clone for a shallow history, no origin, or a detached HEAD. A missing `gh`, an auth or network failure, or a read timeout is an environment error too.

## No password prompts

Checks run with `GIT_TERMINAL_PROMPT=0`, `GH_PROMPT_DISABLED=1` and ssh in batch mode, which `gitx.NoPrompt` returns. Git, ssh and gh then fail at once instead of waiting for a password on the terminal. The hooks, the pre-push gate and the installer set the same for their own runs.

A user's own `GIT_SSH_COMMAND`, `GIT_SSH` or `core.sshCommand` stays, and a credential helper still answers. A check that did not pass, whose output holds git's `terminal prompts disabled` or ssh's `Permission denied` line, shows as info naming the remote.

## Blob input

A check can judge a GitHub blob, an Issue or PR body, instead of a file tree. Given `--body-file`, it reads that body and reports it as one file. Checks never call GitHub themselves.

The `issue-check`, `plan-check`, `pr-check`, `close-check`, and `comment-check` commands fetch the blob and run their declared checks on it. They then hand the findings to an after-effect in `go/internal/aftereffect`. It comments, labels, hides old verdicts, reopens, or replies to a comment and reacts. A GitHub read a rule needs, such as an epic's sub-issues, stays in the command: `issue-check` compares them with the list `epic-structure` judges.

`dupe-check` runs no check. It scores an issue's words against older issues, and its after-effect keeps one note or closes the issue as a duplicate.

### Judging a draft

A check that reads `--body-file` declares `"blob": true` in `--describe`. `fitness --blob <text>`, or `--blob -` for stdin, runs each such check on the text alone and lists the rest as skipped (needs files). It sets `FITNESS_BLOB`, so a body check passes inert on text of another kind and is listed as not applying. A check named in `--checks` always judges the text.

`--checks=name,name` narrows the set. It exits 0 when all pass and 1 on a finding. A missing or empty text, no blob check selected, `--all`, or a check name exits 2.

## Local branch metadata

`local-branch` runs PR and Plan checks locally. Full Fitness and pre-push include it; pre-commit excludes it. `fitness branch-check` is its alias. Actions keeps its existing metadata commands.

No PR skips; invalid or unreadable metadata fails, including unfinished draft checklists. Reads are bounded and never change GitHub state. See [local-branch](../go/cmd/fitness-check-local-branch/README.md) for discovery, forks, selection, and retry behavior.

## Applicability

Org mode runs every check, so each owes the runner one more rule. In a repo it does not apply to, it passes with `filesChecked: 0`. The contract test in `go/cmd/fitness/contract_test.go` builds every check and holds each to this and to a matching `--describe` name. The few checks that apply everywhere, such as `semantic-commit`, are listed there with a reason.

## Changed files

A scoped run lists changed files in `FITNESS_CHANGED_FILES`: the staged files locally, or a pull request's diff against its base. A push to `main` or `fitness --all` leaves it unset, so every file is checked.

A check that compares files cuts its corpus to what the change can alter. duplicate-code compares the changed files with tracked files sharing their endings, as it compares only files of one format. The six `architecture-*` checks read nothing when no changed file is a view, requirement doc, product or tool source, manifest or `.gitattributes`. An unscoped run keeps every check's full walk.

`fitness --dir=<folder>` leaves it unset too. It names a file in `FITNESS_SCOPE_FILE` that lists the folder's files, one per line. `checkkit.ChangedFiles` reads either list, and `checkkit.DiffFiles` reads only the changes.

## Pull request stages

A pull request run of the whole suite takes two stages. Stage 1 sets the list and reports its table, and a failure ends the run. Stage 2 runs the suite again with the list unset and fails on any finding. It then lists the files the pull request does not change apart from the changed ones.

A named check, `--dir`, or a diff git cannot compute keeps one stage, so check out with `fetch-depth: 0`. The `--json` report and `--format=json` hold the last stage run. The git hooks never run stage 2.

A repo too large to scan whole on each pull request sets `"scopedPullRequests": true` in `.fitnessrc.json`. Its pull requests run stage 1 alone, and a push to main still scans every file.

## Config

Optional `.fitnessrc.json` at repo root:

```json
{
  "repeatedStringLiterals": { "allow": ["dist"] }
}
```

Org mode runs every check; each passes clean when it does not apply. `timeoutMs` replaces the 5-second default budget for checks that declare none. A leftover `checks` list is ignored with a warning in org mode.

External mode sets `"policy": "external"` and runs all checks by default; a nonempty `"checks"` list is an optional filter. `--policy=external --checks=name,name` can set both on the CLI; CLI selection replaces the entire config list. Unknown, empty, and duplicate names fail. See [external CI examples](ci.md) for complete setup.

## Local checks

A repo adds its own checks as `fitness/<name>/` folders, as [local checks](local-checks.md) describes. `"localChecks": "<dir>"` names one more directory inside the repo, which may also hold executable `fitness-check-<name>` files. Each follows this protocol and runs in the same suite, hooks, and CI, searched before the bundled checks. A folder check may skip `--describe`, and a handshake may add `minFitness`, the oldest fitness that runs the check.

A new name runs after the bundled checks under either policy. A bundled name replaces that check only in external mode, and the run names it; org mode refuses, as [ADR 0009](../fitness/adr-0009/README.md) decides.

A missing or empty directory fails the run, and so does a path or symlink that leads outside the repo. So does a check without execute permission, or one named like a `fitness` subcommand. Blob mode (`--blob`) and the GitHub body checks, such as `pr-check`, use the same local and disabled checks.

## Disabled checks

External mode turns checks off with `"disabledChecks": ["name"]`, after any `checks` filter. Each name must be a known check, listed once, and at least one check must remain. The git hooks skip a disabled check too, and pre-push drops its Plan gate when `plan-trailer` is off. Org mode refuses the key under the same ADR, so its repos fix the findings instead.

## Every tracked file

File checks judge every tracked file through `walkfs`; outside git, everything but `.git`. A file `walkfs.ScanFiles` cannot read is skipped and named as info, and the other files are still judged. A config setting `ignore`, `skipTheseDirectories`, or `proseBudget.exempt` fails.

A run of several checks lists tracked files once, in a temporary file that `FITNESS_TRACKED_FILES` names. The runner reads modes from `git ls-files --stage`, drops symlinks and submodules, and stats the rest once, in parallel. The file holds the root, the index path and its stamp, then each regular file on disk, every entry NUL-terminated. An interrupted run stops its checks and removes the file.

A check uses the list only when its root matches `--root` and the index stamp is unchanged, and then stats no file. Otherwise, as when run alone or outside git, it lists its own files and stats each tracked path it matches. Each check process lists a root at most once, and asks `check-attr` about each file's `linguist-generated` mark at most once, batching each walk. Checks run with `GIT_OPTIONAL_LOCKS=0`, so their own `git status` never rewrites the index.

## Code view and comment view

A run of several checks reads each file once, through the shared reader in `go/internal/comments`. It writes two views of each to a temporary folder that `FITNESS_VIEWS_DIR` names. The code view blanks every comment to spaces, so lines and columns stay put. It keeps what a tool reads in a comment:

- Go directives, the generated-file line, cgo preambles and Go example output.
- `cspell:` and `shellcheck disable=` directives, and ESLint and SwiftLint ones.
- `duplicate-code:` and `jscpd:` markers, and shebangs.

The comment view lists each comment block as prose, placed beside the code it describes.

Every code check reads the code view through `comments.CodeFile`, so commented-out code never counts. The prose checks read the comment view through `comments.Judge`, with the limits Markdown has. A view counts only while its file keeps the time it was read at. Otherwise, as when it runs alone, a check builds both views from the same reader.

## Checks that read whole files

A few checks hand whole files to a tool that reads comments the way the compiler does, so no view can stand in. These are gofmt, go-vet, go-test, go-test-coverage, go-complexity through Go's parser, eslint, prettier, swiftlint, checkstyle, spotless, java-test and the Vitest run of vitest-coverage-full. cspell spells each whole file, which is both views laid over each other, so its `cspell:` directives still cover the comments. gitignore-why judges whether a comment sits above each entry, which neither view holds alone.

