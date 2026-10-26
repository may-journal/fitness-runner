package requirements

import (
	"encoding/json"
	"strings"
	"testing"
)

// explainedCheck is one check as `fitness explain --json` gives it.
type explainedCheck struct{ Name, Summary, Scope, Body string }

// explainJSON runs `fitness explain <check> --json` in repo and decodes what
// it printed after the installer's own lines.
func explainJSON(t *testing.T, repo, check string) explainedCheck {
	t.Helper()
	out, code := fitness(t, repo, nil, "explain", check, "--json")
	if code != 0 {
		t.Fatalf("fitness explain --json exited %d:\n%s", code, out)
	}
	var got explainedCheck
	start := strings.Index(out, "\n{")
	if start < 0 {
		t.Fatalf("fitness explain --json printed no object:\n%s", out)
	}
	if err := json.Unmarshal([]byte(out[start+1:]), &got); err != nil {
		t.Fatalf("fitness explain --json printed no object: %v\n%s", err, out)
	}
	return got
}

// opening is the first four words of a summary, which a check's Markdown
// documentation opens with, once its escapes are dropped.
func opening(summary string) string {
	words := strings.Fields(summary)
	return strings.Join(words[:min(4, len(words))], " ")
}

func Test0111_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	for _, check := range listJSON(t, repo) {
		out, code := fitness(t, repo, nil, "explain", check.Name)
		sees(t, strings.ReplaceAll(out, `\`, ""), code, 0, "# "+check.Name, "Scope: "+check.Scope, opening(check.Summary))
	}
}

func Test0111_2(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	got := explainJSON(t, repo, "prose-budget")
	summary := listedSummary(t, repo, "prose-budget")
	want := explainedCheck{Name: "prose-budget", Summary: summary, Scope: "files", Body: got.Body}
	if got != want {
		t.Errorf("fitness explain --json gave name %q, scope %q, summary %q; fitness list says %q", got.Name, got.Scope, got.Summary, summary)
	}
	if !opensWith(got.Body, summary) {
		t.Errorf("the body does not open with the summary and go on past it:\n%s", got.Body)
	}
	for _, key := range []string{"## Configuration", "`proseBudget`", "`maxSentenceWords`"} {
		if !strings.Contains(got.Body, key) {
			t.Errorf("the body does not name %s:\n%s", key, got.Body)
		}
	}
}

// listedSummary is the summary `fitness list --json` gives check in repo;
// a check it lists with none fails the test.
func listedSummary(t *testing.T, repo, check string) string {
	t.Helper()
	for _, c := range listJSON(t, repo) {
		if c.Name == check && c.Summary != "" {
			return c.Summary
		}
	}
	t.Fatalf("fitness list --json gives %s no summary", check)
	return ""
}

// opensWith reports whether a Markdown body, once its escapes are dropped,
// opens with summary's first words and says more than summary alone.
func opensWith(body, summary string) bool {
	plain := strings.ReplaceAll(body, `\`, "")
	return strings.HasPrefix(plain, opening(summary)) && len(plain) > len(summary)+1
}

func Test0111_3(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "explain", "prose-budgets")
	sees(t, out, code, 2, `no check named "prose-budgets"; did you mean prose-budget?`)
}

func Test0111_4(t *testing.T) {
	t.Parallel()
	files := localCheck("house-rule", passing)
	files["checks/fitness-check-house-rule"] = "#!/bin/sh\nif [ \"$1\" = --describe ]; then\n  exit 1\nfi\necho '" + passing + "'\n"
	out, code := fitness(t, example(t, "happyRepo", files), nil, "explain", "house-rule")
	sees(t, out, code, 0, "# house-rule", "Note: --describe gave no metadata")
}
