package architecture

import (
	"os"
	"path"
	"path/filepath"
	"strings"
)

// repository answers path questions about tracked files. kept holds the
// files .gitattributes does not mark generated.
type repository struct {
	root                string
	files, folder, kept map[string]bool
	// dev holds the files only package.json scripts run.
	dev map[string]bool
	top map[string]bool
}

// sorted lists every tracked file in path order.
func (r repository) sorted() []string { return sortedKeys(r.files) }

func newRepository(root string, files []string) repository {
	r := repository{root: root, files: map[string]bool{}, folder: map[string]bool{}, top: map[string]bool{}}
	for _, file := range files {
		r.files[file] = true
		r.addFolders(file)
	}
	return r
}

func (r repository) addFolders(file string) {
	first, _, nested := strings.Cut(file, "/")
	if nested {
		r.top[first] = true
	}
	for dir := path.Dir(file); dir != "."; dir = path.Dir(dir) {
		r.folder[dir] = true
	}
}

// candidate reports whether text names a repository path: no spaces or
// scheme, and either . for the repo root, a slash after a first segment
// that is a tracked top-level folder, or, when bare names count, a
// tracked root-level source. Other text, such as a path in a consumer's
// install folder or an untracked file name, is prose.
func (r repository) candidate(text string, bare bool) (string, bool) {
	text = strings.Trim(text, "()[]{},;:'\"")
	if prose(text) {
		return "", false
	}
	if text == "." {
		return text, true
	}
	if !strings.Contains(text, "/") {
		return text, bare && r.rootSource(text)
	}
	first, _, _ := strings.Cut(text, "/")
	return path.Clean(text), r.top[first]
}

// prose reports text no path can be: it holds spaces or a scheme.
func prose(text string) bool {
	return strings.ContainsAny(text, " \t") || strings.Contains(text, "://")
}

// rootSource reports a tracked root-level source file named bare.
func (r repository) rootSource(name string) bool {
	return sourceFile(name) && r.files[name]
}

// exists reports a tracked file or folder, the root . included; a
// pattern such as go/cmd/fitness-check-* exists when it matches any of
// them.
func (r repository) exists(p string) bool {
	if !strings.Contains(p, "*") {
		return p == "." || r.files[p] || r.folder[p]
	}
	return anyCovered(p, r.folder) || anyCovered(p, r.files)
}

func anyCovered(pattern string, paths map[string]bool) bool {
	for p := range paths {
		if covers(pattern, p) {
			return true
		}
	}
	return false
}

// read returns a tracked file's lines; nil when it cannot be read.
func (r repository) read(file string) []string {
	body, err := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(file)))
	if err != nil {
		return nil
	}
	return strings.Split(string(body), "\n")
}

// lines counts a tracked file's lines; zero when it cannot be read.
func (r repository) lines(file string) int {
	body, err := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(file)))
	if err != nil {
		return 0
	}
	return len(strings.Split(strings.TrimSuffix(string(body), "\n"), "\n"))
}

// covers reports whether claim names p or a folder holding it, the root
// . holding every path. A claim with a pattern, such as
// go/cmd/fitness-check-*, names every sibling it matches; a star never
// crosses a slash.
func covers(claim, p string) bool {
	if !strings.Contains(claim, "*") {
		return claim == p || claim == "." || strings.HasPrefix(p, claim+"/")
	}
	depth := strings.Count(claim, "/") + 1
	parts := strings.SplitN(p, "/", depth+1)
	if len(parts) < depth {
		return false
	}
	matched, _ := path.Match(claim, strings.Join(parts[:depth], "/"))
	return matched
}
