package requirements

import (
	"strings"
	"testing"
)

// noFitnessFront writes files over happyRepo and runs
// `fitness no-fitness-front-matter` there.
func noFitnessFront(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, "no-fitness-front-matter")
}

// frontFix is what the check prints after each file that holds a key.
const frontFix = ": front matter holds fitnessFunctions or relatedConfigurations, which no check reads; run `fitness strip-front-matter`"

// byteOrderMark is the UTF-8 BOM some editors put before a file's first line.
const byteOrderMark = string(rune(0xFEFF))

// fitnessBlock is a plain block holding both fitness keys.
const fitnessBlock = "---\nfitnessFunctions:\n  - prose-budget\nrelatedConfigurations: ['../.fitnessrc.json']\n---\n\n"

// flagged asserts out names each file once with the fix.
func flagged(t *testing.T, out string, files ...string) {
	t.Helper()
	for _, f := range files {
		if n := strings.Count(read(out), read(f+frontFix)); n != 1 {
			t.Errorf("%s is named %d times, want once:\n%s", f, n, out)
		}
	}
}

func Test0123_1(t *testing.T) {
	t.Parallel()
	out, code := noFitnessFront(t, map[string]string{
		"docs/guide.md":                    fitnessBlock + "# Guide\n",
		"changelog/2026-10-11-0900-one.md": entryFileText("2026.10.11.0900", "Feat: one change."),
		".github/PULL_REQUEST_TEMPLATE.md": "<!--\n---\nfitnessFunctions: ['pr-structure']\n---\n-->\n<!-- One sentence. -->\n> REPLACE-ME\n",
	})
	sees(t, out, code, 1)
	flagged(t, out, "docs/guide.md", "changelog/2026-10-11-0900-one.md", ".github/PULL_REQUEST_TEMPLATE.md")
}

func Test0123_2(t *testing.T) {
	t.Parallel()
	out, code := noFitnessFront(t, map[string]string{
		"docs/crlf.md":                  "---\r\nrelatedConfigurations: ['../.fitnessrc.json']\r\n---\r\n\r\n# Crlf\r\n",
		"docs/bom.md":                   byteOrderMark + "---\nfitnessFunctions: ['prose-budget']\n---\n\n# Bom\n",
		".github/ISSUE_TEMPLATE/bug.md": "---\nname: Bug\nrelatedConfigurations: ['../../.fitnessrc.json']\nlabels: Bug\n---\n\n## Steps\n",
	})
	sees(t, out, code, 1)
	flagged(t, out, "docs/crlf.md", "docs/bom.md", ".github/ISSUE_TEMPLATE/bug.md")
}

func Test0123_3(t *testing.T) {
	t.Parallel()
	out, code := noFitnessFront(t, map[string]string{
		"docs/notes.md":                     "---\ntitle: Notes\n---\n\n# Notes\n",
		"docs/ruled.md":                     "---\n\nrelatedConfigurations: a note set between two rules.\n\n---\n\n# Ruled\n",
		".github/ISSUE_TEMPLATE/bug.md":     "---\nname: Bug\nlabels: Bug\n---\n\n## Steps\n",
		"tests/examples/demoRepo/README.md": fitnessBlock + "# Demo\n",
		"go/tool/testdata/guide.md":         fitnessBlock + "# Guide\n",
	})
	sees(t, out, code, 0, "All 1 checks passed")
	if strings.Contains(out, "strip-front-matter") {
		t.Errorf("a file without fitness front matter, or a fixture, was flagged:\n%s", out)
	}
}
