package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/report"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// fitnessKeys are the front matter keys only fitness ever read. A block
// holding nothing else is safe to drop; any other key belongs to a tool
// that may still need it, such as GitHub's issue template fields.
var fitnessKeys = []string{"fitnessFunctions", "relatedConfigurations"}

var (
	// plainFront is a leading --- block and the blank lines after it. The
	// block is empty or opens on a non-blank line, so a rule followed by
	// a blank line never matches.
	plainFront = regexp.MustCompile(`\A---\r?\n((?:[^\r\n](?s:.*?))?\r?\n)?---\r?\n(?:\r?\n)*`)
	// commentFront is the same block inside an HTML comment that opens on
	// the first line, the form a PR template carries.
	commentFront = regexp.MustCompile(`\A<!--\r?\n---\r?\n((?:[^\r\n](?s:.*?))?\r?\n)?---\r?\n-->\r?\n(?:\r?\n)*`)
)

// runStripFrontMatter runs `fitness strip-front-matter`: it removes every
// tracked Markdown file's front matter that holds only fitness keys, keeps
// front matter that holds any other key, and prints one line per file.
func runStripFrontMatter(args []string) int {
	root, err := os.Getwd()
	if err == nil && len(args) > 0 {
		err = fmt.Errorf("takes no arguments, got %s", strings.Join(args, " "))
	}
	if err != nil {
		report.Error("fitness strip-front-matter", err)
		return 1
	}
	stripped, kept, err := stripAll(root)
	if err != nil {
		report.Error("fitness strip-front-matter", err)
		return 1
	}
	fmt.Printf("fitness strip-front-matter: stripped %d files, kept %d\n", stripped, kept)
	return 0
}

// stripAll strips each Markdown file under root and counts the files it
// stripped and those it kept for their other keys.
func stripAll(root string) (stripped, kept int, err error) {
	counts := map[string]int{}
	for _, rel := range walkfs.FilesByExt(root, ".md") {
		outcome, err := stripFile(root, rel)
		if err != nil {
			return counts["stripped"], counts["kept"], err
		}
		counts[outcome]++
	}
	return counts["stripped"], counts["kept"], nil
}

// stripFile strips one file, prints what it did, and returns "stripped",
// "kept" for front matter held for its other keys, or "" when untouched.
func stripFile(root, rel string) (string, error) {
	path := filepath.Join(root, filepath.FromSlash(rel))
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	body, others := stripFront(string(data))
	if len(others) > 0 {
		fmt.Printf("kept %s: front matter also holds %s\n", rel, strings.Join(others, ", "))
		return "kept", nil
	}
	if body == string(data) {
		return "", nil
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return "", err
	}
	fmt.Println("stripped " + rel)
	return "stripped", nil
}

// byteOrderMark is the UTF-8 BOM some editors put before a file's first line.
const byteOrderMark = "\ufeff"

// keyLine is a top-level front matter entry: a name, then a colon.
var keyLine = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]*\s*:`)

// stripFront returns content without its front matter when every key in it
// is a fitness key, or when the block is empty. When a fitness key sits
// beside other keys, it returns content unchanged and names those other
// keys. Content without fitness keys, or whose leading --- opens a rule
// rather than front matter, comes back unchanged with no names. A BOM
// before the block stays at the top of the body.
func stripFront(content string) (string, []string) {
	bom := ""
	if strings.HasPrefix(content, byteOrderMark) {
		bom, content = byteOrderMark, strings.TrimPrefix(content, byteOrderMark)
	}
	for _, re := range []*regexp.Regexp{plainFront, commentFront} {
		if body, others, ok := stripWith(re, content); ok {
			return bom + body, others
		}
	}
	return bom + content, nil
}

// stripWith applies one block form to content. It reports false when the
// form does not hold front matter that is safe to decide on.
func stripWith(re *regexp.Regexp, content string) (string, []string, bool) {
	m := re.FindStringIndex(content)
	if m == nil {
		return content, nil, false
	}
	keys, ok := frontKeys(content[m[0]:m[1]])
	others := slices.DeleteFunc(slices.Clone(keys), func(k string) bool { return slices.Contains(fitnessKeys, k) })
	switch {
	case !ok, len(keys) > 0 && len(others) == len(keys):
		return content, nil, false
	case len(others) > 0:
		return content, others, true
	}
	return content[m[1]:], nil, true
}

// frontKeys lists the top-level entries of a front matter block in order.
// Indented lines, list items and comments belong to the entry above. It
// reports false when a line is neither an entry nor a continuation, as with
// prose between two horizontal rules.
func frontKeys(block string) ([]string, bool) {
	lines := strings.Split(strings.ReplaceAll(block, "\r\n", "\n"), "\n")
	var keys []string
	for _, line := range lines[slices.Index(lines, "---")+1:] {
		switch {
		case line == "---":
			return keys, true
		case keyLine.MatchString(line):
			name, _, _ := strings.Cut(line, ":")
			keys = append(keys, strings.TrimSpace(name))
		case !continuesEntry(line):
			return nil, false
		}
	}
	return keys, true
}

// continuesEntry reports whether a line belongs to the entry above it: it
// is blank, indented, a list item or a comment.
func continuesEntry(line string) bool {
	return line == "" || strings.ContainsAny(line[:1], " \t-#")
}
