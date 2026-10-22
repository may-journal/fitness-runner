package requirements

import (
	"strings"
	"testing"
)

// dirRepo is happyRepo with a badly named doc in each of two folders.
func dirRepo(t *testing.T) string {
	t.Helper()
	return example(t, "happyRepo", kebabDocsRepo(map[string]string{
		"docs/releaseNotes.md": "# Notes\n",
		"guides/userGuide.md":  "# Guide\n",
	}))
}

func Test0100_1(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, dirRepo(t), nil, kebabDocsCheck, "--dir=guides")
	sees(t, out, code, 1, "guides/userGuide.md: filename must be kebab-case")
	if strings.Contains(out, "releaseNotes") {
		t.Errorf("the output names a file outside guides:\n%s", out)
	}
}

func Test0100_2(t *testing.T) {
	t.Parallel()
	repo := dirRepo(t)
	for _, dir := range []string{"nowhere", "../elsewhere"} {
		out, code := fitness(t, repo, nil, kebabDocsCheck, "--dir="+dir)
		sees(t, out, code, 1, "--dir "+dir)
	}
}
