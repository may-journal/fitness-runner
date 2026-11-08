package requirements

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// proseChecks are the checks that read every doc and comment.
var proseChecks = []string{"text-readability", "prose-budget", "cspell", "mermaid-callouts", "repeated-string-literals"}

// proseFiles is how many docs, and how many commented Go files, proseRepo
// holds.
const proseFiles = 150

// maxProseGitCalls bounds the git calls of the whole run, runner included,
// far below one per file; a run makes about seven.
const maxProseGitCalls = 20

// proseRepo is goRepo with proseFiles short docs and as many commented Go
// files, each with its own string literal.
func proseRepo() map[string]string {
	files := goRepo()
	for i := range proseFiles {
		files[fmt.Sprintf("docs/notes/note%03d.md", i)] = fmt.Sprintf("# Note %d\n\nThis note keeps the plan for the garden. We water the plants each morning.\n", i)
		files[fmt.Sprintf("garden/plant%03d.go", i)] = fmt.Sprintf("package garden\n\n// Plant%03d names one plant in the garden.\nconst Plant%03d = \"plant %d\"\n", i, i, i)
	}
	return files
}

func Test0152_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", proseRepo())
	path, calls := loggedGit(t)
	out, code := fitness(t, repo, []string{path}, "--policy=external", "--checks="+strings.Join(proseChecks, ","), "--all")
	t.Log("\n" + out)
	sees(t, out, code, 0, "All 5 checks passed")
	for _, check := range proseChecks {
		took := rowMillis.FindStringSubmatch(row(out, check))
		if took == nil {
			t.Fatalf("%s shows no time in its row %q", check, row(out, check))
		}
		if ms, _ := strconv.Atoi(took[1]); ms >= 5000 {
			t.Errorf("%s took %d ms on %d files, want under 5000 ms", check, ms, 2*proseFiles)
		}
	}
	n := occurrences(calls, "\n")
	t.Logf("git started %d times", n)
	if n > maxProseGitCalls {
		t.Errorf("the prose checks started git %d times on %d files, want at most %d, none per file", n, 2*proseFiles, maxProseGitCalls)
	}
}
