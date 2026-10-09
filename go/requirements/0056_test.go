package requirements

import (
	"os"
	"path/filepath"
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

func Test0056_1(t *testing.T) {
	repo := example(t, "happyRepo", nil)
	for _, kind := range []string{"feat", "fix", "docs", "style", "refactor", "perf", "test", "build", "ci", "chore", "revert"} {
		out, code := fitness(t, repo, nil, "semantic-commit", "--message", kind+"(api): change the endpoint")
		sees(t, out, code, 0, "All 1 checks passed")
	}
}

func Test0056_2(t *testing.T) {
	out, code := semanticCommit(t, "--message", "feat: add endpoint")
	sees(t, out, code, 1, `Commit message: "feat: add endpoint" — use type(scope): description `+semanticCommitTypes)
}

func Test0056_3(t *testing.T) {
	out, code := semanticCommit(t, "--message", `Revert "feat(api): add endpoint"`)
	sees(t, out, code, 1, `Commit message: "Revert "feat(api): add endpoint"" — use type(scope): description`)
}

func Test0056_4(t *testing.T) {
	out, code := semanticCommit(t, "--message", "feat(api)!: drop the old endpoint")
	sees(t, out, code, 1, `Commit message: "feat(api)!: drop the old endpoint" — use type(scope): description`)
}

func Test0056_5(t *testing.T) {
	out, code := semanticCommit(t, "--message", "Merge branch 'feature' into main")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0056_6(t *testing.T) {
	out, code := semanticCommit(t, "--message", "chore(deps): bump tools\n\nNot a semantic line at all")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0056_7(t *testing.T) {
	// happyRepo's HEAD subject is valid, so an empty message must not fall back to it.
	out, code := semanticCommit(t, "--message=")
	sees(t, out, code, 1, "No commit message to validate; use type(scope): description")
	noRepo := example(t, "happyRepo", nil)
	mustDo(t, os.RemoveAll(filepath.Join(noRepo, ".git")))
	out, code = fitness(t, noRepo, nil, "semantic-commit")
	sees(t, out, code, 1, "No commit message to validate; use type(scope): description")
}

func Test0056_8(t *testing.T) {
	repo := example(t, "happyRepo", nil)
	commit(t, repo, map[string]string{"notes.txt": "notes\n"}, "Add notes\n\nBody text")
	out, code := fitness(t, repo, nil, "semantic-commit")
	sees(t, out, code, 1, `Commit message: "Add notes" — use type(scope): description `+semanticCommitTypes)
}
