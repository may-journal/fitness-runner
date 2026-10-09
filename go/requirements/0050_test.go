package requirements

import (
	"strings"
	"testing"
)

// prCloses runs `fitness-install -- pr-closes-issue --body-file body.md
// <args>` on happyRepo with body as the PR description.
func prCloses(t *testing.T, body string, args ...string) (string, int) {
	t.Helper()
	repo := example(t, "happyRepo", map[string]string{"body.md": body})
	return fitness(t, repo, nil, append([]string{"pr-closes-issue", "--body-file", "body.md"}, args...)...)
}

// prClosesNoKeyword is the failure a body with no closing keyword gets.
const prClosesNoKeyword = "PR description has no GitHub closing keyword (close/fix/resolve + #NN)"

// prClosesKeywords is GitHub's fixed set of issue-closing keywords.
var prClosesKeywords = strings.Fields("close closes closed fix fixes fixed resolve resolves resolved")

func Test0050_1(t *testing.T) {
	t.Parallel()
	files := map[string]string{}
	for _, keyword := range prClosesKeywords {
		files[keyword+".md"] = "> pitch\n\n" + strings.ToUpper(keyword[:1]) + keyword[1:] + ": #12\n"
	}
	repo := example(t, "happyRepo", files)
	for _, keyword := range prClosesKeywords {
		out, code := fitness(t, repo, nil, "pr-closes-issue", "--body-file", keyword+".md")
		sees(t, out, code, 0, "All 1 checks passed")
	}
}

func Test0050_2(t *testing.T) {
	t.Parallel()
	out, code := prCloses(t, "This addresses #12, is part of #6, and follows #7.\n")
	sees(t, out, code, 1, prClosesNoKeyword)
}

func Test0050_3(t *testing.T) {
	t.Parallel()
	out, code := prCloses(t, "Implements #5.\n\nPlan #63 for context.\n\nCloses #6.\n")
	sees(t, out, code, 1,
		"PR says it implements #5 but no closing keyword (close/fix/resolve + #5) closes it",
		"PR says it implements #63 but no closing keyword (close/fix/resolve + #63) closes it")
}

func Test0050_4(t *testing.T) {
	t.Parallel()
	out, code := prCloses(t, "Merging `Closes #12` closed it.\n\n```\nFixes #12\n```\n")
	sees(t, out, code, 1, prClosesNoKeyword)
}

func Test0050_5(t *testing.T) {
	t.Parallel()
	out, code := prCloses(t, "Closes #12.\n", "--require-close=12,40")
	sees(t, out, code, 1, "PR closes a Plan that closes #40, so the PR must also close #40")
}

func Test0050_6(t *testing.T) {
	t.Parallel()
	out, code := prCloses(t, "Closes #12 and closes #40.\n", "--require-close=12,40")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0050_7(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "pr-closes-issue")
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "pr-closes-issue")
}
