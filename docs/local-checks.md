# Local checks

A repo writes its own rules as Go checks in a `fitness/` folder at its root. Fitness builds them, and runs them in the suite, the git hooks and CI, beside the built-in checks. `fitness list` and `fitness explain` show them too.

## Layout

Each folder `fitness/<name>/` is one check, named in lowercase words joined by dashes. It needs no config.

```text
fitness/
  go.mod          # once, for every Go check
  no-todo/
    README.md     # the rule, and how to pass it
    main.go       # the check, on the go/check SDK
    main_test.go  # its unit tests
    config.json   # optional settings the check reads
```

- `README.md` is required. Its first sentence is the check's summary in `fitness list`, and `fitness explain <name>` prints it.
- `main.go` is a Go program on the [check SDK](../go/check/check.go), built with `fitness/go.mod`.
- Folders named `internal` or `testdata`, or starting with `.` or `_`, hold shared code or fixtures, not checks.

## Write a Go check

Add `fitness/go.mod` once:

```sh
cd fitness
go mod init example.com/your-repo/fitness
go get github.com/may-journal/fitness-runner/go@main
```

Then write `fitness/no-todo/main.go`:

```go
package main

import (
	"strings"

	"github.com/may-journal/fitness-runner/go/check"
)

// Rule fails each Swift source that holds a TODO.
var Rule = check.Check{TimeoutMs: 10000, Run: func(ctx *check.Context) (check.Result, error) {
	files := ctx.InScope(ctx.Files(".swift"))
	var found []check.Finding
	for _, f := range files {
		text, err := ctx.Read(f)
		if err != nil {
			return check.Result{}, err
		}
		if strings.Contains(text, "TODO") {
			found = append(found, check.Finding{File: f, Rule: "no-todo", Message: "holds a TODO", Fix: "do it, or file an issue"})
		}
	}
	return check.Found(len(files), found...), nil
}}

func main() { check.Main(Rule) }
```

`fitness check no-todo` builds it into fitness's cache the first time, saying so on stderr, then runs only that check. Later runs reuse the build until the Go source, `go.mod`, `go.sum`, Go or fitness changes. A machine without Go fails the run, naming where to get Go.

## The SDK

A check's `Run` gets a typed `Context`, and returns a `Result`.

- Inputs: `Root`, `Args`, `Changed` for a scoped run, and `Dir`, the check's own folder.
- Files: `Files` lists tracked files by ending, `InScope` narrows them to the run's changes, and `Read` and `ReadJSON` read one.
- Config: `Config` decodes the check's `config.json` into a struct, and `Settings` decodes `.fitnessrc.json`.
- Findings: each `Finding` names its file, line, rule, message and fix, and `Found` turns them into the verdict.
- Metadata: `TimeoutMs` is the check's budget, and a check past it is stopped and fails.
- `MinFitness` names the oldest fitness that runs the check, and an older one stops the run naming the upgrade.
- An error from `Run` reports info, which never fails a run.

## Unit-test a check

`check.Repo` writes a fixture repo, and `check.Judge` runs the check on it, so a test proves the rule both ways:

```go
func TestRule(t *testing.T) {
	bad := check.Repo(t, map[string]string{"App.swift": "// TODO: ship\n"})
	if got := check.Judge(t, Rule, bad); got.Ok {
		t.Error("a TODO must fail")
	}
}
```

`check.Test(t, Rule)` judges the repo the test runs in, as `fitness no-todo` does. `go -C fitness test ./...` runs every check's tests. A repo whose `requirements` check judges Go tests names each test for the acceptance it proves.

## Other executables

Where Go is not wanted, an executable `check` file may stand in for `main.go`, such as a shell script. It speaks the [check protocol](check-protocol.md) itself. It runs with the repo root as its working folder and `--root <dir>` in its arguments, and prints one JSON result.

```sh
#!/bin/sh
[ "$1" = --describe ] && { echo '{"timeoutMs": 10000}'; exit 0; }
n=$(git ls-files '*.swift' | grep -c .)
printf '{"ok":true,"errors":[],"filesChecked":%s}\n' "$n"
```

The `--describe` line is optional. A script that answers no handshake runs once more for it, so answer it when the work is slow.

## ADR and requirement checks

An architecture decision holds only while a check proves it. So in a repo with a `fitness/` folder, each ADR lives in one place: the README of its `fitness/adr-NNNN/` check.

The [adr-checks](../go/cmd/fitness-check-adr-checks/README.md) check fails a numbered ADR kept anywhere else, such as `docs/architecture/adr/`. It also fails an `adr-NNNN` README that lacks a decision section. An ADR whose status line says it is superseded needs no check program beside it. The line under the title is the summary `fitness list` shows.

```markdown
# 0007 — No work bound to scroll

Scrolling never waits on work.

## Context

<!-- Brief context statement not longer than a Tweet. -->

## Decision

<!-- Numbered bullets with (KISS) for each decision. -->

## Consequences

<!-- Numbered bullets with known positive, negative, or neutral consequences. -->

## Check

<!-- What the check fails, and how to pass it. -->
```

A `fitness/req-NNNN/` check may prove a requirement doc in place of tests, when its rule reads the repo itself. The [requirements](../go/cmd/fitness-check-requirements/README.md#requirement-checks) check then needs no test for its acceptances. An ADR whose decision changes without its check's code fails, and a requirement check pairs with its doc as a test does. A paired check that passes having judged no files fails too, since its scan is broken.

## Where checks run

The suite runs every local check after the built-in ones, under org and external policy alike. A local check named like a built-in one replaces it under external policy only, as [ADR 0009](../fitness/adr-0009/README.md) decides. `"localChecks": "<folder>"` in `.fitnessrc.json` names one more folder, read the same way, which may also hold flat `fitness-check-<name>` programs.

A folder check that cannot run fails the run before any check starts. Causes include a missing README or program, a missing `fitness/go.mod`, or Go that does not build. In CI, the reusable workflow installs Go when `fitness/go.mod` exists.
