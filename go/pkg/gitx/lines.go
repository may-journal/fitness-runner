package gitx

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Lines is a run of lines, counted from 1, From to To inclusive.
type Lines struct{ From, To int }

// hunkHeader reads the new side of a `git diff -U0` hunk: its first line
// and, when not 1, its count.
var hunkHeader = regexp.MustCompile(`^@@ -\S+ \+(\d+)(?:,(\d+))? @@`)

// ChangedLines maps each file the work tree changes since rev to the lines it
// adds or rewrites, numbered as in the work tree. It covers the files in only, or
// every file when only is empty, and a moved file counts only its edits.
func (r Repo) ChangedLines(rev string, only ...string) (map[string][]Lines, error) {
	args := []string{"diff", "-U0", "-M", "--no-color", "--no-ext-diff", "--src-prefix=a/", "--dst-prefix=b/", rev}
	if r.allAt(rev, only) {
		args = append(append(args, "--"), literal(only)...)
	}
	out, err := r.run(args...)
	if err != nil {
		return nil, err
	}
	return parseHunks(out), nil
}

// maxPaths caps the paths one diff names, keeping the command line short.
const maxPaths = 500

// allAt reports whether rev holds every file in only, so none is a move's target.
// A diff naming just those files then finds the same lines and stats no other file.
func (r Repo) allAt(rev string, only []string) bool {
	if len(only) == 0 || len(only) > maxPaths {
		return false
	}
	out, err := r.run(append([]string{"ls-tree", "-r", "-z", "--name-only", rev, "--"}, literal(only)...)...)
	if err != nil {
		return false
	}
	held := strings.Split(out, "\x00")
	return !slices.ContainsFunc(only, func(f string) bool { return !slices.Contains(held, f) })
}

// literal marks each path so git reads none of its characters as a pattern.
func literal(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = ":(literal)" + p
	}
	return out
}

// parseHunks reads the new side of each hunk in a `git diff -U0` patch.
func parseHunks(patch string) map[string][]Lines {
	changed := map[string][]Lines{}
	var file, prev string
	for _, line := range strings.Split(patch, "\n") {
		header, isHeader := fileHeader(prev, line)
		prev = line
		if isHeader {
			file = header
		} else if lines, ok := hunkLines(line); ok {
			changed[file] = append(changed[file], lines)
		}
	}
	return changed
}

// fileHeader reads the path a `+++ ` line names, but only right after a `--- `
// line. Git ends a path that holds a space with a tab.
func fileHeader(prev, line string) (string, bool) {
	name, ok := strings.CutPrefix(line, "+++ ")
	if !ok || !strings.HasPrefix(prev, "--- ") {
		return "", false
	}
	return strings.TrimPrefix(unquote(strings.TrimSuffix(name, "\t")), "b/"), true
}

// hunkLines reads the lines a hunk header adds; ok is false for any other
// line, or for a hunk that only deletes.
func hunkLines(line string) (lines Lines, ok bool) {
	m := hunkHeader.FindStringSubmatch(line)
	if m == nil {
		return Lines{}, false
	}
	from, _ := strconv.Atoi(m[1])
	count := 1
	if m[2] != "" {
		count, _ = strconv.Atoi(m[2])
	}
	return Lines{From: from, To: from + count - 1}, count > 0
}

// unquote reads a path git quoted for its unusual bytes, which it writes
// as Go writes octal escapes.
func unquote(name string) string {
	if plain, err := strconv.Unquote(name); err == nil && strings.HasPrefix(name, `"`) {
		return plain
	}
	return name
}
