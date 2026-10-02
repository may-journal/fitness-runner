// cspell:ignore Liau
package main

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

func defaults() thresholds {
	return thresholds{maxGrade: defaultMaxGrade, maxLix: defaultMaxLix, minWords: defaultMinWords}
}

func TestFormulasOnKnownCounts(t *testing.T) {
	c := measure("The cat sat. The dog ran.")
	if got := [4]int{c.words, c.letters, c.sentences, c.longWords}; got != [4]int{6, 18, 2, 0} {
		t.Fatalf("counts wrong: %+v", c)
	}
	cli, ari, lix := formulas(c)
	// Hand-computed: CLI = 0.0588*300 - 0.296*(2/6*100) - 15.8; ARI = 4.71*3 + 0.5*3 - 21.43; LIX = 3 + 0.
	for _, tc := range []struct {
		name      string
		got, want float64
	}{{"CLI", cli, -8.0267}, {"ARI", ari, -5.8}, {"LIX", lix, 3.0}} {
		if math.Abs(tc.got-tc.want) > 0.01 {
			t.Fatalf("%s = %.4f, want %.4f", tc.name, tc.got, tc.want)
		}
	}
}

func TestSentenceCounting(t *testing.T) {
	cases := []struct {
		name string
		text string
		want int
	}{
		{"periods split", "One here. Two here. Three here.", 3},
		{"abbreviations protected", "Ports land one at a time, e.g. the mermaid set.", 1},
		{"no boundary still one", "a fragment with no period", 1},
		{"question and bang", "Really? Yes!", 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sentenceCount(tc.text); got != tc.want {
				t.Fatalf("sentenceCount(%q) = %d, want %d", tc.text, got, tc.want)
			}
		})
	}
}

func TestOverCountVote(t *testing.T) {
	th := defaults()
	cases := []struct {
		name          string
		cli, ari, lix float64
		want          int
	}{
		{"all under", 15, 15, 45, 0},
		{"one grade over", 19, 15, 45, 1},
		{"lix alone over", 15, 15, 65, 1},
		{"two grades over", 19, 19, 45, 2},
		{"grade plus lix", 19, 15, 65, 2},
		{"all over", 19, 19, 65, 3},
		{"boundary is over", 18, 18, 60, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := overCount(tc.cli, tc.ari, tc.lix, th); got != tc.want {
				t.Fatalf("overCount(%v,%v,%v) = %d, want %d", tc.cli, tc.ari, tc.lix, got, tc.want)
			}
		})
	}
}

func plainProse() string {
	return strings.Repeat("The runner builds each check and then runs the full suite on every commit. ", 10)
}

func awfulProse() string {
	return strings.Repeat("extraordinary bureaucratic infrastructure considerations regarding administrative complexity ", 20) + "conclude."
}

func TestVerdict(t *testing.T) {
	th := defaults()
	for _, tc := range []struct{ file, text, why string }{
		{"good.md", plainProse(), "plain prose must pass"},
		{"short.md", "Dense incomprehensible bureaucratic verbiage", "short files are never judged"},
	} {
		if errs := verdict(tc.file, measure(tc.text), th); len(errs) != 0 {
			t.Fatalf("%s: %v", tc.why, errs)
		}
	}
	errs := verdict("bad.md", measure(awfulProse()), th)
	if len(errs) != 1 || !containsAll(errs[0], "readability alarm", "bad.md:", "alarm at") {
		t.Fatalf("awful prose must alarm, naming the file and the bands: %v", errs)
	}
}

// containsAll reports whether s contains every one of parts.
func containsAll(s string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}

func TestGuidanceIsLLMFuel(t *testing.T) {
	g := strings.Join(guidance(), "\n")
	for _, want := range []string{
		"Coleman-Liau = 5.88", "ARI = 4.71", "LIX = words/sentence",
		"split sentences", "backticks",
		"go/cmd/fitness-check-text-readability/README.md",
		"docs/research/0001-prose-cognitive-complexity.md",
	} {
		if !strings.Contains(g, want) {
			t.Fatalf("guidance missing %q:\n%s", want, g)
		}
	}
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// runExpecting runs the check and requires the verdict and file count.
func runExpecting(t *testing.T, dir string, args []string, ok bool, files int, label string) checkkit.Result {
	t.Helper()
	res, err := run(dir, args)
	if err != nil || res.Ok != ok || res.FilesChecked != files {
		t.Fatalf("%s: %+v %v", label, res, err)
	}
	return res
}

func TestRunEndToEnd(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "good.md", plainProse())
	write(t, dir, "short.md", "Tiny.")
	runExpecting(t, dir, nil, true, 2, "clean repo")
	write(t, dir, "bad.md", awfulProse())
	res := runExpecting(t, dir, nil, false, 3, "one alarm expected")
	if len(res.Errors) != 4 || !strings.Contains(res.Errors[0], "bad.md:") {
		t.Fatalf("want 1 alarm + 3 guidance lines, alarm first: %v", res.Errors)
	}
	if !strings.Contains(res.Errors[3], "docs/research/") {
		t.Fatalf("guidance points at the methodology docs: %v", res.Errors)
	}
}

func TestRunBodyMode(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "bad-body.md", awfulProse())
	write(t, dir, "good-body.md", plainProse())
	bad, good := filepath.Join(dir, "bad-body.md"), filepath.Join(dir, "good-body.md")
	runExpecting(t, t.TempDir(), []string{"--body-file", bad}, false, 1, "awful description must fail")
	runExpecting(t, t.TempDir(), []string{"--body-file", good}, true, 1, "clean description must pass one file")
}

func TestConfigOverrides(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".fitnessrc.json", `{"textReadability": {"maxGrade": 5, "minWords": 10}}`)
	write(t, dir, "plain.md", plainProse())
	runExpecting(t, dir, nil, false, 1, "lowered ceiling must flag plain prose")
	if th := loadThresholds(dir); th.maxGrade != 5 || th.minWords != 10 || th.maxLix != defaultMaxLix {
		t.Fatalf("config overlay wrong: %+v", th)
	}
}
