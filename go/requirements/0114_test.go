package requirements

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoFile reads a file of the fitness repository these tests live in.
func repoFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(path)))
	mustDo(t, err)
	return string(data)
}

func Test0114_2(t *testing.T) {
	t.Parallel()
	readmes, err := filepath.Glob(filepath.Join("..", "cmd", "fitness-check-*", "README.md"))
	mustDo(t, err)
	if len(readmes) > 0 {
		t.Errorf("check READMEs remain in this repo: %v", readmes)
	}
	if !strings.Contains(repoFile(t, "docs/checks.md"), "https://github.com/may-journal/fitness-checks") {
		t.Error("docs/checks.md does not link fitness-checks, where each check's README lives")
	}
}

func Test0114_3(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	readme := repoFile(t, "README.md")
	_, usage, _ := strings.Cut(readme, "## Usage\n")
	_, list, _ := strings.Cut(usage, "```text\n")
	list, _, _ = strings.Cut(list, "```")
	if strings.Count(list, "  fitness ") < 10 {
		t.Fatalf("the README's Usage section has no command list:\n%s", usage)
	}
	out, code := fitness(t, repo, nil, "--help")
	if code != 0 || !strings.Contains(out, "\n"+list+"\n") {
		t.Errorf("fitness --help (exit %d) does not print the README's command list:\n%s\nREADME:\n%s", code, out, list)
	}
}
