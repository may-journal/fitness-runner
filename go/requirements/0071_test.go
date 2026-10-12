package requirements

import (
	"strings"
	"testing"
)

const (
	ajCheck   = "architecture-journey"
	ajView    = acRoot + acJourney
	ajTable   = "\n## Requirements\n\n| Requirement | Journey |\n| --- | --- |\n"
	ajRow     = "| [0001](../docs/requirements/0001-open.md) | `open`, `done` |\n"
	ajDoc     = "docs/requirements/0001-open.md"
	ajDoc2    = "docs/requirements/0002-save.md"
	ajNoRow   = "architecture/05-user-journey.md:33: requirement 0002 has no row in the journey's requirement table"
	ajUnknown = "architecture/05-user-journey.md:35: requirement 0001 maps unknown journey node "
	ajDone    = "done[3 Record saved]"
)

// ajFixture is the contract fixture whose journey view ends with a
// requirement table of the given rows. The table header is line 33 and its
// first row line 35.
func ajFixture(rows ...string) map[string]string {
	files := acFixture(acRoot)
	files[ajView] += ajTable + strings.Join(rows, "")
	return files
}

// ajSecondDoc adds requirement 0002, shaped like 0001.
func ajSecondDoc(files map[string]string) map[string]string {
	files[ajDoc2] = strings.ReplaceAll(files[ajDoc], "0001", "0002")
	return files
}

func ajRun(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, ajCheck)
}

// ajEdit replaces old with new in the journey view of a fixture.
func ajEdit(files map[string]string, old, new string) map[string]string {
	files[ajView] = strings.Replace(files[ajView], old, new, 1)
	return files
}

func Test0071_1(t *testing.T) {
	t.Parallel()
	out, code := ajRun(t, map[string]string{"src/main.go": "package main\n"})
	sees(t, out, code, 0)
	passedWithNoFiles(t, out, ajCheck)
	out, code = ajRun(t, ajFixture(ajRow))
	sees(t, out, code, 0)
	passedWith(t, out, ajCheck, "2")
	out, code = ajRun(t, ajSecondDoc(ajFixture(ajRow, "| 0002 | `open` |\n")))
	sees(t, out, code, 0)
	passedWith(t, out, ajCheck, "3")
}

func Test0071_2(t *testing.T) {
	t.Parallel()
	out, code := ajRun(t, acFixture(acRoot))
	sees(t, out, code, 1, "architecture/05-user-journey.md:1: requirement 0001 has no row in the journey's requirement table")
	out, code = ajRun(t, ajSecondDoc(ajFixture(ajRow)))
	sees(t, out, code, 1, ajNoRow)
	out, code = ajRun(t, ajSecondDoc(ajFixture(ajRow, "| [0002](../docs/requirements/0002-save.md) | `done` |\n")))
	sees(t, out, code, 0)
}

func Test0071_3(t *testing.T) {
	t.Parallel()
	for row, want := range map[string]string{
		"| `0001.1` | `open`, `done` |\n":                          "architecture/05-user-journey.md:35: 0001.1 is an acceptance ID; map requirement doc 0001",
		"| [0009](../docs/requirements/0009-gone.md) | `open` |\n": "architecture/05-user-journey.md:35: no requirement doc 0009 in docs/requirements",
		"| open | `open` |\n":                                      "architecture/05-user-journey.md:35: \"open\" is not a requirement doc ID such as 0001",
	} {
		out, code := ajRun(t, ajFixture(row))
		sees(t, out, code, 1, want)
	}
	repo := example(t, "happyRepo", ajSecondDoc(ajFixture(ajRow, "| 0002 | `open` |\n")))
	write(t, repo, map[string]string{ajDoc2: ""})
	out, code := fitness(t, repo, nil, ajCheck)
	sees(t, out, code, 1, "architecture/05-user-journey.md:36: no requirement doc 0002 in docs/requirements")
}

func Test0071_4(t *testing.T) {
	t.Parallel()
	out, code := ajRun(t, ajFixture(ajRow, "| 0001 | `open` |\n"))
	sees(t, out, code, 1, "architecture/05-user-journey.md:36: requirement 0001 is mapped again; first at architecture/05-user-journey.md:35")
	out, code = ajRun(t, ajFixture(ajRow))
	sees(t, out, code, 0)
}

func Test0071_5(t *testing.T) {
	t.Parallel()
	out, code := ajRun(t, ajFixture("| 0001 | |\n"))
	sees(t, out, code, 1, "architecture/05-user-journey.md:35: requirement 0001 maps no journey node")
	out, code = ajRun(t, ajFixture("| 0001 | `open`, `ghost` |\n"))
	sees(t, out, code, 1, ajUnknown+"ghost")
	out, code = ajRun(t, ajFixture("| 0001 | `open`, `done` |\n"))
	sees(t, out, code, 0)
}

func Test0071_6(t *testing.T) {
	t.Parallel()
	out, code := ajRun(t, ajFixture("| 0001 | `open` |\n"))
	sees(t, out, code, 1, "architecture/05-user-journey.md:19: journey node done maps to no requirement")
	out, code = ajRun(t, ajFixture("| 0001 | `done` |\n"))
	sees(t, out, code, 1, "architecture/05-user-journey.md:18: journey node open maps to no requirement")
	files := ajEdit(ajFixture(ajRow), ajDone, "subgraph steps[Steps]\n"+ajDone+"\nend")
	out, code = ajRun(t, files)
	sees(t, out, code, 0)
}

func Test0071_7(t *testing.T) {
	t.Parallel()
	out, code := ajRun(t, ajEdit(ajFixture(ajRow), ajDone, "done[3 Record stored]"))
	sees(t, out, code, 0)
	renamed := ajEdit(ajEdit(ajFixture(ajRow), ajDone, "stored[3 Record saved]"), "|saves| done", "|saves| stored")
	out, code = ajRun(t, renamed)
	sees(t, out, code, 1, ajUnknown+"done", "architecture/05-user-journey.md:19: journey node stored maps to no requirement")
	out, code = ajRun(t, ajEdit(renamed, "`done`", "`stored`"))
	sees(t, out, code, 0)
}

func Test0071_8(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", ajSecondDoc(ajFixture(ajRow)))
	for _, changed := range []string{"README.md", ajDoc, ajView} {
		out, code := fitness(t, repo, []string{"FITNESS_CHANGED_FILES=" + changed}, ajCheck)
		sees(t, out, code, 1, ajNoRow)
	}
}
