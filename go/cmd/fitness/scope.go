package main

import (
	"os"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// allFlag forces a full scan even when files are staged.
const allFlag = "--all"

// hasAllFlag reports whether argv asks for a full scan.
func hasAllFlag(argv []string) bool {
	for _, a := range argv {
		if a == allFlag {
			return true
		}
	}
	return false
}

// withoutVar returns env minus every entry for the variable name.
func withoutVar(env []string, name string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if !strings.HasPrefix(kv, name+"=") {
			out = append(out, kv)
		}
	}
	return out
}

// changedScope picks the files a run judges, or nil for a full scan. A pull
// request in GitHub Actions scopes to its diff against the base branch, and a
// local run to the staged files. A push to main, --all, a run with nothing
// staged, or a diff git cannot compute all scan everything, so the full
// suite still runs after merge.
func changedScope(root string, all bool) []string {
	if all {
		return nil
	}
	if base := os.Getenv("GITHUB_BASE_REF"); base != "" {
		return gitx.ChangedSince(root, "origin/"+base)
	}
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		return nil
	}
	return gitx.StagedFiles(root)
}
