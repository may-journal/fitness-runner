package requirements

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// boldReadme is a readme markdown-no-bold-italic fails.
const boldReadme = readme + "\nSome **bold** text.\n"

// prBase is happyRepo with every check passing and files written over it,
// committed with origin/main at that commit, as a pull request's base.
func prBase(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := example(t, "happyRepo", with(architectedRepo(), files))
	git(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	return repo
}

// prChanges commits files on the branch and runs the suite as a pull
// request's GitHub Actions job does. It returns the output, exit code and job
// summary.
func prChanges(t *testing.T, repo string, files map[string]string) (out string, code int, summary string) {
	t.Helper()
	commit(t, repo, files, "docs(changelog): note the guide")
	path := filepath.Join(t.TempDir(), "summary.md")
	out, code = fitness(t, repo, append([]string{"GITHUB_STEP_SUMMARY=" + path}, prEnv...), "--policy=external")
	written, _ := os.ReadFile(path)
	return out, code, string(written)
}

// noteGuide is a change to happyRepo's changelog that passes every check:
// one more bullet.
func noteGuide(t *testing.T) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "tests", "examples", "happyRepo", "CHANGELOG.md"))
	mustDo(t, err)
	return map[string]string{"CHANGELOG.md": string(raw) + "- Docs: add the guide.\n"}
}

// boldFinding is the error markdown-no-bold-italic reports for boldReadme.
const boldFinding = "README.md: disallowed **bold**"

func Test0138_1(t *testing.T) {
	t.Parallel()
	out, code, summary := prChanges(t, prBase(t, map[string]string{"README.md": boldReadme}), noteGuide(t))
	sees(t, out, code, 1, "Stage 1 of 2: checking the files this pull request changes: 1", allPassed(), "Stage 2 of 2: the changes pass",
		boldFinding, "Stage 2 failed", "Files this pull request does not change:", "- README.md (markdown-no-bold-italic)")
	sees(t, summary, code, 1, "## Stage 1 of 2", "## Stage 2 failed", "- README.md (markdown-no-bold-italic)")
}

func Test0138_2(t *testing.T) {
	t.Parallel()
	out, code, _ := prChanges(t, prBase(t, nil), noteGuide(t))
	sees(t, out, code, 0, "Stage 1 of 2:", "Stage 2 of 2:")
	if strings.Count(out, allPassed()) != 2 {
		t.Errorf("both stages must pass every check:\n%s", out)
	}
}

func Test0138_3(t *testing.T) {
	t.Parallel()
	repo := prBase(t, map[string]string{"README.md": boldReadme, ".fitnessrc.json": `{"scopedPullRequests": true}` + "\n"})
	out, code, _ := prChanges(t, repo, noteGuide(t))
	sees(t, out, code, 0, "Stage 1 of 1:", allPassed(),
		"Stage 2 skipped: .fitnessrc.json sets scopedPullRequests, so only a push to main scans every file")
	if strings.Contains(out, boldFinding) || strings.Contains(out, "Stage 2 of") {
		t.Errorf("the run judged a file the pull request does not change:\n%s", out)
	}
}

func Test0138_5(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, nil)
	git(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	write(t, repo, noteGuide(t))
	git(t, repo, "add", "CHANGELOG.md")
	git(t, repo, "-c", "core.hooksPath=", "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "docs(changelog): note the guide")
	write(t, repo, map[string]string{"README.md": readme + "\nMore words.\n"})
	git(t, repo, "add", "README.md")
	out, _ := user(t, repo, append(env, prEnv...), "commit", "-q", "-m", "docs(readme): add words")
	if !strings.Contains(out, "Running checks:") || strings.Contains(out, "Stage 1 of") {
		t.Errorf("the pre-commit hook must run the checks once, in no stages:\n%s", out)
	}
}

func Test0138_4(t *testing.T) {
	t.Parallel()
	out, code, _ := prChanges(t, prBase(t, nil), map[string]string{"README.md": boldReadme})
	sees(t, out, code, 1, "Stage 1 of 2:", boldFinding, "Stage 2 skipped: stage 1 failed")
	if strings.Contains(out, "Stage 2 of 2") {
		t.Errorf("stage 2 ran after stage 1 failed:\n%s", out)
	}
}
