// Package walkfs lists a repo's files by extension for the fitness checks.
// Inside a git repo it lists every tracked file still on disk, so only
// untracked files stay out; outside one it walks every file except .git.
// No directory name, config key, or tool ignore list removes a file.
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

// ScanFiles runs scan over every in-scope file under root matching exts —
// the read-loop shared by the file-scanning checks — and returns the check's
// Result: failed with the collected messages, else passed, counting the
// files read. A file that cannot be read is skipped and named as info, so
// the other files are still judged.
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

// scanAll scans every in-scope file matching exts in parallel, returning
// what scan found in file order, how many files it read, and an info note
// for each file it could not read.
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

// Unreadable is the info note for a file a check skipped because reading
// it failed, naming the file by its relative path and the reason, such as
// permission denied.
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

// FilesByExt returns the sorted slash-separated relative paths of files
// under root whose name ends in any of exts (suffix match, not glob — ".md"
// matches "x.custom.md" too; "" matches every file). Inside a git repo the
// candidates are the tracked files that still exist on disk; outside one
// they are every file except those under .git.
func FilesByExt(root string, exts ...string) []string {
	candidates, tracked := trackedFiles(root)
	if !tracked {
		candidates = walkedFiles(root)
	}
	var out []string
	for _, rel := range candidates {
		if wanted(root, rel, exts, tracked) {
			out = append(out, rel)
		}
	}
	sort.Strings(out)
	return out
}

// AllFiles returns every path under root a check judges by name, sorted and
// slash-separated. Inside a git work tree it is every path in the index, so a
// staged addition or a rename's target counts before its commit; outside one
// it is every regular file except those under .git. Unlike FilesByExt, it
// returns an error when git or the walk cannot list the files, so a check
// never passes on a list it could not read.
func AllFiles(root string) ([]string, error) {
	if files, ok := trackedFiles(root); ok {
		out := slices.Clone(files)
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

// insideWorkTree reports whether git sees root as inside a work tree.
func insideWorkTree(root string) bool {
	out, err := exec.Command("git", "-C", root, "rev-parse", "--is-inside-work-tree").Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

// wanted reports whether rel ends in one of exts and, when it came from
// git, is still a regular file on disk. The cheap name test runs first, so
// only a matching tracked path costs an lstat; a walked path is already a
// regular file.
func wanted(root, rel string, exts []string, tracked bool) bool {
	return hasAnySuffix(filepath.Base(rel), exts) && (!tracked || isRegularFile(root, rel))
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
