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

// Test0032_12 puts a block of 3 lines in one pair and a block of 18 tokens
// in another, each under one of the two minimums.
func Test0032_12(t *testing.T) {
	t.Parallel()
	files := dupPair("src/a.js", "src/b.js", dupBlock("w", 3, 20))
	files["src/c.js"] = dupBlock("s", 6, 3) + dupBlock("fc", 24, 3)
	files["src/d.js"] = dupBlock("s", 6, 3) + dupBlock("fd", 24, 3)
	out, code := dupRun(t, files)
	sees(t, out, code, 0, "All 1 checks passed")
}

// Test0032_13 tracks a doc holding a copied block, then commits the block
// in a script, which duplicate-code never compares with a doc, or in a second doc.
// Each commit reads only the files of its staged format: the script, or
// the README, the changelog and both docs.
func Test0032_13(t *testing.T) {
	t.Parallel()
	passedWith(t, dupCommit(t, "src/copy.js"), "duplicate-code", "1")
	out := dupCommit(t, "docs/copy.md")
	if !regexp.MustCompile(`│ failed\s+│ 4\s+│`).MatchString(row(out, "duplicate-code")) {
		t.Errorf("a staged doc copying a tracked doc must fail duplicate-code over the 4 docs, got row %q\n%s", row(out, "duplicate-code"), out)
	}
}

// dupCommit tracks a doc holding dupClone in a hooked repo, stages file
// holding it too, commits as a user does, and returns what the commit
// printed.
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

// Test0032_10 adds a template to Test0032_1's two copies. The share stays
// 7.4%, so only the two docs count: the template is not scanned.
func Test0032_10(t *testing.T) {
	t.Parallel()
	files := dupPair("docs/guide.md", "docs/notes.md", dupClone)
	files["docs/templates/guide.md"] = dupClone + dupBlock("ft", 24, 3)
	out, code := dupRun(t, files)
	sees(t, out, code, 1, "ERROR: duplicate code is 7.4% of scanned lines, over the 1.0% threshold")
}

// Test0032_15 wraps the copied block in each marker spelling: the current
// duplicate-code markers in one pair and the deprecated jscpd ones in the
// other, so a repo written before the rename keeps passing.
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
