package requirements

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// swiftlintArgs is the strict JSON lint fitness asks SwiftLint for.
const swiftlintArgs = "lint --strict --reporter json --quiet ."

// swiftlintShort is what `swiftlint lint --strict --reporter json` prints for
// `let ab = 1` in file: a warning made an error by --strict, with a column.
const swiftlintShort = `[
  {
    "character" : 5,
    "file" : "%s",
    "line" : 1,
    "reason" : "Variable name 'ab' should be between 3 and 40 characters long",
    "rule_id" : "identifier_name",
    "severity" : "Error",
    "type" : "Identifier Name"
  }
]
`

// swiftlintNoNewline is SwiftLint's JSON for a file missing its trailing
// newline: a whole-file finding, so the column is null.
const swiftlintNoNewline = `[
  {
    "character" : null,
    "file" : "%s",
    "line" : 1,
    "reason" : "Files should have a single trailing newline",
    "rule_id" : "trailing_newline",
    "severity" : "Error",
    "type" : "Trailing Newline"
  }
]
`

// swiftlintClean is SwiftLint's JSON for a run with no findings.
var swiftlintClean = response{Match: []string{swiftlintArgs}, Stdout: "[\n\n]\n"}

// swiftlintNothingLeft is SwiftLint's answer when its config excludes every
// Swift file: an error exit naming no lintable files.
var swiftlintNothingLeft = response{
	Match:  []string{swiftlintArgs},
	Stderr: "No lintable files found at paths: '.'\n",
	Exit:   1,
}

// swiftlintFindings is SwiftLint's strict answer for one finding in file of
// repo: the JSON array and exit 2, since --strict makes warnings errors.
func swiftlintFindings(repo, file, body string) response {
	path := filepath.Join(repo, file)
	return response{Match: []string{swiftlintArgs}, Stdout: fmt.Sprintf(body, path), Exit: 2}
}

// swiftlintRepo makes happyRepo with files written over it.
func swiftlintRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	return example(t, "happyRepo", files)
}

// swiftlintRun runs `fitness-install -- swiftlint` on repo with SwiftLint
// answering saved responses, and returns the stand-ins with the result.
func swiftlintRun(t *testing.T, repo string, saved ...response) (replays, string, int) {
	t.Helper()
	rp := standIn(t, map[string][]response{"swiftlint": saved})
	out, code := fitness(t, repo, rp.env, "swiftlint")
	return rp, out, code
}

// swiftlintAsked asserts fitness ran SwiftLint's strict JSON lint once.
func swiftlintAsked(t *testing.T, rp replays) {
	t.Helper()
	if calls := rp.calls("swiftlint"); len(calls) != 1 || calls[0] != swiftlintArgs {
		t.Errorf("swiftlint calls = %q, want one %q", calls, swiftlintArgs)
	}
}

// swiftlintSees asserts out shows want and exit code code. The table wraps
// long absolute paths mid-word, so spaces are ignored when comparing.
func swiftlintSees(t *testing.T, out string, gotCode, code int, want string) {
	t.Helper()
	sees(t, out, gotCode, code)
	if !strings.Contains(swiftlintSquash(out), swiftlintSquash(want)) {
		t.Errorf("output is missing %q:\n%s", want, out)
	}
}

// swiftlintSquash drops table borders and all whitespace from text.
func swiftlintSquash(text string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(text, "│", " ")), "")
}

func Test0057_1(t *testing.T) {
	t.Parallel()
	repo := swiftlintRepo(t, map[string]string{"Short.swift": "let ab = 1\n"})
	rp, out, code := swiftlintRun(t, repo, swiftlintFindings(repo, "Short.swift", swiftlintShort))
	swiftlintAsked(t, rp)
	swiftlintSees(t, out, code, 1, filepath.Join(repo, "Short.swift")+
		":1:5 - Variable name 'ab' should be between 3 and 40 characters long (identifier_name)")
}

func Test0057_2(t *testing.T) {
	t.Parallel()
	repo := swiftlintRepo(t, map[string]string{"NoNewline.swift": "let value = 1"})
	rp, out, code := swiftlintRun(t, repo, swiftlintFindings(repo, "NoNewline.swift", swiftlintNoNewline))
	swiftlintAsked(t, rp)
	swiftlintSees(t, out, code, 1, filepath.Join(repo, "NoNewline.swift")+
		":1 - Files should have a single trailing newline (trailing_newline)")
}

func Test0057_3(t *testing.T) {
	t.Parallel()
	repo := swiftlintRepo(t, map[string]string{"Clean.swift": "let value = 1\n"})
	rp, out, code := swiftlintRun(t, repo, swiftlintClean)
	swiftlintAsked(t, rp)
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0057_4(t *testing.T) {
	t.Parallel()
	repo := swiftlintRepo(t, map[string]string{
		".swiftlint.yml": "disabled_rules:\n  - identifier_name\n",
		"Short.swift":    "let ab = 1\n",
	})
	rp, out, code := swiftlintRun(t, repo, swiftlintClean)
	swiftlintAsked(t, rp)
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0057_5(t *testing.T) {
	t.Parallel()
	rp, out, code := swiftlintRun(t, swiftlintRepo(t, nil))
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "swiftlint")
	if calls := rp.calls("swiftlint"); len(calls) != 0 {
		t.Errorf("swiftlint ran with no Swift files: %q", calls)
	}
}

func Test0057_6(t *testing.T) {
	t.Parallel()
	repo := swiftlintRepo(t, map[string]string{"Clean.swift": "let value = 1\n"})
	out, code := fitness(t, repo, []string{pathWithout(t, "swiftlint")}, "swiftlint")
	sees(t, out, code, 1, "SwiftLint not installed: brew install swiftlint")
}

func Test0057_8(t *testing.T) {
	t.Parallel()
	repo := swiftlintRepo(t, map[string]string{
		".swiftlint.yml": "excluded:\n  - Short.swift\n",
		"Short.swift":    "let ab = 1\n",
	})
	rp, out, code := swiftlintRun(t, repo, swiftlintNothingLeft)
	swiftlintAsked(t, rp)
	sees(t, out, code, 0, "All 1 checks passed")
}
