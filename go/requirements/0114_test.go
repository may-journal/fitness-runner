package requirements

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
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

// checkNameLine matches a line that is only a check name, as `fitness check
// --list` prints one, apart from the installer's own lines.
var checkNameLine = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// builtinNames is what `fitness check --list` prints: each check the engine
// runs itself.
func builtinNames(t *testing.T, repo string) []string {
	t.Helper()
	out, code := fitness(t, repo, nil, "check", "--list")
	if code != 0 {
		t.Fatalf("fitness check --list exited %d:\n%s", code, out)
	}
	var names []string
	for _, line := range strings.Split(out, "\n") {
		if checkNameLine.MatchString(line) {
			names = append(names, line)
		}
	}
	return names
}

func Test0114_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	index := repoFile(t, "docs/checks.md")
	builtin := builtinNames(t, repo)
	for _, check := range listJSON(t, repo) {
		if !slices.Contains(builtin, check.Name) {
			continue
		}
		row := "| [" + check.Name + "](../go/cmd/fitness-check-" + check.Name + "/doc.go) | " + check.Summary + " |"
		if !strings.Contains(index, row) {
			t.Errorf("docs/checks.md has no index row %q", row)
		}
	}
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
