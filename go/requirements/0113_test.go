package requirements

import (
	"strings"
	"testing"
)

// subcommands are the commands fitness runs besides checks.
var subcommands = strings.Fields(`llms list explain docs check init hook boxes
strip-front-matter branch-check pr-check issue-check plan-check close-check
dupe-check comment-check`)

func Test0113_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	page, code := fitness(t, repo, nil, "llms")
	sees(t, page, code, 0, "# fitness", "## Exit codes", "## Commands", "## Checks", "## Topics")
	for _, name := range subcommands {
		sees(t, page, code, 0, "`fitness "+name)
	}
	for _, check := range strings.Fields(catalogRows) {
		sees(t, page, code, 0, "- `"+check+"`: ")
	}
	topics, _ := fitness(t, repo, nil, "docs")
	for _, name := range topicNames(topics) {
		sees(t, page, code, 0, "`fitness docs "+name+"`")
	}
}

func Test0113_2(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "--help")
	sees(t, out, code, 0, "Run `fitness llms`")
	for _, name := range subcommands {
		sees(t, out, code, 0, "  fitness "+name)
	}
}
