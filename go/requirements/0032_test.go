// cspell:ignore mvnw gradlew

package requirements

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// dupBlock builds lines of distinct words, so a shared prefix yields an
// exact copy and different prefixes never match. Six lines of ten words
// plus ";" is 66 tokens, past the 50-token minimum.
func dupBlock(prefix string, lines, perLine int) string {
	var b strings.Builder
	for i := 0; i < lines; i++ {
		for j := 0; j < perLine; j++ {
			fmt.Fprintf(&b, "%s_%d_%d ", prefix, i, j)
		}
		b.WriteString(";\n")
	}
	return b.String()
}

// dupClone is a copied block long enough to count as a clone.
var dupClone = dupBlock("c", 6, 10)

// dupPair puts block at the top of two files that differ after it.
func dupPair(a, b, block string) map[string]string {
	return map[string]string{
		a: block + dupBlock("fa", 24, 3),
		b: block + dupBlock("fb", 24, 3),
	}
}

// dupRun runs `fitness-install -- duplicate-code` on happyRepo with files over it.
func dupRun(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, "duplicate-code")
}

func Test0032_1(t *testing.T) {
	t.Parallel()
	out, code := dupRun(t, dupPair("src/a.js", "src/b.js", dupClone))
	sees(t, out, code, 1, "ERROR: duplicate code is 7.4% of scanned lines, over the 1.0% threshold", "src/b.js: 6 duplicated lines")
	if strings.Contains(read(out), "src/a.js:") {
		t.Errorf("the first copy holds no duplicated line, so only src/b.js is named:\n%s", out)
	}
}

// Test0032_17 puts a 3-line block in one pair and an 18-token block in
// another, each under one minimum. Two docs share a 5-line run only because
// it ends in a 120-dash separator row.
func Test0032_17(t *testing.T) {
	t.Parallel()
	files := dupPair("src/a.js", "src/b.js", dupBlock("w", 3, 20))
	files["src/c.js"] = dupBlock("s", 6, 3) + dupBlock("fc", 24, 3)
	files["src/d.js"] = dupBlock("s", 6, 3) + dupBlock("fd", 24, 3)
	files["docs/a.md"] = "# A\n\n" + dupSeparatorTable + "a | b\n" + dupBlock("da", 24, 3)
	files["docs/b.md"] = "# B\n\n" + dupSeparatorTable + "c | d\n" + dupBlock("db", 24, 3)
	out, code := dupRun(t, files)
	sees(t, out, code, 0, "All 1 checks passed")
}

// dupSeparatorTable is two lines of prose and a table's header and separator
// rows. Its 120 dashes carry it past 50 tokens and its last line ends a
// 5-line run.
var dupSeparatorTable = "Both docs list names.\nEach row says why.\n\n| Name | Why |\n| " +
	strings.Repeat("-", 60) + " | " + strings.Repeat("-", 60) + " |\n"

// Test0032_13 commits a copied block in a script, which duplicate-code never
// compares with a doc, then in a second doc.
func Test0032_13(t *testing.T) {
	t.Parallel()
	passedWith(t, dupCommit(t, "src/copy.js"), "duplicate-code", "1")
	out := dupCommit(t, "docs/copy.md")
	if !regexp.MustCompile(`│ failed\s+│ 4\s+│`).MatchString(row(out, "duplicate-code")) {
		t.Errorf("a staged doc copying a tracked doc must fail duplicate-code over the 4 docs, got row %q\n%s", row(out, "duplicate-code"), out)
	}
}

// dupCommit tracks a doc holding dupClone in a hooked repo, stages file
// holding it too, and returns what the commit printed.
func dupCommit(t *testing.T, file string) string {
	t.Helper()
	repo, env := hooked(t, map[string]string{"docs/guide.md": dupClone + dupBlock("fg", 24, 3)})
	write(t, repo, map[string]string{file: dupClone + dupBlock("fc", 24, 3)})
	user(t, repo, env, "add", file)
	out, _ := user(t, repo, env, "commit", "-m", "docs(guide): copy the block")
	return out
}

func Test0032_9(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{".gitignore": "conf/\n"})
	write(t, repo, dupPair("docs/a.md", "conf/b.json", dupClone))
	git(t, repo, "add", "-f", "docs", "conf")
	out, code := fitness(t, repo, nil, "duplicate-code")
	sees(t, out, code, 1, "ERROR: duplicate code is ")
}

func Test0032_11(t *testing.T) {
	t.Parallel()
	files := dupPair("deps/a.lock", "deps/b.lock", dupClone)
	files[".gitattributes"] = "*.lock linguist-generated\n"
	for name, body := range dupPair("app/gradle.lockfile", "lib/gradle.lockfile", dupClone) {
		files[name] = body
	}
	for name, body := range dupPair("mvnw", "app/gradlew.bat", dupClone) {
		files[name] = body
	}
	out, code := dupRun(t, files)
	sees(t, out, code, 0, "All 1 checks passed")
}

// Test0032_14 tracks a binary pair and leaves a text pair untracked. Both
// pairs share the block and their tails, so only skipping both passes.
func Test0032_14(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", dupPair("blob/a.dat", "blob/b.dat", "\x00\n"+dupClone))
	write(t, repo, dupPair("src/a.js", "src/b.js", dupClone))
	out, code := fitness(t, repo, nil, "duplicate-code")
	sees(t, out, code, 0, "All 1 checks passed")
}

// Test0032_16 adds a template, matching legends and a copied untitled
// diagram to Test0032_1's two docs. A committed third doc scopes the run to
// Markdown, and the exact share proves the template is not scanned.
func Test0032_16(t *testing.T) {
	t.Parallel()
	files := dupPair("docs/guide.md", "docs/notes.md", dupClone)
	files["docs/templates/guide.md"] = dupClone + dupBlock("ft", 24, 3)
	files["docs/architecture/02-containers.md"] = dupLevelDoc("ca")
	files["docs/architecture/03-components.md"] = dupLevelDoc("cb")
	files["docs/flow-a.md"] = "# Flow\n\n" + dupCopiedDiagram + dupBlock("la", 24, 3)
	files["docs/flow-b.md"] = "# Flow\n\n" + dupLevelDiagram("lb") + dupCopiedDiagram + dupBlock("lb", 24, 3)
	repo, env := hooked(t, files)
	write(t, repo, map[string]string{"docs/new.md": "# New\n\n" + dupBlock("fn", 24, 3)})
	user(t, repo, env, "add", "docs/new.md")
	out, code := user(t, repo, env, "commit", "-m", "docs(new): add a doc")
	if code == 0 {
		t.Errorf("the hook must reject the commit:\n%s", out)
	}
	dupNames(t, out, []string{"ERROR: duplicate code is 5.6% of scanned lines", "docs/notes.md: 6 duplicated lines", "docs/flow-b.md: 11 duplicated lines"},
		[]string{"docs/architecture/02-containers.md:", "docs/architecture/03-components.md:", "docs/flow-a.md:", "docs/guide.md:", "docs/templates/guide.md:"})
}

// dupCopiedDiagram is an untitled diagram with no callouts, then a 48-token
// line. The lexer reads the fenced block as one token, so only a copy whose
// body stays reaches both minimums.
var dupCopiedDiagram = "```mermaid\nflowchart TB\n" + dupBlock("m", 6, 3) + "```\n\n" + dupBlock("t", 1, 47)

// dupNames asserts out holds each of wants and none of the clean paths, which
// hold only lines a doc must repeat.
func dupNames(t *testing.T, out string, wants, clean []string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(read(out), want) {
			t.Errorf("the commit output is missing %q:\n%s", want, out)
		}
	}
	for _, path := range clean {
		if regexp.MustCompile(regexp.QuoteMeta(path) + ` \d+ duplicated lines`).MatchString(read(out)) {
			t.Errorf("%s holds no duplicated line:\n%s", path, out)
		}
	}
}

// dupLevelDoc is one C4 level doc: a numbered diagram, the shared legend
// and table, then prose of its own.
func dupLevelDoc(prefix string) string {
	return "# Level\n\n" + dupLevelDiagram(prefix) + dupLegend + dupTable(prefix) + "\n" + dupBlock(prefix, 24, 3)
}

// dupTable is a level's callout rows, which name the level.
func dupTable(prefix string) string {
	return "| 1 | The " + prefix + " user | Starts " + prefix + " |\n| 2 | The " + prefix + " app | Serves " + prefix + " |\n| A | Opens " + prefix + " | Daily |\n"
}

// dupLevelDiagram is a numbered C4 diagram whose only repeated lines are
// its style statements.
func dupLevelDiagram(prefix string) string {
	return "```mermaid\nC4Container\n    Person(" + prefix + ", \"1 " + prefix + " user\")\n    Container(" + prefix + "2, \"2 " + prefix + " app\", \"Go\", \"Runs " + prefix + "\")\n" +
		"    Rel(" + prefix + ", " + prefix + "2, \"A\")\n" +
		"    UpdateElementStyle(" + prefix + ", $bgColor=\"#334155\", $borderColor=\"#1e293b\", $fontColor=\"#ffffff\")\n" +
		"    UpdateElementStyle(" + prefix + "2, $bgColor=\"#0f766e\", $borderColor=\"#115e59\", $fontColor=\"#ffffff\")\n```\n\n"
}

// dupLegend is the legend diagram every C4 level repeats, with its table.
const dupLegend = "```mermaid\nC4Container\n    title Legend\n\n    Person(p, \"Person\", \"Human or agent actor\")\n    System_Boundary(b, \"System boundary\") {\n        Container(c, \"Container\", \"Tech\", \"Runnable or deployable unit\")\n    }\n    System_Ext(e, \"External\", \"Outside our boundary\")\n\n    UpdateElementStyle(p, $bgColor=\"#334155\", $borderColor=\"#1e293b\", $fontColor=\"#ffffff\")\n    UpdateElementStyle(c, $bgColor=\"#0f766e\", $borderColor=\"#115e59\", $fontColor=\"#ffffff\")\n    UpdateElementStyle(e, $bgColor=\"#64748b\", $borderColor=\"#475569\", $fontColor=\"#ffffff\")\n```\n\nNumbers on nodes and letters on arrows match the callout table.\n\n| # | Description | Why |\n| --- | --- | --- |\n"

// Test0032_15 wraps the block in current markers in one pair and deprecated
// jscpd ones in the other, so older repos keep passing.
func Test0032_15(t *testing.T) {
	t.Parallel()
	current := "// duplicate-code:ignore-start\n" + dupClone + "// duplicate-code:ignore-end\n"
	deprecated := "/* jscpd:ignore-start */\n" + dupBlock("j", 6, 10) + "/* jscpd:ignore-end */\n"
	out, code := dupRun(t, map[string]string{
		"src/a.js": current + dupBlock("fa", 24, 3),
		"src/b.js": current + dupBlock("fb", 24, 3),
		"src/c.js": deprecated + dupBlock("fc", 24, 3),
		"src/d.js": deprecated + dupBlock("fd", 24, 3),
	})
	sees(t, out, code, 0, "All 1 checks passed")
}
