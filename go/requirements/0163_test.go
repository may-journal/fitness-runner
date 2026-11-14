package requirements

import (
	"strings"
	"testing"
)

func Test0163_1(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "check-versions")
	sees(t, out, code, 0, "markdown-links v1\n", "gofmt v1\n")
	if strings.Contains(out, "local-branch") {
		t.Errorf("the engine judges local-branch, so no install pins it:\n%s", out)
	}
}

func Test0163_2(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"checkVersions": {"markdown-links": "v1.0.0", "gofmt": "v1"}}`})
	out, code := fitness(t, repo, nil, "check-versions")
	sees(t, out, code, 0, "gofmt v1\nmarkdown-links v1.0.0\n")
	if lines := strings.Count(strings.TrimSpace(pinLines0163(out)), "\n") + 1; lines != 2 {
		t.Errorf("want exactly the two pinned checks, got %d lines:\n%s", lines, out)
	}
}

func Test0163_3(t *testing.T) {
	t.Parallel()
	for config, want := range map[string]string{
		`{"checkVersions": {"gofmt": "1.0"}}`: `checkVersions pins gofmt to "1.0"; use a major such as v1 or a release such as v1.2.3`,
		`{"checkVersions": {"nope": "v1"}}`:   "checkVersions pins nope, which is no check; `fitness list` names them",
	} {
		out, code := fitness(t, example(t, "happyRepo", map[string]string{".fitnessrc.json": config}), nil, "check-versions")
		sees(t, out, code, 1, want)
	}
}

// pinLines0163 keeps the `<name> <pin>` lines of out, dropping the
// installer's own lines.
func pinLines0163(out string) string {
	var kept []string
	for _, line := range strings.Split(out, "\n") {
		if fields := strings.Fields(line); len(fields) == 2 && strings.HasPrefix(fields[1], "v") && !strings.Contains(line, ":") {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}
