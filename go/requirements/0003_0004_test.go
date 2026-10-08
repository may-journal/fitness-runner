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
	repo := example(t, "happyRepo", nil)
	out, _ := fitness(t, repo, nil)
	for _, check := range []string{"eslint", "prettier", "swiftlint", "gofmt", "go-test", "requirements"} {
		passedWithNoFiles(t, out, check)
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
	out, code, _ := inActions(t, example(t, "happyRepo", misspelled(1)))
	sees(t, out, code, 1, "::error file=README.md,line=9::cspell: README.md:9:5 - Unknown word (zorbleflux)")
}

func Test0004_2(t *testing.T) {
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
	out, code, _ := inActions(t, example(t, "happyRepo", nil))
	if strings.Contains(out, "::error") || strings.Contains(out, "::warning") || code != 0 {
		t.Errorf("a green run must add no annotations, got exit %d:\n%s", code, out)
	}
}

func Test0004_4(t *testing.T) {
	out, code, _ := inActions(t, example(t, "happyRepo", map[string]string{"README.md": readme + "\nThe zorbleflux is 100% here.\n"}))
	sees(t, out, code, 1, "::error file=README.md,line=9::cspell: README.md:9:5 - Unknown word (zorbleflux)")
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "::") && strings.Contains(line, "100%") && !strings.Contains(line, "100%25") {
			t.Errorf("a percent sign must arrive escaped in one annotation, got %q", line)
		}
	}
}

func Test0004_5(t *testing.T) {
	_, code, summary := inActions(t, example(t, "happyRepo", misspelled(1)))
	sees(t, summary, code, 1, "## Fitness checks", "| cspell | ❌ fail |", "README.md:9:5 - Unknown word (zorbleflux)", "| prose-budget | ✅ pass |")
}

func Test0004_6(t *testing.T) {
	out, code, _ := inActions(t, example(t, "happyRepo", map[string]string{".fitnessrc.json": "{\n"}))
	sees(t, out, code, 1, "::error::fitness setup: .fitnessrc.json:")
	if strings.Contains(out, "::error file=") {
		t.Errorf("a setup failure must not point at a file line:\n%s", out)
	}
}

func Test0004_7(t *testing.T) {
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
