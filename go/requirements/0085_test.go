// cspell:ignore antml

package requirements

import (
	"strings"
	"testing"
)

// commentStructure runs comment-structure on comment, saved over happyRepo
// with files, as --body-file.
func commentStructure(t *testing.T, comment string, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", with(files, map[string]string{"comment.md": comment})), nil,
		"comment-structure", "--body-file", "comment.md")
}

// words is n words of plain prose.
func words(n int) string {
	return strings.TrimSpace(strings.Repeat("word ", n)) + "."
}

// numbered is a numbered diagram and its matching callout table.
const numbered = "```mermaid\nflowchart TD\n  A[1 Start]:::s --> B[2 End]:::s\n  classDef s fill:#eef\n```\n\n" +
	"| # | Element | Description | Why |\n| - | - | - | - |\n| 1 | Start | First. | Begins. |\n| 2 | End | Last. | Ends. |\n"

// overBudget has 120 prose words, and more inside a table and code.
var overBudget = words(120) + "\n\n| A | B |\n| - | - |\n| " + words(30) + " | x |\n\n```\n" + words(30) + "\n```\n"

func Test0085_1(t *testing.T) {
	t.Parallel()
	out, code := commentStructure(t, overBudget, nil)
	sees(t, out, code, 1, "(comment): 120 words outside tables, code, and diagrams; the budget is 100")
}

func Test0085_2(t *testing.T) {
	t.Parallel()
	out, code := commentStructure(t, overBudget, map[string]string{".fitnessrc.json": `{"comment": {"maxWords": 150}}`})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0085_3(t *testing.T) {
	t.Parallel()
	table := "| A | B |\n| - | - |\n| 1 | 2 |\n"
	out, code := commentStructure(t, table+"\nAnd more.\n\n"+table+"\n"+numbered, nil)
	sees(t, out, code, 1, "(comment): 2 tables besides callout tables; the budget is 1")
}

func Test0085_4(t *testing.T) {
	t.Parallel()
	out, code := commentStructure(t, "Flow.\n\n```mermaid\nflowchart TD\n  A --> B\n```\n", nil)
	sees(t, out, code, 1, "(comment): line 3: mermaid diagram has no numbered callouts")
	legend := strings.Replace(numbered, "\n\n|", "\n\n```mermaid\nflowchart LR\n  k[Step]:::s\n  classDef s fill:#eef\n```\n\n|", 1)
	out, code = commentStructure(t, legend, nil)
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0085_5(t *testing.T) {
	t.Parallel()
	out, code := commentStructure(t, "The parser works.\n\nEdit 2: it handles tabs too.\n", nil)
	sees(t, out, code, 1, `(comment): line 3: "Edit 2:": change-log note; edit the text in place and drop the note`)
	out, code = commentStructure(t, "The cache is updated after every push.\n", nil)
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0085_6(t *testing.T) {
	t.Parallel()
	out, code := commentStructure(t, "Done.\n\n<system-reminder>\n\nRan on claude-opus-4-1 today.\n\n<!-- fitness:comment-check:31:abc -->\n", nil)
	sees(t, out, code, 1, `(comment): line 3: "<system-reminder": agent frame; post only the finished text`,
		`(comment): line 5: "claude-opus-4-1": model identifier; name the agent, not the model`,
		`(comment): line 7: "<!-- fitness:": carries a fitness marker; only fitness writes those, so remove it`)
	out, code = commentStructure(t, "Pinned to `claude-opus-4-1` in config.\n", nil)
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0085_7(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	for _, body := range []string{planBody("- [ ] Write the tests."), epicBody("1. #22\n2. #23, after #22")} {
		out, code := piped(t, repo, body, nil, "--blob", "-")
		sees(t, out, code, 0, "Skipped (does not apply to this text): comment-structure")
	}
}

func Test0085_8(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "comment-structure")
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "comment-structure")
}
