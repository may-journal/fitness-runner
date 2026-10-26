package requirements

import (
	"encoding/json"
	"strings"
	"testing"
)

// listedCheck is one check as `fitness list --json` gives it.
type listedCheck struct {
	Name     string `json:"name"`
	Summary  string `json:"summary"`
	Scope    string `json:"scope"`
	BudgetMs int    `json:"budgetMs"`
	Enabled  bool   `json:"enabled"`
}

// listJSON runs `fitness list --json` in repo and decodes what it printed
// after the installer's own lines.
func listJSON(t *testing.T, repo string) []listedCheck {
	t.Helper()
	out, code := fitness(t, repo, nil, "list", "--json")
	if code != 0 {
		t.Fatalf("fitness list --json exited %d:\n%s", code, out)
	}
	var checks []listedCheck
	start := strings.Index(out, "\n[")
	if start < 0 {
		t.Fatalf("fitness list --json printed no check list:\n%s", out)
	}
	if err := json.Unmarshal([]byte(out[start+1:]), &checks); err != nil {
		t.Fatalf("fitness list --json printed no check list: %v\n%s", err, out)
	}
	return checks
}

// listRow returns the `fitness list` line for check, or "".
func listRow(out, check string) string {
	for _, line := range strings.Split(out, "\n") {
		if fields := strings.Fields(line); len(fields) > 0 && fields[0] == check {
			return line
		}
	}
	return ""
}

func Test0110_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	out, code := fitness(t, repo, nil, "list")
	sees(t, out, code, 0, "CHECK SCOPE BUDGET ENABLED SUMMARY", "Lists Go files that gofmt would change.")
	for _, check := range strings.Fields(catalogRows) {
		if !described(listRow(out, check)) {
			t.Errorf("fitness list shows %s without a scope, budget and summary: %q", check, listRow(out, check))
		}
		if strings.Count(out, "\n"+check+" ") != 1 {
			t.Errorf("fitness list shows %s other than once:\n%s", check, out)
		}
	}
}

// described reports whether a `fitness list` row shows a scope, a budget in
// seconds, and a summary.
func described(row string) bool {
	fields := strings.Fields(row)
	return len(fields) >= 5 && fields[1] != "-" && strings.HasSuffix(fields[2], "s") && fields[4] != "-"
}

func Test0110_2(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"policy": "external", "disabledChecks": ["gofmt"]}` + "\n"})
	checks := listJSON(t, repo)
	if len(checks) != len(strings.Fields(catalogRows)) {
		t.Fatalf("fitness list --json gave %d checks, want %d", len(checks), len(strings.Fields(catalogRows)))
	}
	for _, c := range checks {
		if c.Enabled == (c.Name == "gofmt") {
			t.Errorf("%s reads as enabled=%v", c.Name, c.Enabled)
		}
	}
}

func Test0110_3(t *testing.T) {
	t.Parallel()
	files := localCheck("house-rule", passing)
	files["checks/fitness-check-house-rule"] = "#!/bin/sh\nif [ \"$1\" = --describe ]; then\n  echo '{\"name\": \"house-rule\", \"summary\": \"Keeps the house rule.\", \"scope\": \"files\"}'\n  exit 0\nfi\necho '" + passing + "'\n"
	out, code := fitness(t, example(t, "happyRepo", files), nil, "list")
	sees(t, out, code, 0, "Keeps the house rule.")
	if fields := strings.Fields(listRow(out, "house-rule")); len(fields) < 4 || fields[1] != "files" || fields[3] != "yes" {
		t.Errorf("fitness list shows the local check as %q, want scope files and enabled", listRow(out, "house-rule"))
	}
}

func Test0110_4(t *testing.T) {
	t.Parallel()
	files := localCheck("house-rule", passing)
	files["checks/fitness-check-house-rule"] = "#!/bin/sh\nif [ \"$1\" = --describe ]; then\n  exit 1\nfi\necho '" + passing + "'\n"
	out, code := fitness(t, example(t, "happyRepo", files), nil, "list")
	sees(t, out, code, 0, "Lists Go files that gofmt would change.")
	if row := listRow(out, "house-rule"); !strings.Contains(row, "(--describe gave no metadata") {
		t.Errorf("fitness list shows the broken local check as %q, want a note saying why", row)
	}
}
