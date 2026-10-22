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
	if err := json.Unmarshal([]byte(out[strings.Index(out, "{"):]), &got); err != nil {
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
	got := explainJSON(t, example(t, "happyRepo", nil), "prose-budget")
	want := explainedCheck{Name: "prose-budget", Summary: got.Summary, Scope: "files", Body: got.Body}
	if got != want || got.Summary == "" {
		t.Errorf("fitness explain --json gave name %q, scope %q, summary %q", got.Name, got.Scope, got.Summary)
	}
	for _, key := range []string{"## Configuration", "`proseBudget`", "`maxSentenceWords`"} {
		if !strings.Contains(got.Body, key) {
			t.Errorf("the body does not name %s:\n%s", key, got.Body)
		}
	}
}

func Test0111_3(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "explain", "prose-budgets")
	sees(t, out, code, 2, `no check named "prose-budgets"; did you mean prose-budget?`)
}
