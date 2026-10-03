// Package walkfs lists a repo's files by extension for the fitness checks.
// Inside a git repo it lists every tracked file still on disk, so only
// untracked files stay out; outside one it walks every file except .git.
// No directory name, config key, or tool ignore list removes a file.
package walkfs

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/par"
)

// InScope narrows files to the run's changed files, keeping all of them when
// the run is unscoped. A check that judges each file on its own passes its
// walk through InScope; one that compares files, or only detects whether a
// file type exists, keeps the full walk.
func InScope(files []string) []string {
	changed := checkkit.ChangedFiles()
	if changed == nil {
		return files
	}
	want := make(map[string]bool, len(changed))
	for _, f := range changed {
		want[f] = true
	}
	var out []string
	for _, f := range files {
		if want[f] {
			out = append(out, f)
		}
	}
	return out
}

// ScanFiles runs scan over every in-scope file under root matching exts and
// returns the collected error messages plus the file count — the read-loop
// shared by the file-scanning checks. The first unreadable file aborts with
// its error.
func ScanFiles(root string, exts []string, scan func(relPath, content string) []string) ([]string, int, error) {
	files := InScope(FilesByExt(root, exts...))
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
// matches "x.custom.md" too; "" matches every file). Inside a git repo the
// candidates are the tracked files that still exist on disk; outside one
// they are every file except those under .git.
func FilesByExt(root string, exts ...string) []string {
	all, ok := trackedFiles(root)
	if !ok {
		all = walkedFiles(root)
	}
	var out []string
	for _, rel := range all {
		if hasAnySuffix(filepath.Base(rel), exts) {
			out = append(out, rel)
		}
	}
	sort.Strings(out)
	return out
}

// trackedFiles lists root's tracked files that exist on disk as regular
// files, relative to root; ok is false when root is not inside a git work
// tree (or git is missing), so the caller walks the disk instead.
func trackedFiles(root string) ([]string, bool) {
	out, err := exec.Command("git", "-C", root, "ls-files", "-z", "--cached").Output()
	if err != nil {
		return nil, false
	}
	var files []string
	for _, rel := range bytes.Split(out, []byte{0}) {
		if len(rel) > 0 && isRegularFile(root, string(rel)) {
			files = append(files, string(rel))
		}
	}
	return files, true
}

// isRegularFile reports whether the slash-separated rel under root is a
// regular file on disk: a tracked path deleted from the work tree, a
// submodule directory, or a symlink is not. A symlink's tracked content is
// its target path, and the target is judged as its own file.
func isRegularFile(root, rel string) bool {
	info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil && info.Mode().IsRegular()
}

// walkedFiles walks root outside a git repo, collecting every file's
// slash-separated relative path, skipping symlinks and pruning only .git
// directories.
// Unreadable subtrees are skipped, never fatal.
func walkedFiles(root string) []string {
	var files []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return pruneGit(root, path, d.Name())
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if rel, ok := relSlash(root, path); ok {
			files = append(files, rel)
		}
		return nil
	})
	return files
}

// pruneGit returns filepath.SkipDir for a .git directory below root, the
// one place a walk outside git never looks; every other directory descends.
func pruneGit(root, path, name string) error {
	if path != root && name == ".git" {
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
