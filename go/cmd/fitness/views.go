package main

import (
	"os"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/comments"
	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/langs"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// withViews builds both views of the scoped files, else of tracked's list,
// once for a run of several file checks. Its Remove deletes them too.
func withViews(root string, checks []resolved, env []string, tracked walkfs.Shared) (walkfs.Shared, []string) {
	if len(checks) < 2 || !slices.ContainsFunc(checks, readsFiles) {
		return tracked, env
	}
	files := scopedFiles(env)
	if files == nil {
		files = tracked.Paths
	}
	views := comments.ShareViews(root, files)
	removeTracked := tracked.Remove
	tracked.Remove = func() {
		views.Remove()
		removeTracked()
	}
	return tracked, append(env, views.Env...)
}

// withTestScan scans the requirement tests once for a run of several checks
// that read them. Its Remove deletes the scan too.
func withTestScan(root string, checks []resolved, env []string, tracked walkfs.Shared) (walkfs.Shared, []string) {
	readers := slices.DeleteFunc(slices.Clone(checks), func(c resolved) bool { return !c.desc.Tests })
	if len(readers) < 2 {
		return tracked, env
	}
	tracked.Remember(root)
	scan, remove := lang.ShareTestScan(root, langs.All)
	removeTracked := tracked.Remove
	tracked.Remove = func() {
		remove()
		removeTracked()
	}
	return tracked, append(env, scan...)
}

// readsFiles reports a check that judges the repo's files.
func readsFiles(c resolved) bool {
	return c.desc.Scope == checkkit.ScopeFiles
}

// scopedFiles lists the files env scopes the run to, or nil when the run
// judges every file. A scope is the changed files or a `--dir` folder's.
func scopedFiles(env []string) []string {
	for _, entry := range env {
		if changed, ok := strings.CutPrefix(entry, changedFilesEnv+"="); ok {
			return lines(changed)
		}
		if name, ok := strings.CutPrefix(entry, checkkit.ScopeFileEnv+"="); ok {
			data, err := os.ReadFile(name)
			if err == nil {
				return lines(string(data))
			}
		}
	}
	return nil
}

// lines splits text into its non-empty lines.
func lines(text string) []string {
	return slices.DeleteFunc(strings.Split(text, "\n"), func(s string) bool { return s == "" })
}
