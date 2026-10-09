package requirements

import (
	"maps"
	"strings"
	"testing"
)

// kebabDocsCheck is the check under test.
const kebabDocsCheck = "markdown-filename-kebab-case"

// kebabDocsRepo is a repo whose docs follow kebab-case, with files added.
func kebabDocsRepo(files map[string]string) map[string]string {
	repo := map[string]string{
		"docs/api-design.md": "# API\n",
		"docs/user-guide.md": "# Guide\n",
	}
	maps.Copy(repo, files)
	return repo
}

// kebabDocs runs `fitness-install -- markdown-filename-kebab-case` on
// happyRepo with files written over it.
func kebabDocs(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, kebabDocsCheck)
}

func Test0034_1(t *testing.T) {
	t.Parallel()
	out, code := kebabDocs(t, kebabDocsRepo(map[string]string{"docs/releaseNotes.md": "# Notes\n"}))
	sees(t, out, code, 1, "docs/releaseNotes.md: filename must be kebab-case")
}

func Test0034_2(t *testing.T) {
	t.Parallel()
	out, code := kebabDocs(t, kebabDocsRepo(map[string]string{
		"docs/api_notes.md": "# Notes\n",
		"docs/ApiNotes.md":  "# Notes\n",
		"docs/api notes.md": "# Notes\n",
	}))
	sees(t, out, code, 1,
		"docs/api_notes.md: filename must be kebab-case",
		"docs/ApiNotes.md: filename must be kebab-case",
		"docs/api notes.md: filename must be kebab-case")
}

func Test0034_3(t *testing.T) {
	t.Parallel()
	out, code := kebabDocs(t, kebabDocsRepo(map[string]string{
		"architecture/02-containers.md": "# Containers\n",
		"docs/adr001.md":                "# ADR\n",
	}))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0034_4(t *testing.T) {
	t.Parallel()
	out, code := kebabDocs(t, kebabDocsRepo(map[string]string{
		"CODE_OF_CONDUCT.md": "# Code of conduct\n",
		"docs/AGENTS.md":     "# Agents\n",
	}))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0034_5(t *testing.T) {
	t.Parallel()
	out, code := kebabDocs(t, kebabDocsRepo(map[string]string{"Guides/Api_Docs/api-design.md": "# API\n"}))
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0034_6(t *testing.T) {
	t.Parallel()
	out, code := kebabDocs(t, map[string]string{
		"docs/releaseNotes.md": "# Notes\n",
		"docs/installGuide.md": "# Guide\n",
		"docs/api-design.md":   "# API\n",
	})
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, kebabDocsCheck)
}

func Test0034_7(t *testing.T) {
	t.Parallel()
	out, code := kebabDocs(t, map[string]string{
		"docs/api-design.md":   "# API\n",
		"docs/releaseNotes.md": "# Notes\n",
	})
	sees(t, out, code, 1, "docs/releaseNotes.md: filename must be kebab-case")
}

func Test0034_8(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", kebabDocsRepo(map[string]string{
		"docs/oldNotes.md":    "# Old\n",
		"docs/setup-steps.md": "# Setup\n",
	}))
	write(t, repo, map[string]string{"docs/newNotes.md": "# New\n"})
	git(t, repo, "add", "-A")
	out, code := fitness(t, repo, nil, kebabDocsCheck)
	sees(t, out, code, 1, "docs/newNotes.md: filename must be kebab-case")
	if strings.Contains(out, "oldNotes.md") {
		t.Errorf("only the staged file may be judged:\n%s", out)
	}
}
