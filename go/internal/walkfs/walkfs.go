// Package walkfs walks a repo for files by extension, pruning the directory
// names every fitness check skips — the port of the TypeScript
// getSkipDirsForWalk + findFilesByExtension pair.
package walkfs

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// runnerSkipDirs are always pruned, at any depth, by basename.
var runnerSkipDirs = []string{"node_modules", "dist", "coverage", ".git", "githooks"}

// SkipDirs returns the walk's prune set for root: the runner defaults plus
// every <root>/cspell.json ignorePaths entry that is a bare name (no '/' or
// '*'). Any read/parse error contributes nothing, matching the TS behavior.
func SkipDirs(root string) map[string]bool {
	set := make(map[string]bool, len(runnerSkipDirs))
	for _, d := range runnerSkipDirs {
		set[d] = true
	}
	raw, err := os.ReadFile(filepath.Join(root, "cspell.json"))
	if err != nil {
		return set
	}
	var spell struct {
		IgnorePaths []string `json:"ignorePaths"`
	}
	if err := json.Unmarshal(raw, &spell); err != nil {
		return set
	}
	for _, p := range spell.IgnorePaths {
		if !strings.ContainsAny(p, "/*") {
			set[p] = true
		}
	}
	return set
}

// FilesByExt returns the sorted slash-separated relative paths of files
// under root whose name ends in any of exts (suffix match, not glob — ".md"
// matches "x.custom.md" too). Directories in the skip set are pruned by
// basename at any depth. Unreadable subtrees are skipped, never fatal.
func FilesByExt(root string, exts ...string) []string {
	skip := SkipDirs(root)
	var out []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != root && skip[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		for _, ext := range exts {
			if strings.HasSuffix(d.Name(), ext) {
				rel, relErr := filepath.Rel(root, path)
				if relErr != nil {
					return nil
				}
				out = append(out, filepath.ToSlash(rel))
				return nil
			}
		}
		return nil
	})
	sort.Strings(out)
	return out
}
