package walkfs

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// generatedAttr is the .gitattributes attribute that marks a file as
// generated, the same mark GitHub's linguist reads.
const generatedAttr = "linguist-generated"

// WithoutGenerated drops the files root's .gitattributes marks
// linguist-generated, such as lock files. It is the one way a repo keeps a
// file from a check that cannot judge generated content (cspell, jscpd).
// Outside a git repo, or when git fails, it drops nothing.
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
// not, so each file is asked about once however many walks a check makes.
var generatedMemo = struct {
	sync.Mutex
	roots map[string]map[string]bool
}{roots: map[string]map[string]bool{}}

// generatedSet reports which files carry linguist-generated, asking
// `git check-attr` only about files this process has not asked about yet.
// A git failure caches nothing and marks no file generated.
func generatedSet(root string, files []string) map[string]bool {
	generatedMemo.Lock()
	defer generatedMemo.Unlock()
	known := knownGenerated(filepath.Clean(root))
	if ask := unasked(known, files); len(ask) > 0 && !askGenerated(root, ask, known) {
		return nil
	}
	set := make(map[string]bool)
	for _, f := range files {
		if known[f] {
			set[f] = true
		}
	}
	return set
}

// knownGenerated is root's answer map, created empty on first use. The
// caller holds generatedMemo's lock.
func knownGenerated(root string) map[string]bool {
	known := generatedMemo.roots[root]
	if known == nil {
		known = map[string]bool{}
		generatedMemo.roots[root] = known
	}
	return known
}

// unasked lists the files known holds no answer for yet.
func unasked(known map[string]bool, files []string) []string {
	var ask []string
	for _, f := range files {
		if _, seen := known[f]; !seen {
			ask = append(ask, f)
		}
	}
	return ask
}

// askGenerated runs `git check-attr` on files and records each answer in
// known, reporting whether git answered.
func askGenerated(root string, files []string, known map[string]bool) bool {
	cmd := exec.Command("git", "-C", root, "check-attr", "-z", "--stdin", generatedAttr)
	cmd.Stdin = strings.NewReader(strings.Join(files, "\x00") + "\x00")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	set := parseCheckAttr(out)
	for _, f := range files {
		known[f] = set[f]
	}
	return true
}

// parseCheckAttr reads `git check-attr -z` output — path, attribute, value
// triples, each NUL-terminated — keeping the paths whose value turns the
// attribute on ("set" for a bare name, or "true").
func parseCheckAttr(out []byte) map[string]bool {
	fields := bytes.Split(out, []byte{0})
	set := make(map[string]bool)
	for i := 0; i+2 < len(fields); i += 3 {
		if isOn(string(fields[i+2])) {
			set[string(fields[i])] = true
		}
	}
	return set
}

// isOn reports whether a check-attr value turns the attribute on.
func isOn(value string) bool {
	return value == "set" || value == "true"
}
