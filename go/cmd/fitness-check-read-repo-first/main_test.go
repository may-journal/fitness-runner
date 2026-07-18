package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// captureStderr swaps os.Stderr for a pipe around fn and returns what fn
// wrote to it.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = old }()
	fn()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestRunAlwaysPasses(t *testing.T) {
	var res checkkit.Result
	var runErr error
	_ = captureStderr(t, func() {
		res, runErr = run(t.TempDir(), nil)
	})
	if runErr != nil {
		t.Fatal(runErr)
	}
	if !res.Ok || len(res.Errors) != 0 || res.FilesChecked != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestRunWritesBannerFromEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("FITNESS_ENABLED_CHECKS", "changelog\neslint")
	banner := captureStderr(t, func() {
		if _, err := run(t.TempDir(), nil); err != nil {
			t.Error(err)
		}
	})
	for _, want := range []string{
		"changelog", "eslint", "--no-verify",
		"src/checks/changelog/README.md", "src/checks/eslint/README.md",
	} {
		if !strings.Contains(banner, want) {
			t.Fatalf("banner missing %q:\n%s", want, banner)
		}
	}
}

func TestBuildContextFeedbackGolden(t *testing.T) {
	rule := strings.Repeat("─", 80)
	bar28 := strings.Repeat("─", 28)
	bar38 := strings.Repeat("─", 38)
	want := strings.Join([]string{
		rule,
		"Did you familiarize yourself with the decisions logged in the repo,",
		`specifically all "Fitness Checks" that are enabled via @mayjournal/fitness?`,
		"",
		"┌" + bar28 + "┬" + bar38 + "┐",
		"│ Check                      │ Src                                  │",
		"├" + bar28 + "┼" + bar38 + "┤",
		"│ read-repo-first            │ src/checks/read-repo-first/README.md │",
		"└" + bar28 + "┴" + bar38 + "┘",
		"",
		"NOTE: Do not under any circumstance use `--no-verify` as it will cause issues downstream, fixing locally is your best bet.",
		rule,
		"",
	}, "\n")
	got := buildContextFeedback([]string{"read-repo-first"}, 80, palette{})
	if got != want {
		t.Fatalf("feedback mismatch:\ngot:\n%q\nwant:\n%q", got, want)
	}
}

func TestBuildContextFeedbackCases(t *testing.T) {
	cases := []struct {
		name        string
		checks      []string
		contains    []string
		notContains []string
	}{
		{"includes check names and src links", []string{"changelog", "eslint"},
			[]string{"changelog", "eslint", "src/checks/changelog/README.md", "Check", "Src"}, nil},
		{"omits table when empty", nil,
			[]string{"Did you familiarize", "--no-verify"},
			[]string{"│", "│ Check", "Enabled checks:"}},
		{"truncates overlong cells with ellipsis", []string{"markdown-filename-kebab-case"},
			[]string{"│ markdown-filename-kebab-c… │", "│ src/checks/markdown-filename-kebab-… │"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildContextFeedback(tc.checks, 80, palette{})
			for _, want := range tc.contains {
				if !strings.Contains(got, want) {
					t.Fatalf("missing %q in:\n%s", want, got)
				}
			}
			for _, unwanted := range tc.notContains {
				if strings.Contains(got, unwanted) {
					t.Fatalf("unexpected %q in:\n%s", unwanted, got)
				}
			}
		})
	}
}

func TestTableWidths(t *testing.T) {
	cases := []struct {
		cols, first, second int
	}{
		{80, 28, 38},   // the piped default: 70 < 80-8=72
		{120, 28, 78},  // wide terminal
		{40, 28, 36},   // narrow: maxTotal floors at 70
		{0, 28, 36},    // no width at all: same floor
		{78, 28, 36},   // boundary: 78-8=70 exactly
		{200, 28, 158}, // very wide
	}
	for _, tc := range cases {
		first, second := tableWidths(tc.cols)
		if first != tc.first || second != tc.second {
			t.Fatalf("tableWidths(%d) = %d,%d, want %d,%d",
				tc.cols, first, second, tc.first, tc.second)
		}
	}
}

func TestPadCell(t *testing.T) {
	cases := []struct {
		in    string
		width int
		want  string
	}{
		{"read-repo-first", 26, "read-repo-first           "},
		{"markdown-filename-kebab-case", 26, "markdown-filename-kebab-c…"},
		{"src/checks/repeated-string-literals/README.md", 36, "src/checks/repeated-string-literals…"},
		{"exact-width-content-here-x", 26, "exact-width-content-here-x"},
	}
	for _, tc := range cases {
		if got := padCell(tc.in, tc.width); got != tc.want {
			t.Fatalf("padCell(%q, %d) = %q, want %q", tc.in, tc.width, got, tc.want)
		}
	}
}

// TestDecorationBytes pins the piped-output byte parity with cli-table3:
// gray borders in two spans per line, red header cells spanning padding.
func TestDecorationBytes(t *testing.T) {
	p := palette{gray: "\x1b[90m", head: "\x1b[31m", off: "\x1b[39m"}
	got := buildContextFeedback([]string{"read-repo-first"}, 80, p)
	wantHeader := "\x1b[90m│\x1b[39m\x1b[31m Check                      \x1b[39m" +
		"\x1b[90m│\x1b[39m\x1b[31m Src                                  \x1b[39m\x1b[90m│\x1b[39m"
	wantTop := "\x1b[90m┌" + strings.Repeat("─", 28) + "\x1b[39m" +
		"\x1b[90m┬" + strings.Repeat("─", 38) + "┐\x1b[39m"
	wantRow := "\x1b[90m│\x1b[39m read-repo-first            \x1b[90m│\x1b[39m" +
		" src/checks/read-repo-first/README.md \x1b[90m│\x1b[39m"
	for _, want := range []string{wantHeader, wantTop, wantRow} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing exact bytes %q in:\n%q", want, got)
		}
	}
	// the rule and question stay uncolored when only decoration is on
	if !strings.Contains(got, "\n"+strings.Repeat("─", 80)+"\n") {
		t.Fatalf("rule should be plain without chalk colors:\n%q", got)
	}
}

func TestNewPalette(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if p := newPalette(); p != (palette{}) {
		t.Fatalf("NO_COLOR should disable all color, got %+v", p)
	}
}
