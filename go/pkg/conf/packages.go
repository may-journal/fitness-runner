package conf

import (
	"fmt"
	"maps"
	"path"
	"slices"
)

// PackageRoots returns the folders of files that the packages globs match,
// sorted and once each. A glob that is malformed or matches no folder
// fails, naming it.
func PackageRoots(globs, files []string) ([]string, error) {
	if len(globs) == 0 {
		return nil, nil
	}
	folders := foldersOf(files)
	var roots []string
	for _, glob := range globs {
		matched, err := matchFolders(glob, folders)
		if err != nil {
			return nil, err
		}
		roots = append(roots, matched...)
	}
	slices.Sort(roots)
	return slices.Compact(roots), nil
}

// matchFolders is the folders glob matches, failing when it is malformed
// or matches none. A glob of "." names the repo root.
func matchFolders(glob string, folders []string) ([]string, error) {
	if _, err := path.Match(glob, ""); err != nil {
		return nil, fmt.Errorf("%s: packages glob %q is malformed: %w", FileName, glob, err)
	}
	out := matching(path.Clean(glob), folders)
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: packages glob %q matches no folder of tracked files; fix the glob or remove it", FileName, glob)
	}
	return out, nil
}

// matching is the folders a cleaned glob matches, each `*` within one
// folder name, or the root alone for ".".
func matching(glob string, folders []string) []string {
	if glob == "." {
		return []string{"."}
	}
	var out []string
	for _, folder := range folders {
		if ok, _ := path.Match(glob, folder); ok {
			out = append(out, folder)
		}
	}
	return out
}

// foldersOf is every folder that holds one of files, at any depth, below
// the repo root, sorted.
func foldersOf(files []string) []string {
	seen := map[string]bool{}
	for _, file := range files {
		addFolders(seen, file)
	}
	return slices.Sorted(maps.Keys(seen))
}

// addFolders adds each folder above file to seen. It stops at one already
// there, since that one's own folders are there too.
func addFolders(seen map[string]bool, file string) {
	for dir := path.Dir(file); dir != "." && dir != "/" && !seen[dir]; dir = path.Dir(dir) {
		seen[dir] = true
	}
}
