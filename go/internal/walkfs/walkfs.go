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

	"github.com/may-journal/fitness-runner/go/internal/par"
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
	addCspellBareIgnores(root, set)
	return set
}

// addCspellBareIgnores adds every <root>/cspell.json ignorePaths entry that
// is a bare name (no '/' or '*') to set. Any read/parse error contributes
// nothing, matching the TS behavior.
func addCspellBareIgnores(root string, set map[string]bool) {
	raw, err := os.ReadFile(filepath.Join(root, "cspell.json"))
	if err != nil {
		return
	}
	var spell struct {
		IgnorePaths []string `json:"ignorePaths"`
	}
	if err := json.Unmarshal(raw, &spell); err != nil {
		return
	}
	for _, p := range spell.IgnorePaths {
		if !strings.ContainsAny(p, "/*") {
			set[p] = true
		}
	}
}

// ScanFiles runs scan over every file under root matching exts and returns
// the collected error messages plus the file count — the read-loop shared by
// the file-scanning checks. The first unreadable file aborts with its error.
func ScanFiles(root string, exts []string, scan func(relPath, content string) []string) ([]string, int, error) {
	files := FilesByExt(root, exts...)
	results := par.Map(len(files), 0, func(i int) scanResult {
		return scanOne(root, files[i], scan)
	})
	var errs []string
	for _, r := range results {
		if r.err != nil {
			return nil, 0, r.err
		}
		errs = append(errs, r.errs...)
	}
	return errs, len(files), nil
}

// scanResult carries one file's scan outcome through the worker pool.
type scanResult struct {
	errs []string
	err  error
}

// scanOne reads and scans a single file for the parallel ScanFiles pool.
func scanOne(root, file string, scan func(relPath, content string) []string) scanResult {
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil {
		return scanResult{err: err}
	}
	return scanResult{errs: scan(file, string(content))}
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
			return pruneDir(root, path, d.Name(), skip)
		}
		if hasAnySuffix(d.Name(), exts) {
			if rel, ok := relSlash(root, path); ok {
				out = append(out, rel)
			}
		}
		return nil
	})
	sort.Strings(out)
	return out
}

// pruneDir returns filepath.SkipDir when the walker should prune the
// directory at path (its basename is in the skip set and it is not the walk
// root itself), else nil to descend.
func pruneDir(root, path, name string, skip map[string]bool) error {
	if path != root && skip[name] {
		return filepath.SkipDir
	}
	return nil
}

// hasAnySuffix reports whether name ends in any of exts (suffix match, not
// glob).
func hasAnySuffix(name string, exts []string) bool {
	for _, ext := range exts {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

// relSlash converts path to its slash-separated form relative to root; ok is
// false when path cannot be made relative.
func relSlash(root, path string) (string, bool) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", false
	}
	return filepath.ToSlash(rel), true
}
