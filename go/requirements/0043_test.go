package requirements

import (
	"strconv"
	"strings"
	"testing"
)

// levelBleed runs `fitness-install -- mermaid-level-bleed` on happyRepo with
// files written over it.
func levelBleed(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, "mermaid-level-bleed")
}

// levelBleedDoc is a C4 level doc with one diagram and a callout table of
// [#, Description] rows.
func levelBleedDoc(descriptions ...string) string {
	lines := []string{"# Level", "", "```mermaid", `Rel(a, b, "1")`, "```", "", "| # | Description |", "| --- | --- |"}
	for i, d := range descriptions {
		lines = append(lines, "| "+strconv.Itoa(i+1)+" | "+d+" |")
	}
	return strings.Join(lines, "\n") + "\n"
}

// levelBleedRepeat is the message for file repeating text from level.
func levelBleedRepeat(file, text string, level int) string {
	return file + `: description "` + text + `" repeats level ` + strconv.Itoa(level) +
		" verbatim — lower levels should add level-specific rationale"
}

func Test0043_1(t *testing.T) {
	t.Parallel()
	out, code := levelBleed(t, map[string]string{
		"docs/architecture/01-context.md":    levelBleedDoc("User reads the timeline"),
		"docs/architecture/02-containers.md": levelBleedDoc("User reads the timeline", "New detail"),
	})
	sees(t, out, code, 1, levelBleedRepeat("docs/architecture/02-containers.md", "user reads the timeline", 1))
}

func Test0043_2(t *testing.T) {
	t.Parallel()
	out, code := levelBleed(t, map[string]string{
		"architecture/01-context.md":    levelBleedDoc("The whole system", ""),
		"architecture/02-containers.md": levelBleedDoc("The macOS app", ""),
	})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0043_3(t *testing.T) {
	t.Parallel()
	out, code := levelBleed(t, map[string]string{
		"architecture/01-context.md": levelBleedDoc("The app"),
		"architecture/02-app.md":     levelBleedDoc("The  APP", "the app", "New detail"),
	})
	want := levelBleedRepeat("architecture/02-app.md", "the app", 1)
	sees(t, out, code, 1, want)
	if n := strings.Count(read(out), read(want)); n != 1 {
		t.Errorf("the repeat must be reported once, got %d:\n%s", n, out)
	}
}

func Test0043_4(t *testing.T) {
	t.Parallel()
	out, code := levelBleed(t, map[string]string{
		"architecture/01-context.md":    levelBleedDoc("Shared line"),
		"architecture/02-containers.md": levelBleedDoc("Something else"),
		"architecture/03-components.md": levelBleedDoc("Shared line"),
	})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0043_5(t *testing.T) {
	t.Parallel()
	out, code := levelBleed(t, map[string]string{
		"architecture/2-app.md":   levelBleedDoc("The engine"),
		"architecture/10-deep.md": levelBleedDoc("The engine"),
	})
	sees(t, out, code, 1, levelBleedRepeat("architecture/10-deep.md", "the engine", 2))
}

func Test0043_6(t *testing.T) {
	t.Parallel()
	out, code := levelBleed(t, map[string]string{
		"architecture/overview.md":    levelBleedDoc("Anything"),
		"not-architecture/01-x.md":    levelBleedDoc("Anything"),
		"not-architecture/02-x.md":    levelBleedDoc("Anything"),
		"docs/01-context-notes.md":    levelBleedDoc("Anything"),
		"docs/02-container-notes.md":  levelBleedDoc("Anything"),
		"architecture/sub/01-deep.md": levelBleedDoc("Anything"),
	})
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "mermaid-level-bleed")
}

func Test0043_7(t *testing.T) {
	t.Parallel()
	out, code := levelBleed(t, map[string]string{
		"architecture/01-context.md": "| # | Label | DESCRIPTION |\n| --- | --- | --- |\n| 1 | Same label | The system |\n",
		"architecture/02-app.md":     "| # | Label | Description |\n| --- | --- | --- |\n| 1 | Same label | The system |\n",
	})
	sees(t, out, code, 1, levelBleedRepeat("architecture/02-app.md", "the system", 1))
	if strings.Contains(out, "same label") {
		t.Errorf("only the Description column may be compared:\n%s", out)
	}
}

func Test0043_8(t *testing.T) {
	t.Parallel()
	out, code := levelBleed(t, map[string]string{
		"architecture/01-context.md": "| # | Text | Notes |\n| --- | --- | --- |\n| 1 | Same | x |\n",
		"architecture/02-app.md":     "| # | Text | Notes |\n| --- | --- | --- |\n| 1 | same | x |\n",
	})
	sees(t, out, code, 1, levelBleedRepeat("architecture/02-app.md", "same", 1))
	if strings.Contains(out, `"x"`) {
		t.Errorf("only the second column may be compared:\n%s", out)
	}
}
