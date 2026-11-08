package requirements

import (
	"fmt"
	"strings"
	"testing"
)

// largeDiffFiles is how many product files the large diff edits, so with its
// test, doc and changelog entry it changes 266 files.
const largeDiffFiles = 263

// largeDiffChecks are the checks a large staged diff must keep fast.
var largeDiffChecks = []string{"requirements-changed", "requirements", "changelog-updated"}

// counter is product file i of the large diff, returning i plus step.
func counter(i int, step string) string {
	return fmt.Sprintf("package counters\n\n// Counter%03d returns its number.\nfunc Counter%03d() int {\n\treturn %d%s\n}\n", i, i, i, step)
}

// counters lists the large diff's product files, each returning its number
// plus step.
func counters(step string) map[string]string {
	files := map[string]string{}
	for i := range largeDiffFiles {
		files[fmt.Sprintf("counters/counter%03d.go", i)] = counter(i, step)
	}
	return files
}

// largeDiff is every counter's code changed, with goRepo's test, doc and a
// changelog entry that names the change.
func largeDiff() map[string]string {
	return with(counters(" + 1 // counter offset"), map[string]string{
		"adder_test.go": edited(goRepo()["adder_test.go"]),
		pairedDoc:       reworded(goRepo()[pairedDoc]),
		"changelog/2026-11-08-1400-counter-offset.md": bareEntryText("2026.11.08.1400", "Fix: every counter will return its number plus an offset, by hand."),
	})
}

func Test0151_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", with(goRepo(), counters("")))
	write(t, repo, largeDiff())
	git(t, repo, "add", "-A")
	path, calls := loggedGit(t)
	out, code := fitness(t, repo, []string{path}, "--policy=external", "--checks="+strings.Join(largeDiffChecks, ","))
	passesBatched(t, out, code, largeDiffChecks, calls)
	pull := example(t, "happyRepo", with(goRepo(), counters("")))
	git(t, pull, "update-ref", "refs/remotes/origin/main", "HEAD")
	commit(t, pull, largeDiff(), "fix(counters): add an offset")
	path, calls = loggedGit(t)
	out, code = fitness(t, pull, append([]string{path}, prEnv...), "--policy=external", "--checks="+strings.Join(largeDiffChecks[:2], ","))
	passesBatched(t, out, code, largeDiffChecks[:2], calls)
}

// maxLargeDiffGitCalls bounds the git calls of a whole run on the large diff,
// runner included, far below one per changed file.
const maxLargeDiffGitCalls = 40

// passesBatched asserts checks passed the large diff, and the run started git
// a few times in all, not once per file.
func passesBatched(t *testing.T, out string, code int, checks []string, calls string) {
	t.Helper()
	sees(t, out, code, 0, fmt.Sprintf("All %d checks passed", len(checks)))
	n := occurrences(calls, "\n")
	t.Logf("git started %d times", n)
	if n == 0 || n > maxLargeDiffGitCalls {
		t.Errorf("the run started git %d times on a %d-file diff, want 1 to %d, none per file", n, len(largeDiff()), maxLargeDiffGitCalls)
	}
}
