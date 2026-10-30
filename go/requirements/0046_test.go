package requirements

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// noPlansDir runs `fitness-install -- no-plans-dir` on happyRepo with files
// written over it.
func noPlansDir(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, "no-plans-dir")
}

// Where the check sends each kind of temporary work.
const (
	toPlanIssue  = "open a `Plan`-labeled GitHub Issue and put the plan there, not a file in the repo"
	toPlanDesign = "post it as a comment on the Plan Issue that it designs, not a file in the repo"
	toPRComment  = "post it as a comment on the pull request it reviews, not a file in the repo"
)

// inPlansFolder is the error line for a file in a `plans` folder.
func inPlansFolder(path string) string {
	return path + ": a `plans` folder holds temporary work; " + toPlanIssue
}

// planChecklist is a Plan's checklist as a Markdown file would hold it.
const planChecklist = "# Ship it\n\n## What needs to happen\n\n- [ ] Write the tests.\n"

func Test0046_1(t *testing.T) {
	t.Parallel()
	out, code := noPlansDir(t, map[string]string{"docs/plans/03-publish.md": "# Publish\n"})
	sees(t, out, code, 1, inPlansFolder("docs/plans/03-publish.md"))
}

func Test0046_2(t *testing.T) {
	t.Parallel()
	out, code := noPlansDir(t, map[string]string{"docs/plans/archive/old.md": "# Old\n"})
	sees(t, out, code, 1, inPlansFolder("docs/plans/archive/old.md"))
}

func Test0046_3(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"docs/notes.md":         "# Notes\n",
		"assets/screen.png":     "png",
		"docs/planner-guide.md": "# Planner guide\n\nA Plan lists its steps under a heading.\n",
		"go/plan_test.go":       "package plan\n\nconst body = \"## What needs to happen\\n\\n- [ ] Ship it.\\n\"\n",
	}
	repo := example(t, "happyRepo", files)
	out, code := fitness(t, repo, nil, "no-plans-dir")
	sees(t, out, code, 0, "All 1 checks passed")
	for name := range files {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(name))); err != nil {
			t.Errorf("no-plans-dir must leave %s in place: %v", name, err)
		}
	}
}

func Test0046_4(t *testing.T) {
	t.Parallel()
	out, code := noPlansDir(t, map[string]string{
		"src/Plans/next.md":               "# Next\n",
		"design/mockups/home.png":         "png",
		"reports/validation/run.txt":      "ok",
		"review-artifacts/diff.txt":       "diff",
		"assets/icons/validation-ok.png":  "png",
		"docs/validation-checklist.md":    "# Validation\n",
		"tests/fixtures/review/input.txt": "in",
	})
	sees(t, out, code, 1,
		"design/mockups/home.png: a `mockups` folder holds temporary work; "+toPlanDesign,
		"reports/validation/run.txt: a `validation` folder holds temporary work; "+toPRComment,
		"review-artifacts/diff.txt: a `review-artifacts` folder holds temporary work; "+toPRComment,
		"src/Plans/next.md: a `plans` folder holds temporary work; "+toPlanIssue,
	)
	inOrder(t, out, "design/mockups/home.png:", "reports/validation/run.txt:", "review-artifacts/diff.txt:", "src/Plans/next.md:")
	for _, kept := range []string{"assets/icons/validation-ok.png", "docs/validation-checklist.md", "tests/fixtures/review/input.txt"} {
		if strings.Contains(out, kept+":") {
			t.Errorf("%s names no artifact folder, yet it was reported:\n%s", kept, out)
		}
	}
}

func Test0046_5(t *testing.T) {
	t.Parallel()
	out, code := noPlansDir(t, map[string]string{
		"docs/launch-plan.md":  "# Launch\n",
		"img/home-mockup.png":  "png",
		"pr-42-before.png":     "png",
		"docs/planner.md":      "# Planner\n",
		"img/print-before.png": "png",
	})
	sees(t, out, code, 1,
		"docs/launch-plan.md: a `*-plan.md` file is a plan; "+toPlanIssue,
		"img/home-mockup.png: a `*-mockup.*` file is a design mockup; "+toPlanDesign,
		"pr-42-before.png: a `pr-<number>-*` file belongs to one pull request; "+toPRComment,
	)
	for _, kept := range []string{"docs/planner.md", "img/print-before.png"} {
		if strings.Contains(out, kept+":") {
			t.Errorf("%s matches no artifact name, yet it was reported:\n%s", kept, out)
		}
	}
}

func Test0046_6(t *testing.T) {
	t.Parallel()
	out, code := noPlansDir(t, map[string]string{
		"docs/ship.md":                   planChecklist,
		".github/ISSUE_TEMPLATE/plan.md": planChecklist,
		"docs/what-needs-to-happen.md":   "# Notes\n\n## What needs to happen\n\nNothing yet.\n",
	})
	sees(t, out, code, 1, "docs/ship.md: a `## What needs to happen` checklist is a plan; "+toPlanIssue)
	for _, kept := range []string{".github/ISSUE_TEMPLATE/plan.md", "docs/what-needs-to-happen.md"} {
		if strings.Contains(out, kept+":") {
			t.Errorf("%s holds no Plan checklist outside .github/, yet it was reported:\n%s", kept, out)
		}
	}
}

func Test0046_7(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{"docs/home.png": "png"})
	write(t, repo, map[string]string{"docs/plans/next.md": "# Next\n"})
	git(t, repo, "add", "docs/plans/next.md")
	mustDo(t, os.MkdirAll(filepath.Join(repo, "docs", "mockups"), 0o755))
	git(t, repo, "mv", "docs/home.png", "docs/mockups/home.png")
	out, code := fitness(t, repo, nil, "no-plans-dir")
	sees(t, out, code, 1,
		"docs/mockups/home.png: a `mockups` folder holds temporary work; "+toPlanDesign,
		inPlansFolder("docs/plans/next.md"),
	)
}

func Test0046_8(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	mustDo(t, os.WriteFile(filepath.Join(repo, ".git", "index"), []byte("not an index"), 0o644))
	out, code := fitness(t, repo, nil, "no-plans-dir")
	sees(t, out, code, 1, "cannot list the repo's files")
}

// inOrder asserts each want appears in out after the one before it.
func inOrder(t *testing.T, out string, want ...string) {
	t.Helper()
	text, at := read(out), 0
	for _, w := range want {
		i := strings.Index(text[at:], w)
		if i < 0 {
			t.Errorf("output is missing %q after position %d, or lists it out of order:\n%s", w, at, out)
			return
		}
		at += i + len(w)
	}
}
