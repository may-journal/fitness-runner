// cspell:ignore zorbleflux

package requirements

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// inActions runs the one-liner in repo as a GitHub Actions job does, returning
// its output, exit code and job summary.
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
	for _, check := range strings.Fields(catalogRows) {
		if !strings.Contains(summary, "| "+check+" |") {
			t.Errorf("the job summary leaves out installed check %s:\n%s", check, summary)
		}
	}
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

// lockedChecks are the file-reading checks, each paired with one readable file
// it must still judge and one it cannot read.
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
