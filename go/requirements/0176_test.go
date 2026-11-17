package requirements

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// bodyOnly0176 republishes r's index with the Markdown and Go checks and
// open-questions, a check local-branch needs, whose files find only bodies.
func bodyOnly0176(t *testing.T, r *releases0173) {
	t.Helper()
	r.archives["open-questions"] = archive0173(t, "open-questions")
	r.publish(t, map[string]string{"md-files": "1.0.0", "go-files": "1.0.0", "open-questions": "1.0.0"}, time.Now())
	path := filepath.Join(r.cache, "check-index.json")
	data, err := os.ReadFile(path)
	mustDo(t, err)
	var record struct {
		Value struct {
			Format int                                   `json:"format"`
			Checks map[string]map[string]json.RawMessage `json:"checks"`
		} `json:"value"`
		Checked string `json:"checked"`
	}
	mustDo(t, json.Unmarshal(data, &record))
	var versions map[string]map[string]any
	mustDo(t, json.Unmarshal(record.Value.Checks["open-questions"]["versions"], &versions))
	versions["1.0.0"]["files"] = `^\.fitness/`
	patched, err := json.Marshal(versions)
	mustDo(t, err)
	record.Value.Checks["open-questions"]["versions"] = patched
	data, err = json.Marshal(record)
	mustDo(t, err)
	mustDo(t, os.WriteFile(path, data, 0o600))
}

// checks0176 are installed check programs, by name, whose --describe
// declares the old scope and blob fields, with or without their own files.
var checks0176 = map[string]string{
	"history-rule": `{"name":"history-rule","scope":"git"}`,
	"body-rule":    `{"name":"body-rule","scope":"blob","blob":true}`,
	"md-body":      `{"name":"md-body","scope":"blob","blob":true,"files":"\\.md$"}`,
	"go-text":      `{"name":"go-text","blob":true,"files":"\\.go$"}`,
	"go-files":     `{"name":"go-files","files":"\\.go$"}`,
	"git-text":     `{"name":"git-text","scope":"git","blob":true}`,
}

// bundle0176 is a fitness bundle holding every check in checks0176.
func bundle0176(t *testing.T) string {
	t.Helper()
	bundle := bundle0162(t)
	programs := map[string]string{}
	for name, describe := range checks0176 {
		programs["fitness-check-"+name] = script0172(name, describe)
	}
	write(t, bundle, programs)
	return bundle
}

func Test0176_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{"notes/plan.md": "# Plan\n"})
	out, code := run0172(t, bundle0176(t), repo, "", "--policy=external", "--checks=history-rule,body-rule,md-body", "--dir=notes")
	sees(t, out, code, 1, "history-rule ran", "md-body ran", "Skipped (no matching files): body-rule")
	absent(t, out, "body-rule")
}

func Test0176_2(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	bundle := bundle0176(t)
	selected := "--checks=body-rule,go-text,go-files,git-text"
	out, code := run0172(t, bundle, repo, "Some text.\n", "--blob", "-", selected)
	sees(t, out, code, 1, "body-rule ran", "Skipped (no matching files): go-text", "Skipped (needs files): go-files", "Skipped (requires history): git-text")
	absent(t, out, "go-text", "go-files", "git-text")
	out, code = run0172(t, bundle, repo, "Some text.\n", "--blob", "-", selected, "--as", "x.go")
	sees(t, out, code, 1, "go-text ran", "Skipped (no matching files): body-rule", "Skipped (needs files): go-files")
	absent(t, out, "body-rule", "go-files", "git-text")
}

func Test0176_3(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{
		".fitnessrc.json":                `{"localChecks": "checks"}` + "\n",
		"checks/fitness-check-body-rule": script0172("body-rule", checks0176["body-rule"]),
		"checks/fitness-check-gh-rule":   script0172("gh-rule", `{"name":"gh-rule","scope":"github"}`),
	})
	out, code := fitness(t, repo, nil, "list", "--json")
	sees(t, strings.ReplaceAll(out, `\`, ""), code, 0, `"files": "^.fitness/(github/|blob.md$)"`, `"requires": [`+"\n"+`      "github"`)
	out, code = fitness(t, repo, nil, "explain", "body-rule")
	sees(t, out, code, 0, "Files: `^\\.fitness/(github/|blob\\.md$)` · Exclude: - · Requires: -")
	out, code = fitness(t, repo, nil, "explain", "gh-rule")
	sees(t, out, code, 0, "Requires: github")
}

func Test0176_4(t *testing.T) {
	t.Parallel()
	r := serve0173(t)
	bodyOnly0176(t, r)
	pins := `{"checkVersions": {"md-files": "v1", "go-files": "v1", "open-questions": "v1"}}`
	repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": pins})
	write(t, repo, map[string]string{"notes/plan.md": "# Plan\n"})
	git(t, repo, "add", "-A")
	out, code := run0162(t, bundle0162(t), repo, []string{"FITNESS_CACHE_DIR=" + r.cache}, "--policy=external", "--checks=local-branch,md-files,go-files,open-questions")
	sees(t, out, code, 1, "md-files ran", "Skipped (no matching files): go-files")
	fetchedOnly(t, r, out, "md-files", "open-questions")
}

func Test0176_5(t *testing.T) {
	t.Parallel()
	for files, want := range map[string]int{`^\\.fitness/github/comment/`: 0, `^\\.fitness/github/pull/`: 1} {
		describe := `{"name":"markdown-no-bold-italic","blob":true,"files":"` + files + `"}`
		repo := example(t, "happyRepo", map[string]string{
			".fitnessrc.json": `{"policy": "external", "localChecks": "checks"}` + "\n",
			"checks/fitness-check-markdown-no-bold-italic": script0172("markdown-no-bold-italic", describe),
		})
		out, code := boldPR(t, repo)
		if ran := strings.Contains(out, "markdown-no-bold-italic ran"); code != want || ran != (want == 1) {
			t.Errorf("with files %s, pr-check must exit %d and run the check only when the PR body's path reaches it, got %d:\n%s", files, want, code, out)
		}
	}
}
