package walkfs

import (
	"bytes"
	"os/exec"
	"strings"
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

// generatedSet asks `git check-attr` which files carry linguist-generated.
func generatedSet(root string, files []string) map[string]bool {
	cmd := exec.Command("git", "-C", root, "check-attr", "-z", "--stdin", generatedAttr)
	cmd.Stdin = strings.NewReader(strings.Join(files, "\x00") + "\x00")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	return parseCheckAttr(out)
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
