// cspell:ignore zorbleflux

package requirements

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test0003_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	out, _ := fitness(t, repo, nil)
	for _, check := range []string{"eslint", "prettier", "swiftlint", "localized-text", "gofmt", "go-test", "requirements"} {
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
release-changelog repeated-string-literals requirements semantic-commit
text-readability workflow-ref-exists
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
