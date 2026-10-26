package requirements

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// linkTarget matches a Markdown link's target. A README links a repository
// file relatively and `fitness explain` with a full URL, so a reader
// compares the link text.
var linkTarget = regexp.MustCompile(`\]\([^)]*\)`)

// repoFile reads a file of the fitness repository these tests live in.
func repoFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(path)))
	mustDo(t, err)
	return string(data)
}

// readmeBody is the documentation a check README holds below its title,
// with link targets dropped.
func readmeBody(t *testing.T, check string) string {
	t.Helper()
	readme := repoFile(t, "go/cmd/fitness-check-"+check+"/README.md")
	_, body, ok := strings.Cut(readme, "\n# "+check+"\n\n")
	if !ok {
		t.Fatalf("the %s README has no %q title:\n%s", check, "# "+check, readme)
	}
	return linkTarget.ReplaceAllString(strings.TrimSpace(body), "]")
}

func Test0114_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	for _, check := range listJSON(t, repo) {
		body := linkTarget.ReplaceAllString(strings.TrimSpace(explainJSON(t, repo, check.Name).Body), "]")
		if got := readmeBody(t, check.Name); got != body {
			t.Errorf("the %s README differs from fitness explain:\nREADME:\n%s\n\nexplain:\n%s", check.Name, got, body)
		}
	}
}

func Test0114_2(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	index := repoFile(t, "docs/checks.md")
	for _, check := range listJSON(t, repo) {
		row := "| [" + check.Name + "](../go/cmd/fitness-check-" + check.Name + "/README.md) | " + check.Summary + " |"
		if !strings.Contains(index, row) {
			t.Errorf("docs/checks.md has no index row %q", row)
		}
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
	if code != 0 || !strings.Contains(out, "Usage:\n"+list+"\n") {
		t.Errorf("fitness --help (exit %d) does not print the README's command list:\n%s\nREADME:\n%s", code, out, list)
	}
}
