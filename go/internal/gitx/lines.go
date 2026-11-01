package gitx

import (
	"regexp"
	"strconv"
	"strings"
)

// Lines is a run of lines, counted from 1, From to To inclusive.
type Lines struct{ From, To int }

// hunkHeader reads the new side of a `git diff -U0` hunk: its first line
// and, when not 1, its count.
var hunkHeader = regexp.MustCompile(`^@@ -\S+ \+(\d+)(?:,(\d+))? @@`)

// ChangedLines maps each file the work tree changes since rev to the lines
// it adds or rewrites. They are numbered as the work tree holds them. A
// moved file counts only the lines its move edits.
func (r Repo) ChangedLines(rev string) (map[string][]Lines, error) {
	out, err := r.run("diff", "-U0", "-M", "--no-color", "--no-ext-diff",
		"--src-prefix=a/", "--dst-prefix=b/", rev)
	if err != nil {
		return nil, err
	}
	return parseHunks(out), nil
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

// fileHeader reads the path a `+++ ` line names. Only a line right after
// a `--- ` line is a header, so an added line that starts with `++` is
// not. Git ends a path that holds a space with a tab.
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
