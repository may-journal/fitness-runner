package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// dirFlag names the one folder a run judges.
const dirFlag = "--dir"

// dirScope pulls `--dir=<folder>` out of argv and lists the repo files under
// it, tracked or new but not ignored, so neither check selection nor a single
// check's passthrough sees the flag. A run without the flag returns nil files.
func dirScope(root string, argv []string) (files, rest []string, err error) {
	dir, rest, err := oneValueFlag(argv, dirFlag, "<folder>")
	if err != nil || dir == "" {
		return nil, rest, err
	}
	files, err = folderFiles(root, dir)
	return files, rest, err
}

// folderFiles lists the files under dir, relative to root. A folder that is
// missing, outside the repo, or holds no files is an error: an empty scope
// would otherwise read as no scope and judge the whole repo.
func folderFiles(root, dir string) ([]string, error) {
	rel, err := folderInRepo(root, dir)
	if err != nil {
		return nil, err
	}
	files, err := gitx.Repo(root).FilesUnder(rel)
	if err != nil {
		return nil, fmt.Errorf("--dir %s: listing its files: %w", dir, err)
	}
	if len(files) == 0 {
		return nil, errors.New("--dir " + dir + " holds no files to check")
	}
	return files, nil
}

// folderInRepo returns dir as a slash path relative to root, or an error
// when it is not an existing folder inside the repo. Both paths resolve
// symlinks first, so a folder reached through a linked path still counts.
func folderInRepo(root, dir string) (string, error) {
	abs := dir
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, dir)
	}
	if !isFolder(abs) {
		return "", errors.New("--dir " + dir + " is not a folder")
	}
	rel, err := filepath.Rel(realPath(root), realPath(abs))
	if err != nil || escapesRoot(rel) {
		return "", errors.New("--dir " + dir + " is outside the repo")
	}
	return filepath.ToSlash(rel), nil
}

// realPath returns path with its symlinks resolved, or path itself when
// they cannot be.
func realPath(path string) string {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		return real
	}
	return path
}

// escapesRoot reports whether a path relative to the root leaves it.
func escapesRoot(rel string) bool {
	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// isFolder reports whether path names an existing folder.
func isFolder(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// shareScope writes the `--dir` folder's files to a file the checks read
// through checkkit.ScopeFileEnv, returning that env entry and a cleanup. A
// run without the flag shares nothing.
func shareScope(files []string) (env []string, cleanup func(), err error) {
	cleanup = func() {}
	if len(files) == 0 {
		return nil, cleanup, nil
	}
	f, err := os.CreateTemp("", "fitness-scope-*")
	if err != nil {
		return nil, cleanup, err
	}
	remove := func() { _ = os.Remove(f.Name()) }
	_, writeErr := f.WriteString(strings.Join(files, "\n"))
	if err := errors.Join(writeErr, f.Close()); err != nil {
		remove()
		return nil, cleanup, err
	}
	return []string{checkkit.ScopeFileEnv + "=" + f.Name()}, remove, nil
}
