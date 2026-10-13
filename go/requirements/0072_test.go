package requirements

import (
	"strings"
	"testing"
)

const (
	ajCheck   = "architecture-journey"
	ajView    = acRoot + acJourney
	ajDoc     = "docs/requirements/0001-open.md"
	ajDoc2    = "docs/requirements/0002-save.md"
	ajOpen    = "open(2 Open the app)"
	ajDone    = "done[3 Record saved]"
	ajTagOpen = "open(2 Open the app<br/>Req 0001)"
	ajTagDone = "done[3 Record saved<br/>Req 0001]"
	ajMissing = "architecture/05-user-journey.md:15: requirement %s is missing from the journey diagram"
	ajTable   = "\n## Requirements\n\n| Requirement | Journey |\n| --- | --- |\n| [0001](../docs/requirements/0001-open.md) | `open`, `done` |\n"
)

// ajFixture is the contract fixture whose journey steps both name
// requirement 0001 in their labels. The diagram fence is line 15, the open
// step line 18, and the done step line 19.
func ajFixture() map[string]string {
	return ajEdit(ajEdit(acFixture(acRoot), ajOpen, ajTagOpen), ajDone, ajTagDone)
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

func ajMissingDoc(id string) string {
	return strings.Replace(ajMissing, "%s", id, 1)
}

func Test0072_1(t *testing.T) {
	t.Parallel()
	out, code := ajRun(t, map[string]string{"src/main.go": "package main\n"})
	sees(t, out, code, 0)
	passedWithNoFiles(t, out, ajCheck)
	out, code = ajRun(t, ajFixture())
	sees(t, out, code, 0)
	passedWith(t, out, ajCheck, "2")
	files := ajEdit(ajSecondDoc(ajFixture()), ajTagOpen, "open(2 Open the app<br/>Req 0001, 0002)")
	out, code = ajRun(t, ajEdit(files, "-->|saves| done", "-->|saves 0002| done"))
	sees(t, out, code, 0)
	passedWith(t, out, ajCheck, "3")
	out, code = ajRun(t, ajEdit(ajFixture(), ajTagOpen, "open(2 Open #9829; the app<br/>Req 0001)"))
	sees(t, out, code, 0)
}

func Test0072_2(t *testing.T) {
	t.Parallel()
	out, code := ajRun(t, acFixture(acRoot))
	sees(t, out, code, 1, ajMissingDoc("0001"))
	out, code = ajRun(t, ajSecondDoc(ajFixture()))
	sees(t, out, code, 1, ajMissingDoc("0002"))
	out, code = ajRun(t, ajEdit(ajSecondDoc(ajFixture()), ajTagDone, "done[3 Record saved<br/>Req 0002]"))
	sees(t, out, code, 0)
}

func Test0072_3(t *testing.T) {
	t.Parallel()
	for label, want := range map[string]string{
		"open(2 Open the app<br/>Req 0001.1)":     "architecture/05-user-journey.md:18: 0001.1 is an acceptance ID; name requirement doc 0001",
		"open(2 Open the app<br/>Req 0001, 0009)": "architecture/05-user-journey.md:18: no requirement doc 0009 in docs/requirements",
	} {
		out, code := ajRun(t, ajEdit(ajFixture(), ajTagOpen, label))
		sees(t, out, code, 1, want)
	}
	out, code := ajRun(t, ajEdit(ajFixture(), ajTagOpen, "open(2 Open the app<br/>Req 0009)"))
	sees(t, out, code, 1, "architecture/05-user-journey.md:18: no requirement doc 0009 in docs/requirements")
	if strings.Contains(out, "journey node open names no requirement") {
		t.Errorf("a mistyped ID also reported its step as naming none:\n%s", out)
	}
	repo := example(t, "happyRepo", ajEdit(ajSecondDoc(ajFixture()), ajTagDone, "done[3 Record saved<br/>Req 0001, 0002]"))
	write(t, repo, map[string]string{ajDoc2: ""})
	out, code = fitness(t, repo, nil, ajCheck)
	sees(t, out, code, 1, "architecture/05-user-journey.md:19: no requirement doc 0002 in docs/requirements")
}

func Test0072_4(t *testing.T) {
	t.Parallel()
	nine, eight := ajFixture(), ajFixture()
	for _, id := range []string{"0002", "0003", "0004", "0005", "0006", "0007", "0008", "0009"} {
		nine[strings.Replace(ajDoc2, "0002", id, 1)] = strings.ReplaceAll(nine[ajDoc], "0001", id)
		if id != "0009" {
			eight[strings.Replace(ajDoc2, "0002", id, 1)] = strings.ReplaceAll(eight[ajDoc], "0001", id)
		}
	}
	ids := "0001, 0002, 0003, 0004, 0005, 0006, 0007, 0008"
	out, code := ajRun(t, ajEdit(nine, ajTagOpen, "open(2 Open the app<br/>Req "+ids+", 0009)"))
	sees(t, out, code, 1, "architecture/05-user-journey.md:18: label names 9 requirements; split the step so each names at most 8")
	out, code = ajRun(t, ajEdit(eight, ajTagOpen, "open(2 Open the app<br/>Req "+ids+")"))
	sees(t, out, code, 0)
}

func Test0072_5(t *testing.T) {
	t.Parallel()
	files := acFixture(acRoot)
	files[ajView] += ajTable + "\nThe open step fulfils 0001.\n"
	out, code := ajRun(t, files)
	sees(t, out, code, 1, ajMissingDoc("0001"), "journey node open names no requirement", "journey node done names no requirement")
}

func Test0072_6(t *testing.T) {
	t.Parallel()
	out, code := ajRun(t, ajEdit(ajFixture(), ajTagDone, ajDone))
	sees(t, out, code, 1, "architecture/05-user-journey.md:19: journey node done names no requirement")
	out, code = ajRun(t, ajEdit(ajFixture(), ajTagOpen, ajOpen))
	sees(t, out, code, 1, "architecture/05-user-journey.md:18: journey node open names no requirement")
	out, code = ajRun(t, ajEdit(ajFixture(), ajTagDone, "subgraph steps[Steps]\n"+ajDone+"\nend"))
	sees(t, out, code, 1, "journey node done names no requirement")
	out, code = ajRun(t, ajEdit(ajFixture(), ajTagDone, "subgraph steps[Steps<br/>Req 0001]\n"+ajDone+"\nend"))
	sees(t, out, code, 0)
}

func Test0072_7(t *testing.T) {
	t.Parallel()
	out, code := ajRun(t, ajEdit(ajFixture(), ajTagDone, "done[3 Record stored<br/>Req 0001]"))
	sees(t, out, code, 0)
	renamed := ajEdit(ajEdit(ajFixture(), ajTagDone, "stored[3 Record saved<br/>Req 0001]"), "|saves| done", "|saves| stored")
	out, code = ajRun(t, renamed)
	sees(t, out, code, 0)
	dropped := ajEdit(ajEdit(ajFixture(), ajTagDone, "done[3 Record saved<br/>Req 0002]"), ajTagOpen, "open(2 Open the app<br/>Req 0002)")
	out, code = ajRun(t, ajSecondDoc(dropped))
	sees(t, out, code, 1, ajMissingDoc("0001"))
}

func Test0072_8(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", ajSecondDoc(ajFixture()))
	for _, changed := range []string{"README.md", ajDoc, ajView} {
		out, code := fitness(t, repo, []string{"FITNESS_CHANGED_FILES=" + changed}, ajCheck)
		sees(t, out, code, 1, ajMissingDoc("0002"))
	}
}
