package comments

import (
	"os"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// Uncommented is src without its comments, its spacing collapsed to single
// spaces, so two versions that differ only in comments read the same.
// Directives and shebangs stay, since they change what a tool does. It
// reports false when no reader covers the file at rel.
func Uncommented(rel string, src []byte) (string, bool) {
	text := strings.ReplaceAll(string(src), "\r\n", "\n")
	read, ok := readers[Lang(rel, []byte(text))]
	if !ok {
		return "", false
	}
	starts := newOffsets(text).starts
	var code strings.Builder
	last := 0
	for _, n := range read([]byte(text)) {
		start := starts[n.line-1] + n.col
		if directive(n.text) || start < last {
			continue
		}
		code.WriteString(text[last:start] + " ")
		last = start + len(n.text)
	}
	code.WriteString(text[last:])
	return strings.Join(strings.Fields(code.String()), " "), true
}

// CommentOnly reports whether file holds the same code at base as in the
// scope being judged. From is its path at base, before any rename.
func CommentOnly(root, base, from, file string) bool {
	before, ok := gitx.Show(root, base+":"+from)
	after, found := gitx.Show(root, judged()+file)
	if base == "" || !ok || !found {
		return false
	}
	was, ok := Uncommented(from, []byte(before))
	is, _ := Uncommented(file, []byte(after))
	return ok && was == is
}

// judged prefixes a path with what a run judges: HEAD for a pull request,
// else the index, which holds what a commit records.
func judged() string {
	if os.Getenv("GITHUB_BASE_REF") != "" {
		return "HEAD:"
	}
	return ":"
}
