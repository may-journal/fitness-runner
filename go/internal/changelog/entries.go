package changelog

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// EntryDir is the folder that holds one entry file per pull request, the
// only way to add an entry, so two open pull requests never edit the same
// lines.
const EntryDir = "changelog"

// Entry is one entry file: its slash path from the repo root and its text.
type Entry struct {
	Path string
	Text string
}

var (
	// entryNameRe is an entry file's base name: the date prefix, then the
	// change.
	entryNameRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}-\d{4})-.+\.md$`)
	// datePrefixRe is a leading date, whole or in part, that the expected
	// name replaces.
	datePrefixRe = regexp.MustCompile(`^\d{4}(-\d{2}){0,2}(-\d{4})?(-|$)`)
)

// NameError says why the entry file at p, holding text, is misnamed: its
// name must start with its first dated heading's stamp, dots read as
// hyphens, then the change. It names the expected file and is "" when the
// name fits or text has no dated heading.
func NameError(p, text string) string {
	stamps := Headings(text)
	if len(stamps) == 0 {
		return ""
	}
	date := strings.ReplaceAll(stamps[0], ".", "-")
	base := path.Base(p)
	if m := entryNameRe.FindStringSubmatch(base); m != nil && m[1] == date {
		return ""
	}
	change := datePrefixRe.ReplaceAllString(strings.TrimSuffix(base, ".md"), "")
	if change == "" {
		change = "<change>"
	}
	want := EntryDir + "/" + date + "-" + change + ".md"
	return fmt.Sprintf("%s must be named %s: its heading's date, then the change", p, want)
}

// IsEntryPath reports whether p, a slash path from the repo root, is an
// entry file: a Markdown file directly in EntryDir.
func IsEntryPath(p string) bool {
	return path.Dir(p) == EntryDir && path.Ext(p) == ".md"
}

// ReadEntries returns root's entry files, newest name first. A repo with
// no entry folder has none.
func ReadEntries(root string) ([]Entry, error) {
	files, err := entryDirFiles(root)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, f := range files {
		e, err := readEntry(root, f)
		if err != nil {
			return nil, err
		}
		if e != nil {
			entries = append(entries, *e)
		}
	}
	SortEntries(entries)
	return entries, nil
}

// entryDirFiles lists the entry folder; a missing folder lists nothing.
func entryDirFiles(root string) ([]os.DirEntry, error) {
	files, err := os.ReadDir(filepath.Join(root, EntryDir))
	if os.IsNotExist(err) {
		return nil, nil
	}
	return files, err
}

// readEntry reads f from the entry folder, or nil when f is no entry file.
func readEntry(root string, f os.DirEntry) (*Entry, error) {
	p := EntryDir + "/" + f.Name()
	if f.IsDir() || !IsEntryPath(p) {
		return nil, nil
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
	if err != nil {
		return nil, err
	}
	return &Entry{Path: p, Text: string(raw)}, nil
}

// SortEntries orders entries newest first by the date their file name
// starts with, then by name. A name without a date, which the checks fail,
// sorts by its heading's date, so release notes keep release order.
func SortEntries(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := sortDate(entries[i]), sortDate(entries[j])
		if a != b {
			return a > b
		}
		return entries[i].Path > entries[j].Path
	})
}

// sortDate is the date an entry's file name starts with, else its first
// heading's date with dots read as hyphens, else "".
func sortDate(e Entry) string {
	if m := entryNameRe.FindStringSubmatch(path.Base(e.Path)); m != nil {
		return m[1]
	}
	if stamps := Headings(e.Text); len(stamps) > 0 {
		return strings.ReplaceAll(stamps[0], ".", "-")
	}
	return ""
}

var (
	// h3Re is any level-3 heading.
	h3Re = regexp.MustCompile(`(?m)^###\s`)
	// otherHeadingRe is a heading of any level but three.
	otherHeadingRe = regexp.MustCompile(`(?m)^(#{1,2}|#{4,})\s`)
	// frontMatterRe is a leading YAML front matter block.
	frontMatterRe = regexp.MustCompile(`\A---\n(?s:.*?)\n---\n`)
)

// Shaped reports whether an entry holds exactly one heading, a dated ###
// one, so a release's notes show it as one section.
func (e Entry) Shaped() bool {
	body := frontMatterRe.ReplaceAllString(e.Text, "")
	return len(h3Re.FindAllString(body, -1)) == 1 && len(Headings(body)) == 1 && !otherHeadingRe.MatchString(body)
}

// Body is the entry's text without its front matter, trimmed: the heading
// and bullets a release's notes show.
func (e Entry) Body() string {
	return strings.TrimSpace(frontMatterRe.ReplaceAllString(e.Text, ""))
}

// BranchEntryFiles returns the entry files the current branch's own commits
// add. It is nil on the default branch, on a detached HEAD, without origin
// refs, or when the branch's own commits add none.
func BranchEntryFiles(root string) []string {
	fork, ok := forkPoint(root)
	if !ok {
		return nil
	}
	return entryPaths(gitOut(root, "diff", "--name-only", "--diff-filter=A", fork, "HEAD", "--", EntryDir))
}

// StagedEntryFiles returns the entry files the index adds over HEAD.
func StagedEntryFiles(root string) []string {
	return entryPaths(gitOut(root, "diff", "--cached", "--name-only", "--diff-filter=A", "--", EntryDir))
}

// StagedNameErrors names each entry file the index adds whose name does not
// fit its staged heading, with the name it should have.
func StagedNameErrors(root string) []string {
	var errs []string
	for _, p := range StagedEntryFiles(root) {
		if msg := NameError(p, gitOut(root, "show", ":"+p)); msg != "" {
			errs = append(errs, msg)
		}
	}
	return errs
}

// AddedText is the text of the lines a unified diff adds, one per line.
func AddedText(patch string) string {
	var lines []string
	for _, line := range strings.Split(patch, "\n") {
		if sign, text := diffLine(line); sign == '+' {
			lines = append(lines, text)
		}
	}
	return strings.Join(lines, "\n")
}

// entryPaths keeps the entry files in git's newline-separated path list.
func entryPaths(list string) []string {
	var paths []string
	for _, p := range strings.Split(list, "\n") {
		if IsEntryPath(p) {
			paths = append(paths, p)
		}
	}
	return paths
}
