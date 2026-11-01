package requirements

import (
	"strings"
	"testing"
)

// unchangedTestFails checks ownership when staged and PR diffs leave tests alone.
func unchangedTestFails(t *testing.T, files map[string]string, want string) {
	t.Helper()
	for _, path := range []string{"README.md", "docs/requirements/0001-adds-numbers.md"} {
		for _, pr := range []bool{false, true} {
			repo := example(t, "happyRepo", files)
			git(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
			write(t, repo, map[string]string{path: files[path] + "\n"})
			git(t, repo, "add", path)
			var env []string
			if pr {
				commit(t, repo, nil, "docs: change prose")
				env = []string{"GITHUB_ACTIONS=true", "GITHUB_BASE_REF=main"}
			}
			out, code := fitness(t, repo, env, "requirements")
			sees(t, out, code, 1, want)
		}
	}
}

// Paths and edits the pairing tests share.
const (
	pairedTest = "reqs/0001_test.go"
	pairedDoc  = "docs/requirements/0001-adds-numbers.md"
	twoTest    = "reqs/0001_0002_test.go"
)

// edited is body with one more function, a code change that keeps Go and
// Swift valid.
func edited(body string) string {
	return body + "\nfunc oneMore() {}\n"
}

// withComment is body with one more comment line, a change to no code.
func withComment(body string) string {
	return body + "\n// One more line.\n"
}

// reworded is a requirement doc with one more word in its Why, changing what
// it says, not only its formatting.
func reworded(doc string) string {
	return strings.Replace(doc, "by hand.", "by hand at all.", 1)
}

// stagedRun writes files over a fresh copy of given, stages them, and runs
// the requirements check on the staged changes.
func stagedRun(t *testing.T, given, files map[string]string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", given)
	write(t, repo, files)
	git(t, repo, "add", "-A")
	return fitness(t, repo, nil, "requirements")
}

// prRun commits files with message on a pull request branch off a fresh copy
// of given and runs the requirements check on it.
func prRun(t *testing.T, given, files map[string]string, message string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", given)
	git(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	write(t, repo, files)
	git(t, repo, "add", "-A")
	git(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", message)
	return fitness(t, repo, prEnv, "requirements")
}

// prEnv is the GitHub Actions environment of a pull request into main.
var prEnv = []string{"GITHUB_ACTIONS=true", "GITHUB_BASE_REF=main"}

// replayed replays an unpaired --no-verify test change over the same line
// changed with its doc, returning what the first failing commit printed.
func replayed(t *testing.T, replay []string, finish ...[]string) (out string, code int) {
	t.Helper()
	repo, env := hooked(t, pairedRepo())
	base := testFile("reqs", "Test0001_1")
	userSteps(t, repo, env, []string{"add", "-A"}, []string{"commit", "-q", "-m", "chore(hooks): add the fitness hooks"})
	write(t, repo, map[string]string{pairedTest: strings.Replace(base, "{}", "{ _ = 1 }", 1)})
	userSteps(t, repo, env, []string{"checkout", "-q", "-b", "side"}, []string{"add", "-A"},
		[]string{"commit", "-q", "--no-verify", "-m", "test(reqs): reword the test"}, []string{"checkout", "-q", "-"})
	write(t, repo, map[string]string{pairedTest: strings.Replace(base, "{}", "{ _ = 2 }", 1), pairedDoc: reworded(requirementDoc("0001.1"))})
	userSteps(t, repo, env, []string{"add", "-A"}, []string{"commit", "-q", "-m", "test(reqs): check sums again"})
	if _, code := user(t, repo, env, replay...); code == 0 {
		t.Fatalf("git %v did not conflict", replay)
	}
	userSteps(t, repo, env, []string{"checkout", "--theirs", pairedTest}, []string{"add", "-A"})
	for _, args := range finish {
		if out, code = user(t, repo, env, append([]string{"commit"}, args...)...); code != 0 {
			break
		}
	}
	return out, code
}

// mergedBase is a branch that merged its moved base, whose new commit
// changes a requirement doc alone. It returns the repo, with origin/main at
// the base's new tip.
func mergedBase(t *testing.T) string {
	t.Helper()
	repo := example(t, "happyRepo", pairedRepo())
	git(t, repo, "checkout", "-q", "-b", "base")
	commit(t, repo, map[string]string{pairedDoc: reworded(requirementDoc("0001.1"))}, "docs(requirements): reword 0001")
	git(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	git(t, repo, "checkout", "-q", "-b", "work", "HEAD~1")
	commit(t, repo, map[string]string{"README.md": readme + "\nMore words.\n"}, "docs(readme): add words")
	git(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "merge", "-q", "--no-edit", "base")
	return repo
}

// userSteps runs each git command as a user, failing the test on the first
// one that fails.
func userSteps(t *testing.T, repo string, env []string, steps ...[]string) {
	t.Helper()
	for _, step := range steps {
		if out, code := user(t, repo, env, step...); code != 0 {
			t.Fatalf("git %v: exit %d\n%s", step, code, out)
		}
	}
}
