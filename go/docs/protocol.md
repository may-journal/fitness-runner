# protocol

A check binary talks to the fitness runner by printing one JSON Result on stdout, under the contract go/pkg/checkkit holds.

## SkipNeedsNoFile, SkipNoDiff

The Skipped reasons a run gives for a check it leaves out, which the release smoke test also expects.

## ScopeFiles, ScopeBlob, ScopeGit, ScopeGitHub

The scopes a check declares: what it reads when it judges a repo.

Deprecated: name what a check reads in Files and what else it needs in Requires; Effective maps each scope.

## StageFast, StageTests

The stages a check declares: which total times it. No check has a time limit of its own; a stage's total bounds its checks together.

## CommitMessagePath, HistoryPath, PullBodyPath, CommentBodyPath, BlobPath

The virtual paths of the documents that are not repo files, which a check's Files pattern names as it names files. A check reads one through `--body-file`, or a commit message through `FITNESS_CTX_MESSAGE`.

## BinPrefix

BinPrefix starts the name every built-in check runs under, and every local check's file name.

## BodyPattern

BodyPattern finds the virtual path of every GitHub body and piped text.

## ChecksDirEnv

ChecksDirEnv names the folder of check programs that Discover searches first, ahead of the folder beside fitness and PATH.

## ExitInternal

ExitInternal is the exit code of a check that could not judge the repo.

## FastTotal

FastTotal bounds the wall time of a run's fast checks together. A run past it stops what still runs, fails, and names the slowest checks.

## GitPattern

GitPattern finds the commit message and branch history paths.

## MarkerPrefix

MarkerPrefix starts every marker fitness writes into a comment.

## NetworkEnv

NetworkEnv names the variable through which the runner tells each check that GitHub's API host did not answer its probe: `unreachable:<host>`. Unset means the host answered or the run probed nothing.

## ProbeTimeout

ProbeTimeout bounds the runner's one probe of GitHub's API host.

## Protocol

Protocol is the version of this contract the runner and each check speak. A check whose --describe declares another fails the run before any check starts.

## ProtocolEnv

ProtocolEnv names the variable through which the runner tells each check the protocol version it speaks.

## ScopeFileEnv

ScopeFileEnv names the file, one path per line, that lists the files under a `fitness --dir` folder. A folder can list more paths than one variable holds, so the runner hands them over in a file.

## TestsTotal

TestsTotal bounds the wall time of a run's test runners together. It is 1.5 times the 300s go-test-coverage takes on a large repo.

## Requirements

Requirements lists every requirement this protocol defines, in order.

## Check

Check is one check implementation: its metadata and run function.

Fields:

- `Describe` is the metadata --describe prints.
- `Run` receives the repo root and passthrough args, and the Result it returns is printed as is.
  - An error return means the check could not judge. It shows as info with exit 3, never as a finding.

## ConfigKey

ConfigKey is one key the config file may set, as a check or a registered engine section declares it.

Fields:

- `key` is the key's dotted JSON path, such as proseBudget.maxWords.
- `type` is the key's JSON type: string, integer, number, boolean, array, or object.
- `items` is the JSON type of an array's items.
- `values` is the JSON type of a map's values, whose keys are free names.
- `enum` lists the values a string key accepts, from its enum tag.
- `description` is the doc comment on the key's Go field.
- `default` is the value the key takes when the file leaves it unset, or nil for none.
- `keys` are an object's own keys.

## Describe

Describe is the metadata a check prints for `--describe`.

Fields:

| Key | Holds |
| --- | --- |
| `name` | The check name, as `fitness <name>` runs it. |
| `summary` | What the check enforces, in one line. A built-in check leaves it empty, and the runner serves its own. |
| `stage` | StageTests for a test runner, timed by the test stage's total. Empty or StageFast joins the fast checks. |
| `protocol` | The check protocol version the check speaks. A check that declares none speaks version 1. |
| `minFitness` | The oldest fitness release, as major.minor.patch, that runs the check. An older one stops the run. |
| `config` | The top-level .fitnessrc.json keys the check reads, which the schema, `fitness config` and `fitness explain` show. |
| `files` | An RE2 pattern searched in each repo-relative or virtual path, such as PullBodyPath. Empty matches every path. |
| `exclude` | An RE2 pattern for paths that never reach the check, even when `files` matches them. |
| `needs` | An RE2 pattern a tracked file must match for the check to apply, such as package.json. |
| `requires` | What the check needs beyond its documents. A run lacking one skips the check, as docs/check-protocol.md lists. |
| `tests` | Whether the check reads the repo's requirement tests. Two such checks in a run share one scan. |
| `contextInlineArg` | A flag, such as "--message", whose value the runner moves into `FITNESS_CTX_MESSAGE` and strips from the arguments. |
| `scope` | Deprecated: one of the Scope constants, which Effective maps onto `files` and `requires`. |
| `blob` | Deprecated: whether `fitness --blob` runs the check, which Effective maps onto `files` and `exclude`. |

A scoped run skips the check when no document it judges matches `files`.

## DescribeOf

DescribeOf returns the Describe main hands Main without running the check, or ok false when main never reached Main. Calls must not overlap.

## Effective

Effective maps d's deprecated Scope and Blob onto Files, Exclude and Requires, as docs/check-protocol.md lists, keeping d's own Files.

## Finding

Finding is one problem a check found.

Fields:

- `file` is the path the problem is in, relative to the repo root, or empty when it is not in one file.
- `line` is the 1-based line in File, or 0 when the whole file is meant.
- `column` is the 1-based column on Line, or 0 when not known.
- `rule` names what the check enforces here, in lowercase words joined by dashes, such as unknown-word.
- `message` says what is wrong, in words a person reads.
- `fix` says what to change so the check passes, when one change does.
- `explain` is the command that prints why the check exists. The check's entry point fills it in from the check's name.

## Outcome

Outcome is what one tool run left behind. SpawnErr and Signaled tell a broken environment apart from a tool that ran and judged.

Fields:

- `Code` is the exit code, or 1 when the process ended without one.
- `Stdout` is what the tool printed on standard output.
- `Stderr` is what the tool printed on standard error.
- `SpawnErr` says why the tool could not start, nil when it did.
- `Signaled` is true when a signal ended the tool.
- `Err` is the error the run returned, nil on exit 0.

## Exec

Exec runs cmd with stdout and stderr captured apart. A writer the caller already set on cmd.Stderr, such as os.Stderr for live output, still receives the stream.

## ExecCombined

ExecCombined runs cmd with stdout and stderr in one buffer, in the order the tool wrote them, and returns it as Stdout.

## Outcome.EnvErr

EnvErr is the internal error for a run that never judged: the tool could not start, or a signal ended it. Nil when the tool ran to an exit code.

## Patterns

Patterns are a check's compiled Files, Exclude and Needs. A nil pattern was left empty.

## PatternsOf

PatternsOf compiles the Files, Exclude and Needs of d's Effective fields, naming the field and pattern of the first that does not compile.

## Patterns.Filters

Filters reports whether the patterns narrow which files reach the check.

## Patterns.HasNeeds

HasNeeds reports whether the check declared a Needs pattern.

## Patterns.Matches

Matches reports whether path reaches the check: Files finds it, or is empty, and Exclude does not find it. Paths are repo-relative with `/`.

## Patterns.Met

Met reports whether tracked holds a file Needs finds, or Needs is empty.

## Patterns.ReachedBy

ReachedBy reports whether a run of paths reaches the check. A nil paths is a run of the whole repo, which every check reaches.

## Patterns.Reaches

Reaches reports whether any of paths reaches the check.

## Program

Program is how to start one check.

Fields:

- `Path` is the file to exec.
- `Name`: Name, when set, is the program name the check starts under: the one fitness executable runs the built-in check that name names.

## Builtin

Builtin returns the program for a built-in check: the running fitness executable, started as `fitness-check-<name>`. Every check runs from that one file, so a new install loads one binary instead of one per check.

## Self

Self returns the program that restarts the running check: this executable under its start name, since one fitness executable holds every check.

## Program.Command

Command returns the command that runs the program with args.

## Requirement

Requirement is something a check needs beyond the documents it judges. A run that lacks one skips the check and says why.

## RequiresRepo, RequiresDiff, RequiresHistory, RequiresGitHub, RequiresNetwork

The requirements a check declares in Requires.

## UnknownRequirement

UnknownRequirement returns the first requirement of d this protocol does not define, or "" when it knows them all.

## Requirement.Known

Known reports whether this protocol defines r.

## Result

Result is the single JSON object a check prints on stdout.

Fields:

- `ok` is true when the check found no problem.
- `errors` lists each problem found, one line each.
- `filesChecked` counts the files judged; -1 when the check judged none.
- `info` carries notes that never fail a run, such as an internal error.
- `findings` repeats Errors in a shape an agent can act on without parsing prose: where, which rule, and how to fix it.
  - It is optional, so a check that sets none still speaks the protocol.
  - Errors stays for the readers that predate it.

## Fail

Fail builds a failing Result.

## Found

Found builds a Result from findings: it fails with one error line for each finding, or passes when there are none.

## Info

Info builds a passing Result that carries info notes.

## Pass

Pass builds a passing Result.

## Beside

Beside maps the name of each `fitness-check-<name>` program beside the running fitness executable to its path. It leaves out the links to fitness that an installed bundle makes for built-in checks.

## ChangedFiles

ChangedFiles returns the paths the runner scoped this run to, or nil when the run is unscoped and every file is judged.

## CtxMessage

CtxMessage returns the context-inline commit message and whether the runner provided one at all (present-but-empty differs from absent).

## DiffFiles

DiffFiles returns the paths a commit or pull request changes, or nil when the run changes nothing, so pairing checks judge no pairs.

## Discover

Discover maps each installed `fitness-check-<name>` program to its path: in the ChecksDirEnv folder, beside fitness, then in each absolute PATH folder. The first of a name wins, and a link to a fitness executable never counts.

## Document

Document sets the summary --describe prints when the check declares none. The runner calls it with the summary generated from the check's package doc comment before it starts a built-in check.

## EnabledChecks

EnabledChecks returns the enabled check names for this run, in order.

## ExplainCommand

ExplainCommand is the command that prints why check exists.

## GHEnvironment

GHEnvironment reports whether a failed gh run blames its environment: no start, a signal, or an auth or network failure.

## GHErr

GHErr returns a gh run's error: nil on success, Internal for environment failures, and the run's own error otherwise.

## Internal

Internal marks err as an internal error, reported as info with exit 3. A nil err stays nil.

## IsInternal

IsInternal reports whether err, or an error it wraps, is internal.

## IssueBodyPath

IssueBodyPath is the virtual path of an issue body of kind label kind, such as Plan. An unlabeled issue's kind is `issue`.

## JudgesText

JudgesText reports whether d reads a text through `--body-file`. It sets Blob or scope blob, or its own Files finds the body paths by their folder.

## LastJSONLine

LastJSONLine returns the last nonempty stdout line after any tool noise.

## Lines

Lines renders each finding as an error line with line.

## Main

Main is the entry point every check binary delegates to: it parses --root/--describe, runs the check, and emits the protocol JSON.

## NoVerdict

NoVerdict is the internal error for a tool that exited nonzero without a parseable finding, quoting its last output.

## Offline

Offline is the info note naming skipped reads when GitHub's API host is unreachable, and empty otherwise.

## OfflineNote

OfflineNote says that host did not answer the probe, through which proxy, and what was skipped without connecting.

## Probe

Probe opens one connection to target's host and closes it, through the environment's proxy within ProbeTimeout. A nil error means the host answered, and a proxy of another kind, such as SOCKS, counts as answering.

## ProtocolOf

ProtocolOf is the protocol version d declares, 1 when it declares none.

## ProxyFor

ProxyFor is the proxy `HTTPS_PROXY`, `HTTP_PROXY` and `NO_PROXY` name for target, as Go's HTTP client and gh pick it, or nil for none.

## Requiring

Requiring reports whether d declares r.

## SandboxText

SandboxText returns the first sandbox phrase in output, matched without case, or empty when there is none.

## StageOf

StageOf is the stage d declares, StageFast when it declares none.

## StagedFiles

StagedFiles returns the staged paths the runner provided (may be empty).

## ToolMissing

ToolMissing is the internal error for a tool the check needs but cannot find.

## Unreachable

Unreachable is the host the runner's probe could not reach, from NetworkEnv, or empty when it answered or was not probed.

## UnreachableEnv

UnreachableEnv is the NetworkEnv entry that names host as unreachable.
