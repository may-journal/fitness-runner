---
relatedConfigurations: ['../.fitnessrc.json']
---

# Local checks

A repo writes its own rules as checks in a `fitness/` folder at its root. Fitness runs them in the suite, the git hooks and CI, beside the built-in checks. `fitness list` and `fitness explain` show them too.

## Layout

Each folder `fitness/<name>/` is one check, named in lowercase words joined by dashes. It needs no config.

```text
fitness/
  go.mod          # only for Go checks
  no-todo/
    README.md     # the rule, and how to pass it
    check         # an executable, or
    main.go       # Go source that fitness builds
```

- `README.md` is required. Its first sentence is the check's summary in `fitness list`, and `fitness explain <name>` prints it.
- `check` is any executable file, such as a shell script. It must have its execute bit set.
- Without `check`, the folder's Go source is built with `fitness/go.mod`.
- Folders named `internal` or `testdata`, or starting with `.` or `_`, hold shared code or fixtures, not checks.

A check speaks the [check protocol](check-protocol.md). It runs with the repo root as its working folder and `--root <dir>` in its arguments. It prints one JSON result on stdout and exits 0 when it passed or 1 when it found problems.

## A shell check

```sh
#!/bin/sh
# fitness/no-todo/check: no TODO in Swift sources.
[ "$1" = --describe ] && { echo '{"timeoutMs": 10000}'; exit 0; }
files=$(git ls-files '*.swift')
n=$(printf '%s\n' "$files" | grep -c .)
bad=$(grep -l TODO $files 2>/dev/null)
[ -z "$bad" ] && { printf '{"ok":true,"errors":[],"filesChecked":%s}\n' "$n"; exit 0; }
printf '{"ok":false,"errors":["%s: remove the TODO"],"filesChecked":%s}\n' "$bad" "$n"
exit 1
```

The `--describe` line is optional for a folder check. It declares a budget in `timeoutMs`, else the check gets the five-second default or the repo's `timeoutMs`. A check that answers no handshake runs once more for it, so answer it when the work is slow. A check past its budget is stopped and fails.

## A Go check

The [check SDK](../go/check/check.go) does the protocol for you. Add `fitness/go.mod` once:

```sh
cd fitness
go mod init example.com/your-repo/fitness
go get github.com/may-journal/fitness-runner/go@main
```

Then write `fitness/no-todo/main.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/check"
)

var rule = check.Check{TimeoutMs: 10000, Run: func(root string, _ []string) (check.Result, error) {
	files := check.Files(root, ".swift")
	var errs []string
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			return check.Result{}, err
		}
		if strings.Contains(string(data), "TODO") {
			errs = append(errs, f+": remove the TODO")
		}
	}
	if len(errs) > 0 {
		return check.Fail(len(files), errs...), nil
	}
	return check.Pass(len(files)), nil
}}

func main() { check.Main(rule) }
```

Fitness builds it into its cache the first time it runs, and says so on stderr. Later runs reuse the build until the Go source, `go.mod`, `go.sum`, Go or fitness changes. A machine without Go fails the run, naming where to get Go.

## The SDK

`check.Files` lists tracked files by ending, and `check.ChangedFiles` lists the files a commit or pull request changes. An error from `Run` reports info, which never fails a run. Set `MinFitness` when the check needs a newer fitness, and an older one stops the run naming the upgrade.

## Run one check alone

`fitness check no-todo`, or `fitness no-todo`, runs only that check, as a unit test does. A Go check also runs as a Go test:

```go
// fitness/no-todo/main_test.go
func TestRule(t *testing.T) { check.Test(t, rule) }
```

`go -C fitness test ./...` then judges the repo with every Go check that has such a test. A repo whose `requirements` check judges Go tests names each test for the acceptance it proves.

## ADR and requirement checks

An architecture decision holds only while a check proves it. So in a repo with a `fitness/` folder, the [adr-checks](../go/cmd/fitness-check-adr-checks/README.md) check pairs each numbered ADR with a `fitness/adr-NNNN/` check. The ADR links the check's README, and the README links the ADR. An ADR whose status line says it is superseded needs none.

```markdown
# 0007 — No work bound to scroll

Check: [fitness/adr-0007](../../../fitness/adr-0007/README.md)
```

A `fitness/req-NNNN/` check may prove a requirement doc in place of tests, when its rule reads the repo itself. The [requirements](../go/cmd/fitness-check-requirements/README.md#requirement-checks) check then needs no test for its acceptances. Either way, a change to the doc alone, or to its check alone, fails. A paired check that passes having judged no files fails too, since its scan is broken.

## Where checks run

The suite runs every local check after the built-in ones, under org and external policy alike. A local check named like a built-in one replaces it under external policy only, as [ADR 0009](architecture/adr/0009-repos-add-checks-org-keeps-the-bar.md) decides. `"localChecks": "<folder>"` in `.fitnessrc.json` names one more folder, read the same way, which may also hold flat `fitness-check-<name>` programs.

A folder check that cannot run fails the run before any check starts. Causes include a missing README or program, a missing `fitness/go.mod`, or Go that does not build. In CI, the reusable workflow installs Go when `fitness/go.mod` exists.
