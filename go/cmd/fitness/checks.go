package main

import (
	localBranchCheck "github.com/may-journal/fitness-runner/go/cmd/fitness-check-local-branch"
	"github.com/may-journal/fitness-runner/go/docs"
)

// localBranchMain is local-branch's entry point, whose describe metadata
// the engine runs its own judgment under.
var localBranchMain = localBranchCheck.Main

// localBranchDoc is local-branch's rule, from go/docs/runner/local-branch.md.
var localBranchDoc = docs.Runner(localBranch)
