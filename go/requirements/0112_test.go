package requirements

import (
	"slices"
	"strings"
	"testing"
)

// topicNames reads the topic names `fitness docs` lists: the first word of
// each line after the installer's own lines.
func topicNames(out string) []string {
	var names []string
	for _, line := range strings.Split(out, "\n") {
		if fields := strings.Fields(line); len(fields) > 1 && !strings.HasPrefix(line, "fitness") {
			names = append(names, fields[0])
		}
	}
	return names
}

func Test0112_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", nil)
	out, code := fitness(t, repo, nil, "docs")
	sees(t, out, code, 0, "protocol", "config")
	names := topicNames(out)
	for _, want := range []string{"config", "coverage", "github", "hooks", "languages", "protocol"} {
		if !slices.Contains(names, want) {
			t.Fatalf("fitness docs lists %q, not topic %q:\n%s", names, want, out)
		}
	}
	for _, name := range names {
		page, code := fitness(t, repo, nil, "docs", name)
		sees(t, page, code, 0, "# "+name, "From the doc comments of go/internal/", "Package ")
	}
}

func Test0112_2(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "docs", "nope")
	sees(t, out, code, 2, `no topic named "nope"; the topics are:`, "protocol", "config")
}

func Test0112_3(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", nil), nil, "docs", "--json")
	sees(t, out, code, 2, "name one topic, as `fitness docs [topic]`")
}
