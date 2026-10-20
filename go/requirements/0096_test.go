package requirements

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stripRun writes files over happyRepo, runs `fitness strip-front-matter`
// there, and returns the repo, what it printed, and its exit code.
func stripRun(t *testing.T, files map[string]string) (string, string, int) {
	t.Helper()
	repo := example(t, "happyRepo", files)
	out, code := fitness(t, repo, nil, "strip-front-matter")
	return repo, out, code
}

// holds asserts the file at rel in repo reads exactly want.
func holds(t *testing.T, repo, rel, want string) {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rel)))
	mustDo(t, err)
	if string(got) != want {
		t.Errorf("%s reads %q, want %q", rel, got, want)
	}
}

// stripBody is a doc body that opens with a rule and a list, so a strip that
// eats more than the block shows.
const stripBody = "# Guide\n\n---\n\n- relatedConfigurations: stays in the body.\n"

func Test0096_1(t *testing.T) {
	t.Parallel()
	entry := bareEntryText("2026.10.11.0900", "Feat: one change.")
	repo, out, code := stripRun(t, map[string]string{
		"docs/guide.md": "---\nfitnessFunctions:\n  - prose-budget\nrelatedConfigurations: ['../.fitnessrc.json']\n---\n\n" + stripBody,
		"changelog/2026-10-11-0900-one-change.md": entryFileText("2026.10.11.0900", "Feat: one change."),
		".github/PULL_REQUEST_TEMPLATE.md":        "<!--\n---\nfitnessFunctions: ['pr-structure']\n---\n-->\n<!-- One sentence. -->\n> REPLACE-ME\n",
	})
	sees(t, out, code, 0, "stripped docs/guide.md", "stripped changelog/2026-10-11-0900-one-change.md",
		"stripped .github/PULL_REQUEST_TEMPLATE.md", "stripped README.md")
	holds(t, repo, "docs/guide.md", stripBody)
	holds(t, repo, "changelog/2026-10-11-0900-one-change.md", entry)
	holds(t, repo, ".github/PULL_REQUEST_TEMPLATE.md", "<!-- One sentence. -->\n> REPLACE-ME\n")
}

func Test0096_2(t *testing.T) {
	t.Parallel()
	bug := "---\nrelatedConfigurations: ['../../.fitnessrc.json']\nname: Bug\nlabels: Bug\n---\n\n## Steps\n"
	skill := "---\nname: tidy\ndescription: Tidy docs.\nrelatedConfigurations: ['../../.fitnessrc.json']\n---\n\n# Tidy\n"
	repo, out, code := stripRun(t, map[string]string{".github/ISSUE_TEMPLATE/bug.md": bug, "skills/tidy/SKILL.md": skill})
	sees(t, out, code, 0, "kept .github/ISSUE_TEMPLATE/bug.md: front matter also holds name, labels",
		"kept skills/tidy/SKILL.md: front matter also holds name, description")
	holds(t, repo, ".github/ISSUE_TEMPLATE/bug.md", bug)
	holds(t, repo, "skills/tidy/SKILL.md", skill)
}

func Test0096_3(t *testing.T) {
	t.Parallel()
	plain := "# Plain\n\nShort and clean.\n"
	titled := "---\ntitle: Notes\n---\n\n# Notes\n"
	ruled := "---\n\nA note set between two rules.\n\n---\n\n# Ruled\n"
	repo, out, code := stripRun(t, map[string]string{"plain.md": plain, "docs/notes.md": titled, "docs/ruled.md": ruled})
	// happyRepo's own README.md and CHANGELOG.md are the two files stripped.
	sees(t, out, code, 0, "fitness strip-front-matter: stripped 2 files, kept 0")
	if strings.Contains(out, "plain.md") || strings.Contains(out, "notes.md") {
		t.Errorf("the output lists a file it left alone:\n%s", out)
	}
	holds(t, repo, "plain.md", plain)
	holds(t, repo, "docs/notes.md", titled)
	holds(t, repo, "docs/ruled.md", ruled)
}

func Test0096_4(t *testing.T) {
	t.Parallel()
	repo, out, code := stripRun(t, map[string]string{
		"docs/crlf.md":  "---\r\nfitnessFunctions:\r\n  - prose-budget\r\n---\r\n\r\n# Crlf\r\n",
		"docs/bom.md":   "\ufeff---\nfitnessFunctions: ['prose-budget']\n---\n\n# Bom\n",
		"docs/empty.md": "---\n---\n\n# Empty\n",
	})
	sees(t, out, code, 0, "stripped docs/crlf.md", "stripped docs/bom.md", "stripped docs/empty.md")
	holds(t, repo, "docs/crlf.md", "# Crlf\r\n")
	holds(t, repo, "docs/bom.md", "\ufeff# Bom\n")
	holds(t, repo, "docs/empty.md", "# Empty\n")
}
