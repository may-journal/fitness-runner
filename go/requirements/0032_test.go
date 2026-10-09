package requirements

import (
	"fmt"
	"strings"
	"testing"
)

// jscpdBlock builds lines of distinct words, so a shared prefix yields an
// exact copy and different prefixes never match. Six lines of ten words
// plus ";" is 66 tokens, past the 50-token minimum.
func jscpdBlock(prefix string, lines, perLine int) string {
	var b strings.Builder
	for i := 0; i < lines; i++ {
		for j := 0; j < perLine; j++ {
			fmt.Fprintf(&b, "%s_%d_%d ", prefix, i, j)
		}
		b.WriteString(";\n")
	}
	return b.String()
}

// jscpdClone is a copied block long enough to count as a clone.
var jscpdClone = jscpdBlock("c", 6, 10)

// jscpdPair puts block at the top of two files that differ after it.
func jscpdPair(a, b, block string) map[string]string {
	return map[string]string{
		a: block + jscpdBlock("fa", 24, 3),
		b: block + jscpdBlock("fb", 24, 3),
	}
}

// jscpdRun runs `fitness-install -- jscpd` on happyRepo with files over it.
func jscpdRun(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, "jscpd")
}

func Test0032_1(t *testing.T) {
	t.Parallel()
	out, code := jscpdRun(t, jscpdPair("src/a.js", "src/b.js", jscpdClone))
	sees(t, out, code, 1, "ERROR: jscpd found too many duplicates (7.4%) over threshold (1.0%)")
}

func Test0032_2(t *testing.T) {
	t.Parallel()
	out, code := jscpdRun(t, jscpdPair("src/a.js", "src/b.js", jscpdBlock("w", 3, 20)))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0032_3(t *testing.T) {
	t.Parallel()
	out, code := jscpdRun(t, jscpdPair("src/a.js", "src/b.js", jscpdBlock("s", 6, 3)))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0032_4(t *testing.T) {
	t.Parallel()
	out, code := jscpdRun(t, jscpdPair("docs/a.md", "conf/b.json", jscpdClone))
	sees(t, out, code, 1, "ERROR: jscpd found too many duplicates (")
}

func Test0032_5(t *testing.T) {
	t.Parallel()
	files := jscpdPair("deps/a.lock", "deps/b.lock", jscpdClone)
	files[".gitattributes"] = "*.lock linguist-generated\n"
	out, code := jscpdRun(t, files)
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0032_6(t *testing.T) {
	t.Parallel()
	out, code := jscpdRun(t, jscpdPair("blob/a.dat", "blob/b.dat", "\x00\n"+jscpdClone))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0032_7(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	write(t, repo, jscpdPair("src/a.js", "src/b.js", jscpdClone))
	out, code := fitness(t, repo, nil, "jscpd")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0032_8(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{".gitignore": "vendor/\n"})
	write(t, repo, jscpdPair("vendor/a.js", "src/b.js", jscpdClone))
	git(t, repo, "add", "-f", "vendor", "src")
	out, code := fitness(t, repo, nil, "jscpd")
	sees(t, out, code, 1, "ERROR: jscpd found too many duplicates (")
}
