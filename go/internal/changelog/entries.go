package changelog

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// EntryDir is the folder that holds one entry file per pull request, the
// preferred way to add an entry, so two open pull requests never edit the
// same lines.
const EntryDir = "changelog"

// Entry is one entry file: its slash path from the repo root and its text.
type Entry struct {
	Path string
	Text string
}

// Stamp is the entry's first dated heading, empty when it has none.
func (e Entry) Stamp() string {
	if stamps := Headings(e.Text); len(stamps) > 0 {
		return stamps[0]
	}
	return ""
}

// IsEntryPath reports whether p, a slash path from the repo root, is an
// entry file: a Markdown file directly in EntryDir.
func IsEntryPath(p string) bool {
	return path.Dir(p) == EntryDir && path.Ext(p) == ".md"
}

// ReadEntries returns root's entry files, newest stamp first. A repo with
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

// SortEntries orders entries newest stamp first, then by path.
func SortEntries(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.Stamp() != b.Stamp() {
			return a.Stamp() > b.Stamp()
		}
		return a.Path < b.Path
	})
}

var (
	// h3Re is any level-3 heading.
	h3Re = regexp.MustCompile(`(?m)^###\s`)
	// otherHeadingRe is a heading of any level but three.
	otherHeadingRe = regexp.MustCompile(`(?m)^(#{1,2}|#{4,})\s`)
	// datedLineRe is a dated entry heading line.
	datedLineRe = regexp.MustCompile(`^### \d{4}\.\d{2}\.\d{2}\.\d{4}`)
	// foldStopRe is a level-2 heading, where a release section ends.
	foldStopRe = regexp.MustCompile(`^## `)
	// frontMatterRe is a leading YAML front matter block.
	frontMatterRe = regexp.MustCompile(`\A---\n(?s:.*?)\n---\n`)
)

// Shaped reports whether an entry holds exactly one heading, a dated ###
// one, so it folds into CHANGELOG.md as one section.
func (e Entry) Shaped() bool {
	body := frontMatterRe.ReplaceAllString(e.Text, "")
	return len(h3Re.FindAllString(body, -1)) == 1 && len(Headings(body)) == 1 && !otherHeadingRe.MatchString(body)
}

// Fold places entries, newest first, into a CHANGELOG.md text. They go
// above the first dated section, or right below the release notes of the
// newest Release Please section when one comes first, else at the end. So
// a release PR lists the entries it ships under its version heading.
func Fold(text string, entries []Entry) string {
	if len(entries) == 0 {
		return text
	}
	var block strings.Builder
	for _, e := range entries {
		block.WriteString(strings.TrimSpace(frontMatterRe.ReplaceAllString(e.Text, "")) + "\n\n")
	}
	lines := strings.SplitAfter(text, "\n")
	at := foldIndex(lines)
	head := strings.Join(lines[:at], "")
	if at == len(lines) && head != "" {
		head = strings.TrimRight(head, "\n") + "\n\n"
	}
	return head + block.String() + strings.Join(lines[at:], "")
}

// foldIndex is the line entries go before: the first dated heading, or the
// first level-2 heading after a release heading, else the end.
func foldIndex(lines []string) int {
	inRelease := false
	for i, line := range lines {
		line = strings.TrimRight(line, "\n")
		if foldsBefore(line, inRelease) {
			return i
		}
		inRelease = inRelease || releaseRe.MatchString(line)
	}
	return len(lines)
}

// foldsBefore reports whether entries go above line: a dated heading, or a
// level-2 heading that ends the release section entries follow.
func foldsBefore(line string, inRelease bool) bool {
	return datedLineRe.MatchString(line) || (inRelease && foldStopRe.MatchString(line))
}

// BranchEntryFiles returns the entry files the current branch's own commits
// add. Like BranchEntries, it is nil off a feature branch.
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
