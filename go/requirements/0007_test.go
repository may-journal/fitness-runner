package requirements

import (
	"strings"
	"testing"
)

// proseBudget runs `fitness-install -- prose-budget <args>` on happyRepo
// with files written over it.
func proseBudget(t *testing.T, files map[string]string, args ...string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, append([]string{"prose-budget"}, args...)...)
}

// longSentence has 30 words, past the 23-word default.
var longSentence = strings.Repeat("word ", 29) + "end.\n"

func Test0007_1(t *testing.T) {
	t.Parallel()
	out, code := proseBudget(t, map[string]string{"README.md": readme + "\n" + longSentence})
	sees(t, out, code, 1, "README.md: a sentence has 30 words (max 23)")
}

func Test0007_2(t *testing.T) {
	t.Parallel()
	body := readme + "\n## Notes\n"
	for i := 0; i < 16; i++ {
		body += "\n- " + strings.Repeat("item word ", 10) + "\n"
	}
	out, code := proseBudget(t, map[string]string{"README.md": body})
	sees(t, out, code, 1, `README.md: section "Notes" has 320 prose words (max 300)`)
}

func Test0007_3(t *testing.T) {
	t.Parallel()
	changelog := "---\nrelatedConfigurations: ['.fitnessrc.json']\n---\n\n# Changelog\n\n## Changes\n\n### 2026.10.08.1400\n\n- Docs: " + longSentence
	out, code := proseBudget(t, map[string]string{"CHANGELOG.md": changelog})
	sees(t, out, code, 1, "CHANGELOG.md: a list item has 31 words (max 23)")
}

func Test0007_4(t *testing.T) {
	t.Parallel()
	out, code := proseBudget(t, map[string]string{"body.md": "## Background\n\n" + longSentence}, "--body-file", "body.md")
	sees(t, out, code, 1, "a sentence has 30 words (max 23)")
}

func Test0007_5(t *testing.T) {
	t.Parallel()
	nested := readme + "\n- Top item\n" + strings.Repeat("    - Child item\n", 5)
	out, code := proseBudget(t, map[string]string{"README.md": nested})
	sees(t, out, code, 1, "README.md: a list has more than 4 items at level 2")
}

func Test0007_6(t *testing.T) {
	t.Parallel()
	out, code := proseBudget(t, map[string]string{"README.md": readme + "\nOne. Two. Three. Four. Five.\n"})
	sees(t, out, code, 1, "README.md: a paragraph has 5 sentences (max 4)")
}

func Test0007_7(t *testing.T) {
	t.Parallel()
	out, code := proseBudget(t, map[string]string{"README.md": readme + "\n```\n" + longSentence + "```\n"})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0007_8(t *testing.T) {
	t.Parallel()
	out, code := proseBudget(t, map[string]string{
		".fitnessrc.json": `{"proseBudget": {"maxSentenceWords": 5}}` + "\n",
		"README.md":       readme + "\nThis sentence has exactly seven short words.\n",
	})
	sees(t, out, code, 1, "README.md: a sentence has 7 words (max 5)")
}
