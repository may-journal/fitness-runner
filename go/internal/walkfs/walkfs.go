// Package walkfs lists a repo's files by extension for the fitness checks.
// Inside a git repo it lists tracked files still on disk; outside one it walks
// every file except .git. No ignore list removes a file.
package walkfs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/par"
)

// InScope narrows files to the run's changed files, keeping all of them when
// the run is unscoped.
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

// ScanFiles runs scan over every in-scope file under root matching exts,
// skipping and naming as info any file it cannot read.
func ScanFiles(root string, exts []string, scan func(relPath, content string) []string) checkkit.Result {
	errs, read, info := scanAll(root, exts, scan)
	result := checkkit.Pass(read)
	if len(errs) > 0 {
		result = checkkit.Fail(read, errs...)
	}
	result.Info = info
	return result
}

// ScanFindings is ScanFiles for a check that reports findings: line renders
// each as the error line older readers of the result see.
func ScanFindings(root string, exts []string, line func(checkkit.Finding) string, scan func(relPath, content string) []checkkit.Finding) checkkit.Result {
	found, read, info := scanAll(root, exts, scan)
	result := checkkit.Found(read, line, found...)
	result.Info = info
	return result
}

// scanAll scans in-scope files matching exts in parallel, returning findings in
// file order, the read count, and a note per unreadable file.
func scanAll[T any](root string, exts []string, scan func(relPath, content string) []T) (found []T, read int, info []string) {
	files := InScope(FilesByExt(root, exts...))
	results := par.Map(len(files), 0, func(i int) scanResult[T] {
		return scanOne(root, files[i], scan)
	})
	for _, r := range results {
		if r.skipped != "" {
			info = append(info, r.skipped)
			continue
		}
		read++
		found = append(found, r.found...)
	}
	return found, read, info
}

// scanResult carries one file's scan outcome through the worker pool.
type scanResult[T any] struct {
	found []T
	// skipped names a file that could not be read, empty once read.
	skipped string
}

// Unreadable is the info note for a file a check skipped, naming its relative
// path and why reading failed.
func Unreadable(rel string, err error) string {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}
	return fmt.Sprintf("%s: skipped, cannot be read (%v)", rel, err)
}

// scanOne reads and scans a single file for the parallel ScanFiles pool.
func scanOne[T any](root, file string, scan func(relPath, content string) []T) scanResult[T] {
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil {
		return scanResult[T]{skipped: Unreadable(file, err)}
	}
	return scanResult[T]{found: scan(file, string(content))}
}

// FilesByExt returns the sorted relative paths under root whose name ends in
// any of exts, where "" matches every file and ".md" matches "x.custom.md".
func FilesByExt(root string, exts ...string) []string {
	list := trackedFiles(root)
	if !list.ok {
		list = trackedList{files: walkedFiles(root), onDisk: true}
	}
	var out []string
	for _, rel := range list.files {
		if wanted(root, rel, exts, !list.onDisk) {
			out = append(out, rel)
		}
	}
	sort.Strings(out)
	return out
}

// AllFiles returns every sorted path under root, erroring when listing fails so
// a check never passes on an unread list.
func AllFiles(root string) ([]string, error) {
	if list := trackedFiles(root); list.ok {
		out := listedOnDisk(root, list)
		sort.Strings(out)
		return out, nil
	}
	if insideWorkTree(root) {
		return nil, errors.New("git cannot list the tracked files; run `git ls-files` to see why")
	}
	files, err := walkFiles(root, true)
	sort.Strings(files)
	return files, err
}

// listedOnDisk returns a copy of the list's regular files on disk, stating them
// only when the list has not been checked.
func listedOnDisk(root string, list trackedList) []string {
	if list.onDisk {
		return slices.Clone(list.files)
	}
	return onDisk(root, list.files)
}

// insideWorkTree reports whether git sees root as inside a work tree.
func insideWorkTree(root string) bool {
	out, err := exec.Command("git", "-C", root, "rev-parse", "--is-inside-work-tree").Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

// wanted reports whether rel ends in one of exts and, if stat is set, is a
// regular file; the name test runs first.
func wanted(root, rel string, exts []string, stat bool) bool {
	return hasAnySuffix(filepath.Base(rel), exts) && (!stat || RegularFile(root, rel))
}

// RegularFile reports whether rel under root is a regular file on disk, so a
// deleted path, submodule directory or symlink is not.
func RegularFile(root, rel string) bool {
	info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil && info.Mode().IsRegular()
}

// walkedFiles walks root outside a git repo, skipping symlinks, .git
// directories, and unreadable subtrees.
func walkedFiles(root string) []string {
	files, _ := walkFiles(root, false)
	return files
}

// walkFiles walks root as walkedFiles says. When strict, the first
// unreadable entry stops the walk and is returned; otherwise it is skipped.
func walkFiles(root string, strict bool) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return keptIf(strict, err)
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
	return files, err
}

// keptIf returns err when strict, so the walk stops on it, and nil
// otherwise, so the walk skips the entry.
func keptIf(strict bool, err error) error {
	if strict {
		return err
	}
	return nil
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
