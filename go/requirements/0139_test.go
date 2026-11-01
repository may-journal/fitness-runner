package requirements

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// commentTreatments are the values a check may declare for code comments.
var commentTreatments = []string{"judged", "stripped", "not applicable"}

// treatedCheck is one check as `fitness list --json` gives it, with its
// comment treatment.
type treatedCheck struct{ Name, Summary, Scope, Comments string }

// listTreatments runs `fitness list --json` in repo and decodes every check
// it lists after the installer's own lines.
func listTreatments(t *testing.T, repo string) []treatedCheck {
	t.Helper()
	out, code := fitness(t, repo, nil, "list", "--json")
	start := strings.Index(out, "\n[")
	if code != 0 || start < 0 {
		t.Fatalf("fitness list --json (exit %d) printed no check list:\n%s", code, out)
	}
	var checks []treatedCheck
	mustDo(t, json.Unmarshal([]byte(out[start+1:]), &checks))
	return checks
}

// commentedOut puts marker before every line of code, so the whole of it
// becomes comments.
func commentedOut(marker, code string) string {
	lines := strings.SplitAfter(code, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = marker + line
		}
	}
	return strings.Join(lines, "")
}

// typoComments puts a misspelled comment on line 3 of a Go, Swift,
// TypeScript and shell file, each after head.
func typoComments(head string) map[string]string {
	line := "Retries the " + cspellTypo + " call once.\n"
	return map[string]string{
		"app/main.go":    "package main\n" + head + "// " + line + "func main() {}\n",
		"app/App.swift":  "import Foundation\n" + head + "// " + line + "func run() {}\n",
		"app/index.ts":   "export {};\n" + head + "// " + line + "export const x = 1;\n",
		"scripts/run.sh": "#!/bin/sh\n" + strings.ReplaceAll(head, "//", "#") + "# " + line + "echo ok\n",
	}
}

func Test0139_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	for _, check := range listTreatments(t, repo) {
		if !slices.Contains(commentTreatments, check.Comments) {
			t.Errorf("fitness list --json gives %s the comments %q, want one of %q", check.Name, check.Comments, commentTreatments)
		}
		out, code := fitness(t, repo, nil, "explain", check.Name)
		sees(t, out, code, 0, "Scope: "+check.Scope+" · Comments: "+check.Comments+" ·")
	}
}

func Test0139_2(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	index := repoFile(t, "docs/checks.md")
	for _, check := range listTreatments(t, repo) {
		row := "| " + strings.ReplaceAll(check.Summary, "|", `\|`) + " | " + check.Comments + " |"
		if !strings.Contains(index, "/README.md) "+row) {
			t.Errorf("docs/checks.md has no %s row ending %q", check.Name, row)
		}
	}
}

func Test0139_3(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, dir := range []string{"go/cmd", "go/internal"} {
		mustDo(t, os.CopyFS(filepath.Join(root, dir), os.DirFS(filepath.Join("..", "..", dir))))
	}
	write(t, root, map[string]string{"README.md": repoFile(t, "README.md"), "docs/checks.md": repoFile(t, "docs/checks.md")})
	out, code := releaseToolRun(t, root, nil, "checks")
	sees(t, out, code, 0)
	source := "go/cmd/fitness-check-node-version/main.go"
	declared := repoFile(t, source)
	omitted := strings.Replace(declared, ", Comments: checkkit.CommentsNotApplicable", "", 1)
	if omitted == declared {
		t.Fatalf("%s declares no comments field to remove:\n%s", source, declared)
	}
	write(t, root, map[string]string{source: omitted})
	out, code = releaseToolRun(t, root, nil, "checks")
	sees(t, out, code, 1, "go/cmd/fitness-check-node-version: every checkkit.Describe must set Comments to one of checkkit.CommentsJudged, checkkit.CommentsNotApplicable, checkkit.CommentsStripped")
}

func Test0139_4(t *testing.T) {
	t.Parallel()
	out, code := cspellRun(t, typoComments("\n"))
	unknown := " - Unknown word (" + cspellTypo + ")"
	sees(t, out, code, 1, "app/App.swift:3:16"+unknown, "app/index.ts:3:16"+unknown, "app/main.go:3:16"+unknown, "scripts/run.sh:3:15"+unknown)
}

func Test0139_5(t *testing.T) {
	t.Parallel()
	out, code := cspellRun(t, typoComments("// cspell:ignore "+cspellTypo+"\n"))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0139_6(t *testing.T) {
	t.Parallel()
	out, code := repeatedStringLiterals(t, map[string]string{"src/a.ts": commentedOut("// ", rslLines("active", 2)) + "/* " + rslLines("active", 1) + "*/\n"})
	sees(t, out, code, 0, "All 1 checks passed")
	out, code = repeatedStringLiterals(t, map[string]string{"src/a.ts": rslLines("active", 3)})
	sees(t, out, code, 1, `"active" appears 3 times`)
}

func Test0139_7(t *testing.T) {
	t.Parallel()
	function := "func f() {\n" + goComplexityIfs(6) + "}\n"
	out, code := goComplexity(t, map[string]string{"a.go": "package p\n\nvar ok bool\n\n" + commentedOut("// ", function)})
	sees(t, out, code, 0, "All 1 checks passed")
	out, code = goComplexity(t, map[string]string{"a.go": "package p\n\nvar ok bool\n\n" + function})
	sees(t, out, code, 1, "a.go:5: func f has a complexity of 7; maximum allowed is 5")
}

func Test0139_8(t *testing.T) {
	t.Parallel()
	out, code := dupRun(t, dupPair("src/a.js", "src/b.js", commentedOut("// ", dupClone)))
	sees(t, out, code, 0, "All 1 checks passed")
	out, code = dupRun(t, dupPair("src/a.js", "src/b.js", dupClone))
	sees(t, out, code, 1, "ERROR: duplicate code is", "src/b.js: 6 duplicated lines")
}
