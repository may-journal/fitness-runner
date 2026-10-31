// cspell:ignore zorbleflux

package requirements

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func Test0003_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	out, _ := fitness(t, repo, nil)
	for _, check := range []string{"eslint", "prettier", "swiftlint", "localized-text", "gofmt", "go-test", "requirements", "requirements-changed"} {
		passedWithNoFiles(t, out, check)
	}
	fixtures := example(t, "happyRepo", map[string]string{"tool/testdata/bad.go": "package bad\nfunc  X( ) {}\n"})
	out, _ = fitness(t, fixtures, nil, "gofmt")
	passedWithNoFiles(t, out, "gofmt")
}

func Test0003_2(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	out, _ := fitness(t, repo, nil)
	cells := rowCells(out)
	want := strings.Fields(catalogRows)
	if len(cells) != len(want) {
		t.Fatalf("fitness printed %d rows, want %d:\n%s", len(cells), len(want), out)
	}
	for i, check := range want {
		if !names(cells[i], check) {
			t.Errorf("row %d is %q, want %s", i+1, cells[i], check)
		}
	}
}

// catalogRows is the order the result rows print in: the repo-wide checks,
// sorted, then the Go, Swift, Node and Java modules' checks, each sorted.
const catalogRows = `
adr-checks
architecture-actors architecture-code architecture-components
architecture-containers architecture-contract architecture-journey
build-output-untracked changelog changelog-bullets changelog-updated
comment-structure commit-attribution cspell doc-template epic-structure
gitignore-why issue-checklist issue-link-once jscpd local-branch
markdown-filename-camel-case
markdown-filename-kebab-case markdown-links
markdown-no-bold-italic mermaid-callout-why mermaid-callouts
mermaid-diagram-prose mermaid-diagram-table-gap mermaid-legend
mermaid-level-bleed mermaid-syntax no-contrastive-reframing no-fitness-front-matter
no-plans-dir plan-structure
plan-trailer pr-closes-issue pr-structure prose-budget read-repo-first
release-changelog repeated-string-literals requirements requirements-changed
semantic-commit text-readability workflow-ref-exists workflow-reporting
go-complexity go-test go-vet gofmt
localized-text swiftlint
dependency-currency eslint no-eslint-disable node-version prettier
vitest-coverage-exclude vitest-coverage-full
checkstyle java-test java-version spotless`

// allPassed is the summary of a run where every installed check passed.
func allPassed() string {
	return fmt.Sprintf("All %d checks passed", len(strings.Fields(catalogRows)))
}

// passedOf is the summary count of a run where all but notPassed installed
// checks passed.
func passedOf(notPassed int) string {
	n := len(strings.Fields(catalogRows))
	return fmt.Sprintf("%d of %d checks passed", n-notPassed, n)
}

// rowCells returns the check cell of each results-table row, top to bottom.
func rowCells(out string) []string {
	var cells []string
	for _, line := range strings.Split(out, "\n") {
		if m := resultRow.FindStringSubmatch(line); m != nil {
			cells = append(cells, m[1])
		}
	}
	return cells
}

func Test0003_3(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{
		"README.md": readme + "\nSee [the guide](gone.md).\n",
		"locked.md": "# Locked\n",
	})
	mustDo(t, os.Chmod(filepath.Join(repo, "locked.md"), 0o000))
	out, code := fitness(t, repo, nil, "markdown-links")
	sees(t, out, code, 1, "locked.md: skipped, cannot be read (permission denied)", "README.md:9: broken relative link: gone.md")
	if !strings.Contains(row(out, "markdown-links"), "failed") {
		t.Errorf("the readable file's finding must still fail, got row %q", row(out, "markdown-links"))
	}
}

func Test0003_4(t *testing.T) {
	t.Parallel()
	out, code, _ := refCheck(t, refWorkflow("may-journal/fitness-runner/.github/workflows/ci-reusable.yml@v1"), []response{
		{Stderr: "error connecting to api.github.com\n", Exit: 1},
	})
	sees(t, out, code, 0, "workflow refs were not verified: gh api repos/may-journal/fitness-runner/git/ref/tags/v1: error connecting to api.github.com")
	if strings.Contains(row(out, "workflow-ref-exists"), "failed") {
		t.Errorf("a check whose service cannot answer must not fail, got row %q", row(out, "workflow-ref-exists"))
	}
}

// inActions runs the one-liner in repo as a GitHub Actions job does, and
// returns what it printed, its exit code, and the job summary it wrote.
func inActions(t *testing.T, repo string) (string, int, string) {
	t.Helper()
	summary := filepath.Join(t.TempDir(), "summary.md")
	out, code := fitness(t, repo, []string{"GITHUB_ACTIONS=true", "GITHUB_WORKSPACE=" + repo, "GITHUB_STEP_SUMMARY=" + summary})
	written, _ := os.ReadFile(summary)
	return out, code, string(written)
}

// misspelled is happyRepo's README with n unknown words, one per line.
func misspelled(n int) map[string]string {
	body := readme
	for i := 0; i < n; i++ {
		body += "\nThe zorbleflux sits here.\n"
	}
	return map[string]string{"README.md": body}
}

func Test0004_1(t *testing.T) {
	t.Parallel()
	out, code, _ := inActions(t, example(t, "happyRepo", misspelled(1)))
	sees(t, out, code, 1, "::error file=README.md,line=9,col=5,title=cspell (unknown-word)::Unknown word (zorbleflux)")
}

func Test0004_2(t *testing.T) {
	t.Parallel()
	out, code, summary := inActions(t, example(t, "happyRepo", misspelled(13)))
	annotated := strings.Count(out, "\n::error ")
	findings := strings.Count(summary, "\n- ")
	want := fmt.Sprintf("::warning::%d more issue(s) not annotated; see the job summary", findings-annotated)
	if annotated != 10 {
		t.Errorf("want 10 annotations at the cap, got %d", annotated)
	}
	sees(t, out, code, 1, want)
}

func Test0004_3(t *testing.T) {
	t.Parallel()
	out, code, _ := inActions(t, example(t, "happyRepo", nil))
	if strings.Contains(out, "::error") || strings.Contains(out, "::warning") || code != 0 {
		t.Errorf("a green run must add no annotations, got exit %d:\n%s", code, out)
	}
}

func Test0004_4(t *testing.T) {
	t.Parallel()
	out, code, _ := inActions(t, example(t, "happyRepo", map[string]string{"docs/notes-100%.md": "The zorbleflux sits here.\n"}))
	sees(t, out, code, 1, "::error file=docs/notes-100%25.md,line=1,col=5,title=cspell (unknown-word)::Unknown word (zorbleflux)%0AFix: ")
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "::") && strings.Contains(line, "100%") && !strings.Contains(line, "100%25") {
			t.Errorf("a percent sign must arrive escaped in one annotation, got %q", line)
		}
	}
}

func Test0004_5(t *testing.T) {
	t.Parallel()
	_, code, summary := inActions(t, example(t, "happyRepo", misspelled(1)))
	sees(t, summary, code, 1, "## Fitness checks", "| cspell | ❌ fail |", "README.md:9:5 - Unknown word (zorbleflux)", "| prose-budget | ✅ pass |")
}

func Test0004_6(t *testing.T) {
	t.Parallel()
	out, code, _ := inActions(t, example(t, "happyRepo", map[string]string{".fitnessrc.json": "{\n"}))
	sees(t, out, code, 1, "::error::fitness setup: .fitnessrc.json:")
	if strings.Contains(out, "::error file=") {
		t.Errorf("a setup failure must not point at a file line:\n%s", out)
	}
}

func Test0004_7(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", misspelled(1))
	summary := filepath.Join(t.TempDir(), "summary.md")
	mustDo(t, os.WriteFile(summary, []byte(strings.Repeat("Earlier step output.\n", 52000)), 0o644))
	_, code := fitness(t, repo, []string{"GITHUB_ACTIONS=true", "GITHUB_WORKSPACE=" + repo, "GITHUB_STEP_SUMMARY=" + summary, "RUNNER_TEMP=" + t.TempDir()})
	written, _ := os.ReadFile(summary)
	sees(t, string(written), code, 1, "Summary truncated.", "Complete report saved on the runner at")
	complete, err := os.ReadFile(summary + ".fitness-report.md")
	if err != nil || !strings.Contains(string(complete), "Unknown word (zorbleflux)") {
		t.Errorf("the complete report must hold every finding: %v", err)
	}
	if len(written) > 1<<20 {
		t.Errorf("summary is %d bytes, over the 1 MiB limit", len(written))
	}
}

// unknownWord is a word no dictionary holds, built so cspell never reads it here.
var unknownWord = strings.Repeat("q", 3) + "xw" + strings.Repeat("v", 3)

// lockedChecks are the file-reading checks beside one finding each, one
// readable file that the check must still judge, and one file it cannot read.
var lockedChecks = []struct {
	check, finding, locked string
	files                  map[string]string
}{
	{"no-eslint-disable", "src/open.ts:1: eslint-disable", "src/locked.ts", map[string]string{
		"src/open.ts": "/* eslint-disable */\n", "src/locked.ts": "export const a = 1;\n"}},
	{"repeated-string-literals", "shared-literal-value", "src/locked.ts", map[string]string{
		"src/a.ts":      "export const a = 'shared-literal-value';\nexport const b = 'shared-literal-value';\n",
		"src/b.ts":      "export const c = 'shared-literal-value';\n",
		"src/locked.ts": "export const d = 1;\n"}},
	{"cspell", "Unknown word (" + unknownWord + ")", "locked.txt", map[string]string{
		"open.txt": unknownWord + "\n", "locked.txt": "hello\n"}},
	{"prose-budget", "docs/open.md", "docs/locked.md", map[string]string{
		"docs/open.md":   "# Open\n\n" + strings.Repeat("This sentence has some words in it. ", 40) + "\n",
		"docs/locked.md": "# Locked\n\nShort.\n"}},
	{"text-readability", "docs/open.md", "docs/locked.md", map[string]string{
		"docs/open.md":   "# Open\n\n" + strings.Repeat("Notwithstanding interdepartmental organizational responsibilities institutional administrators characteristically misunderstand straightforwardly communicated recommendations notwithstanding counterrevolutionary disproportionately uncharacteristically incomprehensible institutionalization. ", 10) + "\n",
		"docs/locked.md": "# Locked\n\nShort.\n"}},
	{"go-complexity", "open.go", "locked.go", map[string]string{
		"go.mod":    "module example.com/m\n\ngo 1.22\n",
		"open.go":   "package m\n\nfunc Open(a int) int {\n\tif a > 1 {\n\t\treturn 1\n\t}\n\tif a > 2 {\n\t\treturn 2\n\t}\n\tif a > 3 {\n\t\treturn 3\n\t}\n\tif a > 4 {\n\t\treturn 4\n\t}\n\tif a > 5 {\n\t\treturn 5\n\t}\n\tif a > 6 {\n\t\treturn 6\n\t}\n\treturn 0\n}\n",
		"locked.go": "package m\n"}},
}

func Test0003_5(t *testing.T) {
	t.Parallel()
	for _, c := range lockedChecks {
		t.Run(c.check, func(t *testing.T) {
			t.Parallel()
			repo := example(t, "happyRepo", c.files)
			mustDo(t, os.Chmod(filepath.Join(repo, filepath.FromSlash(c.locked)), 0o000))
			out, code := fitness(t, repo, nil, c.check)
			sees(t, out, code, 1, c.locked+": skipped, cannot be read (permission denied)", c.finding)
			if !strings.Contains(row(out, c.check), "failed") {
				t.Errorf("the readable file's finding must still fail, got row %q", row(out, c.check))
			}
		})
	}
}

func Test0003_6(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{"go.mod": "module example.com/m\n\ngo 1.22\n", "open.go": "package m\n\nfunc Open( {\n", "locked.go": "package m\n"})
	mustDo(t, os.Chmod(filepath.Join(repo, "locked.go"), 0o000))
	out, code := fitness(t, repo, nil, "go-complexity")
	sees(t, out, code, 1, "locked.go: skipped, cannot be read (permission denied)", "open.go: ")
	if !strings.Contains(row(out, "go-complexity"), "failed") {
		t.Errorf("a file that does not parse must still fail, got row %q", row(out, "go-complexity"))
	}
}

func Test0003_7(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", goRepo())
	mustDo(t, os.Chmod(filepath.Join(repo, "docs", "requirements", "0001-adds-numbers.md"), 0o000))
	out, code := fitness(t, repo, nil, "requirements")
	sees(t, out, code, 1, "docs/requirements/0001-adds-numbers.md: skipped, cannot be read (permission denied)")
	repo = example(t, "happyRepo", goRepo())
	head, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	mustDo(t, err)
	keep := strings.TrimSpace(string(head))
	mustDo(t, filepath.WalkDir(filepath.Join(repo, ".git", "objects"), func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Base(filepath.Dir(p))+filepath.Base(p) != keep {
			err = os.Remove(p)
		}
		return err
	}))
	out, code = fitness(t, repo, nil, "requirements")
	showsInfo(t, out, code, "requirements", "requirement IDs were not checked against git history")
}
