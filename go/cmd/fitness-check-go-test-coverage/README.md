---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# go-test-coverage

Require full entry coverage and report gaps below it. Select this check by name; it is not yet in the default suite.

## Run

```sh
fitness go-test-coverage --all
```

Every command package is an entry and must reach 100% statement coverage. Declare library entry packages by their full import paths:

```json
{
  "goTestCoverage": {
    "entries": ["example.com/team/project/api"]
  }
}
```

A module without commands needs a declared entry. Unknown entries fail. Entries add to discovered commands; they cannot hide a command package.

## Results

The check prints file, package, and folder counts to standard error. Entries require 100%; other code targets 100% and reports gaps. Exact counts decide failure, even when a displayed percentage rounds to 100.0%.

```text
entry example.com/team/project/cmd/tool: 9/10 statements; required 100%
```

Entry tests earn coverage in child packages. A child folder does not need separate tests for statements already reached. Untested code remains in the totals. No executable statements means no coverage gap.

This measures statements, not branches or assertion quality.

## Test overlap

The audit runs each top-level test and shared setup twice with fresh profiles. Subtests stay with their parent. It subtracts setup coverage before comparing covered blocks. Changes across repeats mark the group as unstable.

Reports show test names, elapsed time including setup, covered blocks, unique blocks, and identical coverage sets. Unstable sets are flagged and excluded from overlap advice. Tests with no measured production coverage get a separate finding and do not count as identical coverage.

The audit always runs. Exact and partial overlap remain review signals because different inputs and assertions can legitimately execute the same blocks. A top-level test fails when it passes a named helper to `t.Run`; this prevents hiding an independently measured test behind a parent. The legacy `--audit` flag remains accepted.

## Source map

Keep the summary metrics and add line-to-test claims with an explicit export. The map includes test block sets and pairwise intersections, so partial overlap is visible.

```sh
AUDIT_DIR="$(mktemp -d)"
fitness go-test-coverage --all --audit-map="$AUDIT_DIR/claims.json"
```

The output path must be new, with an existing parent outside the repository. A completed measurement writes the map even when entry coverage fails.

## Map fields

Each module has numbered blocks and lines, test claims, and overlapping test pairs. Source paths are relative to the module; `sourceFiles` maps profile names to those paths. IDs are local to each module and repeat consistently for the same claims.

| Field | Meaning |
| --- | --- |
| `blocks` | File, line and column range, statement count, test owners, and shared setup owners. |
| `lines` | Source line, block IDs, test owners, and shared setup owners. |
| `tests` | Test name, stable status, covered block IDs, and setup block IDs. |
| `overlaps` | Two test names, shared block IDs, and shared source-line IDs. |

Go measures statement blocks. The line map projects their ranges; it does not claim that each physical line ran. Tests can share a line while reaching separate blocks: their shared line list is nonempty and their shared block list is empty.

## Claim rules

Only stable test contributions count toward pair comparisons. Shared setup is retained with its own owners, and unstable groups retain their status without confirmed claims. Subtests remain grouped under their top-level test.

Array order and IDs are fixed by source positions and test names. The map omits wall-clock timing; durations remain in the existing summary. Pair entries include partial intersections as well as identical sets.

## Compiled tools

Tests that launch a Go binary must build it with `go build -cover`. They must set the child's `GOCOVERDIR` to the `FITNESS_GO_COVER_DIR` path supplied by this check. The check merges those measured statements with the test profile. Ordinary builds and child runs without collected data earn no credit.

Use the same source and build flags for test and child coverage. Conflicting counts or unknown source paths fail. Standard library and dependency coverage are outside the module inventory.

## Scope and cost

The check uses existing Go module discovery and changed-file scope. Each module runs with its own fresh profiles and workspace mode off. Test errors, missing Go, bad profiles, and missing source coverage fail. Repos without Go modules pass.

All profiles live in private temporary paths and are removed on return. Tests may still write their own files. The audit can be costly because it reruns every top-level test twice. The runner allows fifteen minutes; the check stops work after fourteen.

## Rollout

Both org and external defaults keep their current checks. Select this check explicitly while entry coverage gaps are repaired. Lower-level floors and default adoption need a separate policy decision.

```sh
fitness --policy=external --checks=go-test-coverage --all
```
