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
	// plainFront is a leading --- block and the blank lines after it.
	plainFront = regexp.MustCompile(`\A---\r?\n((?s:.*?)\r?\n)?---\r?\n(?:\r?\n)*`)
	// commentFront is the same block inside an HTML comment that opens on
	// the first line, the form a PR template carries.
	commentFront = regexp.MustCompile(`\A<!--\r?\n---\r?\n((?s:.*?)\r?\n)?---\r?\n-->\r?\n(?:\r?\n)*`)
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

// stripFront returns content without its front matter when every key in it
// is a fitness key. When a fitness key sits beside other keys, it returns
// content unchanged and names those other keys. Content without fitness
// keys comes back unchanged with no names.
func stripFront(content string) (string, []string) {
	for _, re := range []*regexp.Regexp{plainFront, commentFront} {
		m := re.FindStringSubmatchIndex(content)
		if m == nil {
			continue
		}
		keys := frontKeys(content[m[0]:m[1]])
		others := slices.DeleteFunc(slices.Clone(keys), func(k string) bool { return slices.Contains(fitnessKeys, k) })
		switch {
		case len(others) == len(keys):
			return content, nil
		case len(others) > 0:
			return content, others
		}
		return content[m[1]:], nil
	}
	return content, nil
}

// frontKeys lists the top-level entries of a front matter block in order.
// Indented lines, list items and comments belong to the entry above. A line
// that is not a key, such as prose after a horizontal rule, counts as an
// entry no fitness key matches, so that block is kept whole.
func frontKeys(block string) []string {
	var keys []string
	for _, raw := range strings.Split(block, "\n") {
		line := strings.TrimRight(raw, "\r")
		if line == "" || line == "<!--" || strings.ContainsAny(line[:1], " \t-#") {
			continue
		}
		name, _, _ := strings.Cut(line, ":")
		keys = append(keys, strings.TrimSpace(name))
	}
	return keys
}
