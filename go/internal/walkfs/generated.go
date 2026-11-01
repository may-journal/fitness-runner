package walkfs

import (
	"path/filepath"
	"sync"

	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// generatedAttr is the .gitattributes attribute that marks a file as
// generated, the same mark GitHub's linguist reads.
const generatedAttr = "linguist-generated"

// WithoutGenerated drops the files root's .gitattributes marks
// linguist-generated, such as lock files. It is the one way a repo keeps a
// file from a check that cannot judge generated content (cspell, duplicate-code).
// Outside a git repo, or when git fails, it drops only files an earlier
// answer in this process marked.
func WithoutGenerated(root string, files []string) []string {
	if len(files) == 0 {
		return files
	}
	generated := generatedSet(root, files)
	var kept []string
	for _, f := range files {
		if !generated[f] {
			kept = append(kept, f)
		}
	}
	return kept
}

// generatedMemo keeps each root's linguist-generated answers for the life
// of the process, true for a generated file and false for one git said is
// not, so a check that walks several times asks about each file once. The
// lock guards only the maps, never a git call.
var generatedMemo = struct {
	sync.Mutex
	roots map[string]map[string]bool
}{roots: map[string]map[string]bool{}}

// generatedSet reports which files carry linguist-generated, asking
// `git check-attr` only about files this process has no answer for yet.
// When git fails, the answers already known still count.
func generatedSet(root string, files []string) map[string]bool {
	key := filepath.Clean(root)
	var answers map[string]bool
	if ask := unasked(key, files); len(ask) > 0 {
		answers = askGenerated(root, ask)
	}
	return rememberAmong(key, answers, files)
}

// unasked lists the files root has no answer for yet.
func unasked(root string, files []string) []string {
	generatedMemo.Lock()
	defer generatedMemo.Unlock()
	known := generatedMemo.roots[root]
	var ask []string
	for _, f := range files {
		if _, seen := known[f]; !seen {
			ask = append(ask, f)
		}
	}
	return ask
}

// rememberAmong stores git's answers for root, if any, and returns the set
// of files that root's known answers mark generated, under one lock.
func rememberAmong(root string, answers map[string]bool, files []string) map[string]bool {
	generatedMemo.Lock()
	defer generatedMemo.Unlock()
	known := generatedMemo.roots[root]
	if known == nil {
		known = make(map[string]bool, len(answers))
		generatedMemo.roots[root] = known
	}
	for f, on := range answers {
		known[f] = on
	}
	set := make(map[string]bool)
	for _, f := range files {
		if known[f] {
			set[f] = true
		}
	}
	return set
}

// askGenerated runs `git check-attr` on files and returns an answer for
// each, or nil when git fails.
func askGenerated(root string, files []string) map[string]bool {
	out, err := gitx.Repo(root).CheckAttr(generatedAttr, files)
	if err != nil {
		return nil
	}
	set := parseCheckAttr([]byte(out))
	answers := make(map[string]bool, len(files))
	for _, f := range files {
		answers[f] = set[f]
	}
	return answers
}

// parseCheckAttr reads `git check-attr -z` output — path, attribute, value
// triples, each NUL-terminated — keeping the paths whose value turns the
// attribute on ("set" for a bare name, or "true"). Git prints no empty
// field, so dropping empty entries keeps the triples aligned.
func parseCheckAttr(out []byte) map[string]bool {
	fields := splitNUL(out)
	set := make(map[string]bool)
	for i := 0; i+2 < len(fields); i += 3 {
		if isOn(fields[i+2]) {
			set[fields[i]] = true
		}
	}
	return set
}

// isOn reports whether a check-attr value turns the attribute on.
func isOn(value string) bool {
	return value == "set" || value == "true"
}
