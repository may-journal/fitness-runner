package requirements

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// semanticCommit runs `fitness-install -- semantic-commit <args>` on
// happyRepo, whose HEAD subject is "docs(readme): add readme".
func semanticCommit(t *testing.T, args ...string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", nil), nil, append([]string{"semantic-commit"}, args...)...)
}

// semanticCommitTypes is the type list the check prints in every failure.
const semanticCommitTypes = "(types: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert)"

// inherited is an upstream subject with no scope, as a fork's main can hold.
const inherited = "test: migrate from jest to vite (#75)"

// defaultNames are fork default branches with origin/HEAD unset: the usual
// two, and one Fitness cannot guess.
var defaultNames = []string{"main", "master", "develop"}

// forkedRepo is happyRepo on a fresh branch over inherited history that
// origin/<base> holds.
func forkedRepo(t *testing.T, base string) string {
	t.Helper()
	repo := example(t, "happyRepo", nil)
	fork(t, repo, base)
	return repo
}

// fork commits an inherited bad subject to repo without hooks, as a fetch
// does, records it as origin/<base>, and branches fresh.
func fork(t *testing.T, repo, base string) {
	t.Helper()
	out, err := exec.Command("git", "-C", repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit-tree", "HEAD^{tree}", "-p", "HEAD", "-m", inherited).Output()
	mustDo(t, err)
	git(t, repo, "reset", "-q", "--hard", strings.TrimSpace(string(out)))
	remote(t, repo)
	git(t, repo, "update-ref", "refs/remotes/origin/"+base, "HEAD")
	git(t, repo, "checkout", "-q", "-b", "work")
}

// hookedFork is a happyRepo fork with the shared hooks, or with only the
// installer's pre-commit hook when installerHook is set.
func hookedFork(t *testing.T, installerHook bool) (string, []string) {
	t.Helper()
	if !installerHook {
		repo, env := hooked(t, nil)
		fork(t, repo, "main")
		return repo, env
	}
	repo := forkedRepo(t, "main")
	cmd := exec.Command(installer(t), "--install-hook")
	cmd.Dir, cmd.Env = repo, userEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("--install-hook: %v\n%s", err, out)
	}
	return repo, nil
}

// quotesOnly asserts semantic-commit failed on subject and never quoted the
// inherited one.
func quotesOnly(t *testing.T, out string, code int, subject string) {
	t.Helper()
	sees(t, out, code, 1, `Commit message: "`+subject+`" — use type(scope): description `+semanticCommitTypes)
	if strings.Contains(out, inherited) {
		t.Errorf("semantic-commit judged inherited history:\n%s", out)
	}
}

func Test0056_2(t *testing.T) {
	t.Parallel()
	out, code := semanticCommit(t, "--message", "feat: add endpoint")
	sees(t, out, code, 1, `Commit message: "feat: add endpoint" — use type(scope): description `+semanticCommitTypes)
}

func Test0056_7(t *testing.T) {
	t.Parallel()
	// happyRepo's HEAD subject is valid, so an empty message must not fall back to it.
	out, code := semanticCommit(t, "--message=")
	sees(t, out, code, 1, "No commit message to validate; use type(scope): description")
	noRepo := example(t, "happyRepo", nil)
	mustDo(t, os.RemoveAll(filepath.Join(noRepo, ".git")))
	out, code = fitness(t, noRepo, nil, "semantic-commit")
	sees(t, out, code, 1, "No commit message to validate; use type(scope): description")
}

func Test0056_9(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	messages := []string{"Merge branch 'feature' into main", "chore(deps): bump tools\n\nNot a semantic line at all"}
	for _, kind := range []string{"feat", "fix", "docs", "style", "refactor", "perf", "test", "build", "ci", "chore", "revert"} {
		messages = append(messages, kind+"(api): change the endpoint")
	}
	for _, message := range messages {
		out, code := fitness(t, repo, nil, "semantic-commit", "--message", message)
		sees(t, out, code, 0, "All 1 checks passed")
	}
}

func Test0056_10(t *testing.T) {
	t.Parallel()
	for _, subject := range []string{`Revert "feat(api): add endpoint"`, "feat(api)!: drop the old endpoint"} {
		out, code := semanticCommit(t, "--message", subject)
		sees(t, out, code, 1, `Commit message: "`+subject+`" — use type(scope): description`)
	}
}

func Test0056_15(t *testing.T) {
	t.Parallel()
	for _, base := range defaultNames {
		repo := forkedRepo(t, base)
		out, code := fitness(t, repo, nil, "semantic-commit")
		sees(t, out, code, 0, "All 1 checks passed")
		commit(t, repo, nil, "Add notes")
		commit(t, repo, nil, "docs(readme): note more")
		out, code = fitness(t, repo, nil)
		quotesOnly(t, out, code, "Add notes")
		if !strings.Contains(row(out, "semantic-commit"), "failed") {
			t.Errorf("semantic-commit did not fail the full run on %s:\n%s", base, out)
		}
		git(t, repo, "checkout", "-q", "-B", base, "origin/"+base)
		commit(t, repo, nil, "Add base notes")
		out, code = fitness(t, repo, nil, "semantic-commit")
		quotesOnly(t, out, code, "Add base notes")
	}
}

func Test0056_16(t *testing.T) {
	t.Parallel()
	for _, installerHook := range []bool{false, true} {
		repo, env := hookedFork(t, installerHook)
		out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "docs(readme): touch nothing")
		sees(t, out, code, 0, "docs(readme): touch nothing")
		if strings.Contains(out, inherited) {
			t.Errorf("a hook judged the parent commit:\n%s", out)
		}
	}
}

func Test0056_17(t *testing.T) {
	t.Parallel()
	repo, env := hookedFork(t, false)
	out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "Add notes")
	quotesOnly(t, out, code, "Add notes")
	repo, env = hookedFork(t, true)
	out, code = user(t, repo, env, "commit", "--allow-empty", "-m", "Add notes")
	sees(t, out, code, 0, "Add notes")
	out, code = user(t, repo, env, "commit", "--allow-empty", "-m", "docs(readme): note more")
	quotesOnly(t, out, code, "Add notes")
}

func Test0056_14(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	commit(t, repo, nil, "Add notes\n\nBody text")
	out, code := fitness(t, repo, nil, "semantic-commit")
	sees(t, out, code, 1, `Commit message: "Add notes" — use type(scope): description `+semanticCommitTypes)
}
