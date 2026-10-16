package requirements

import (
	"strings"
	"testing"
)

// cleanText is a short draft every prose check accepts.
const cleanText = "Short and clean.\n"

func Test0077_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{"README.md": readme + "\nSome **bold** text.\n"})
	out, code := fitness(t, repo, nil, "--blob", cleanText)
	if strings.Contains(out, "README.md") {
		t.Errorf("--blob must judge only the text, not repo files:\n%s", out)
	}
	for _, check := range []string{"cspell", "markdown-no-bold-italic", "prose-budget", "plan-structure"} {
		if row(out, check) == "" {
			t.Errorf("blob-capable %s did not judge the text:\n%s", check, out)
		}
	}
	// A plain draft is no Plan or PR body, so those structure checks fail.
	sees(t, out, code, 1, "Skipped (needs files):", "jscpd", "go-test", "markdown-links")
	if row(out, "jscpd") != "" {
		t.Errorf("jscpd needs files and must be skipped, not run:\n%s", out)
	}
}

func Test0077_2(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	out, code := fitness(t, repo, nil, "--blob="+cleanText, "--checks=cspell,markdown-no-bold-italic,jscpd")
	sees(t, out, code, 0, "All 2 checks passed", "Skipped (needs files): jscpd")
	passedWith(t, out, "cspell", "1")
	if row(out, "prose-budget") != "" {
		t.Errorf("--checks must narrow the run to the named checks:\n%s", out)
	}
}

func Test0077_3(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	out, code := piped(t, repo, "Some **bold** text.\n", nil, "--blob", "-", "--checks=markdown-no-bold-italic")
	sees(t, out, code, 1, "[markdown-no-bold-italic] Please fix these items:", "disallowed **bold**", "0 of 1 checks passed, 1 failed")
}

func Test0077_5(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"timeoutMs": 1}` + "\n"})
	out, code := fitness(t, repo, nil, "--blob", cleanText, "--checks=cspell")
	sees(t, out, code, 1, "Check timed out after 0.001s")
}

func Test0077_4(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	for _, args := range [][]string{
		{"--blob", ""},
		{"--blob", cleanText, "--all"},
		{"cspell", "--blob", cleanText},
	} {
		out, code := fitness(t, repo, nil, args...)
		sees(t, out, code, 2, "fitness --blob")
		if strings.Contains(out, "Running checks:") {
			t.Errorf("%q must run no check:\n%s", args, out)
		}
	}
}
