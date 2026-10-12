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
	top                 map[string]bool
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
// scheme, a slash, and a first segment that is a tracked top-level folder.
// Other text, such as a consumer's node_modules/.bin, is prose.
func (r repository) candidate(text string) (string, bool) {
	text = strings.Trim(text, "()[]{},;:'\"")
	if strings.ContainsAny(text, " \t") || strings.Contains(text, "://") || !strings.Contains(text, "/") {
		return "", false
	}
	first, _, _ := strings.Cut(text, "/")
	return path.Clean(text), r.top[first]
}

// exists reports a tracked file or folder; a pattern such as
// go/cmd/fitness-check-* exists when it matches any of them.
func (r repository) exists(p string) bool {
	if !strings.Contains(p, "*") {
		return r.files[p] || r.folder[p]
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

// lines counts a tracked file's lines; zero when it cannot be read.
func (r repository) lines(file string) int {
	body, err := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(file)))
	if err != nil {
		return 0
	}
	return len(strings.Split(strings.TrimSuffix(string(body), "\n"), "\n"))
}

// covers reports whether claim names p or a folder holding it. A claim
// with a pattern, such as go/cmd/fitness-check-*, names every sibling it
// matches; a star never crosses a slash.
func covers(claim, p string) bool {
	if !strings.Contains(claim, "*") {
		return claim == p || strings.HasPrefix(p, claim+"/")
	}
	depth := strings.Count(claim, "/") + 1
	parts := strings.SplitN(p, "/", depth+1)
	if len(parts) < depth {
		return false
	}
	matched, _ := path.Match(claim, strings.Join(parts[:depth], "/"))
	return matched
}
