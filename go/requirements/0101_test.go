// cspell:ignore zorbleflux

package requirements

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"testing"
)

// finding is one structured finding, as a reader of `--format=json` sees it.
type finding struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
	Fix     string `json:"fix"`
	Explain string `json:"explain"`
}

// jsonRun is what `fitness --format=json` prints on stdout.
type jsonRun struct {
	Ok     bool `json:"ok"`
	Checks []struct {
		Name     string    `json:"name"`
		Status   string    `json:"status"`
		Errors   []string  `json:"errors"`
		Findings []finding `json:"findings"`
	} `json:"checks"`
}

// asJSON runs the suite with `--format=json` in repo as an agent does,
// keeping stdout apart from the stderr table, and decodes stdout.
func asJSON(t *testing.T, repo string) jsonRun {
	t.Helper()
	var stdout, stderr bytes.Buffer
	started(t, func() *exec.Cmd {
		stdout.Reset()
		stderr.Reset()
		cmd := exec.Command(installer(t), "--", "--policy=external", "--all", "--format=json")
		cmd.Dir, cmd.Env = repo, userEnv()
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		return cmd
	}, (*exec.Cmd).Run)
	var run jsonRun
	if err := json.Unmarshal(stdout.Bytes(), &run); err != nil {
		t.Fatalf("stdout must be one JSON object: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	return run
}

// findingsOf returns check's findings in run, failing when it did not run.
func findingsOf(t *testing.T, run jsonRun, check string) []finding {
	t.Helper()
	for _, c := range run.Checks {
		if c.Name == check {
			return c.Findings
		}
	}
	t.Fatalf("the JSON lists no %s check", check)
	return nil
}

// oneFinding asserts check reported exactly one finding at file and line,
// with rule, a fix, and the command that explains check.
func oneFinding(t *testing.T, run jsonRun, check, file string, line int, rule string) {
	t.Helper()
	got := findingsOf(t, run, check)
	if len(got) != 1 {
		t.Fatalf("%s must report one finding, got %+v", check, got)
	}
	f := got[0]
	fixed := f.Fix != ""
	f.Message, f.Fix = "", ""
	if want := (finding{File: file, Line: line, Rule: rule, Explain: "fitness explain " + check}); f != want || !fixed {
		t.Errorf("%s finding must name %s:%d, rule %s, a fix, and its explain command, got %+v", check, file, line, rule, got[0])
	}
}

func Test0101_1(t *testing.T) {
	t.Parallel()
	files := misspelled(1)
	files["README.md"] += "\nSee [the guide](missing.md).\n"
	run := asJSON(t, example(t, "happyRepo", files))
	if run.Ok {
		t.Error("a run with findings must report ok false")
	}
	// The misspelled word sits on line 9, as requirement 0004 shows, and
	// the link two lines below it.
	oneFinding(t, run, "cspell", "README.md", 9, "unknown-word")
	oneFinding(t, run, "markdown-links", "README.md", 11, "broken-relative-link")
}

func Test0101_2(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	commit(t, repo, map[string]string{"notes.txt": "notes\n"}, "updated stuff")
	got := findingsOf(t, asJSON(t, repo), "semantic-commit")
	if len(got) == 0 {
		t.Fatal("semantic-commit must report its error lines as findings")
	}
	want := finding{Rule: "semantic-commit", Explain: "fitness explain semantic-commit"}
	for _, f := range got {
		if (finding{Rule: f.Rule, Explain: f.Explain}) != want || f.Message == "" {
			t.Errorf("a finding from an error line must name the check as its rule and its explain command, got %+v", f)
		}
	}
}

func Test0101_3(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "--policy=external", "--format=yaml")
	sees(t, out, code, 1, "use --format=json or --format=table once")
}

func Test0101_4(t *testing.T) {
	t.Parallel()
	out, code, summary := inActions(t, example(t, "happyRepo", misspelled(1)))
	sees(t, out, code, 1, "::error file=README.md,line=9,col=5,title=cspell (unknown-word)::Unknown word (zorbleflux)"+
		"%0AFix: correct the spelling, or add the word to the words list in cspell.json%0AWhy: fitness explain cspell")
	sees(t, summary, code, 1, "- README.md:9:5 - Unknown word (zorbleflux)\n"+
		"  - Fix: correct the spelling, or add the word to the words list in cspell.json\n"+
		"  - Why: fitness explain cspell\n")
}
