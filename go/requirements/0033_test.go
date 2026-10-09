package requirements

import (
	"strings"
	"testing"
)

// camelDocsCheck is the check under test.
const camelDocsCheck = "markdown-filename-camel-case"

// camelDocsRepo is a repo whose docs follow camelCase, with files added.
func camelDocsRepo(files map[string]string) map[string]string {
	return with(map[string]string{
		"docs/releaseNotes.md": "# Release notes\n",
		"docs/installGuide.md": "# Install guide\n",
	}, files)
}

// camelDocs runs `fitness-install -- markdown-filename-camel-case` on
// happyRepo with files written over it.
func camelDocs(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, camelDocsCheck)
}

func Test0033_1(t *testing.T) {
	t.Parallel()
	out, code := camelDocs(t, camelDocsRepo(map[string]string{"docs/api-design.md": "# API\n"}))
	sees(t, out, code, 1, "docs/api-design.md: filename must be camelCase")
}

func Test0033_2(t *testing.T) {
	t.Parallel()
	out, code := camelDocs(t, camelDocsRepo(map[string]string{
		"docs/api_design.md": "# API\n",
		"docs/ApiDesign.md":  "# API\n",
	}))
	sees(t, out, code, 1, "docs/api_design.md: filename must be camelCase", "docs/ApiDesign.md: filename must be camelCase")
}

func Test0033_3(t *testing.T) {
	t.Parallel()
	out, code := camelDocs(t, camelDocsRepo(nil))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0033_4(t *testing.T) {
	t.Parallel()
	out, code := camelDocs(t, camelDocsRepo(map[string]string{
		"CODE_OF_CONDUCT.md": "# Code of conduct\n",
		"docs/AGENTS.md":     "# Agents\n",
	}))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0033_5(t *testing.T) {
	t.Parallel()
	out, code := camelDocs(t, camelDocsRepo(map[string]string{"Guides/Api_Docs/apiDesign.md": "# API\n"}))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0033_6(t *testing.T) {
	t.Parallel()
	out, code := camelDocs(t, map[string]string{
		"docs/api-design.md":   "# API\n",
		"docs/user-guide.md":   "# Guide\n",
		"docs/releaseNotes.md": "# Release notes\n",
	})
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, camelDocsCheck)
}

func Test0033_7(t *testing.T) {
	t.Parallel()
	out, code := camelDocs(t, map[string]string{
		"docs/api-design.md":   "# API\n",
		"docs/releaseNotes.md": "# Release notes\n",
	})
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, camelDocsCheck)
}

func Test0033_8(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", camelDocsRepo(map[string]string{
		"docs/old-notes.md": "# Old\n",
		"docs/userGuide.md": "# Guide\n",
	}))
	write(t, repo, map[string]string{"docs/new-notes.md": "# New\n"})
	git(t, repo, "add", "-A")
	out, code := fitness(t, repo, nil, camelDocsCheck)
	sees(t, out, code, 1, "docs/new-notes.md: filename must be camelCase")
	if strings.Contains(out, "old-notes.md") {
		t.Errorf("only the staged file may be judged:\n%s", out)
	}
}
