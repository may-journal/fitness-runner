package architecture

import (
	"os"
	"path"
	"path/filepath"
	"strings"
)

// repository answers path questions about tracked files.
type repository struct {
	root          string
	files, folder map[string]bool
	top           map[string]bool
}

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

func (r repository) exists(p string) bool { return r.files[p] || r.folder[p] }

// lines counts a tracked file's lines; zero when it cannot be read.
func (r repository) lines(file string) int {
	body, err := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(file)))
	if err != nil {
		return 0
	}
	return len(strings.Split(strings.TrimSuffix(string(body), "\n"), "\n"))
}

// covers reports whether claim names p or a folder holding it.
func covers(claim, p string) bool {
	return claim == p || strings.HasPrefix(p, claim+"/")
}
