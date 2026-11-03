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

// dirFlag judges only the files under one folder.
//
//fitness:usage --dir=<folder>
const dirFlag = "--dir"

// fileFlag judges only the named file; repeat it to name more.
//
//fitness:usage --file=<path>
const fileFlag = "--file"

// runScope pulls `--dir=<folder>` or each `--file=<path>` out of argv and
// lists the repo files they name. A run without either returns nil files.
func runScope(root string, argv []string) (files, rest []string, err error) {
	named, rest := splitFlag(argv, fileFlag)
	if len(named) == 0 {
		return dirScope(root, rest)
	}
	if dir, _ := splitFlag(rest, dirFlag); len(dir) > 0 {
		return nil, nil, errors.New("use " + dirFlag + " or " + fileFlag + ", not both")
	}
	files, err = namedFiles(root, named)
	return files, rest, err
}

// namedFiles lists the repo files each `--file=<path>` names, skipping
// ignored ones. None left is an error, since an empty scope would judge the
// whole repo.
func namedFiles(root string, flags []string) ([]string, error) {
	var files []string
	for _, flag := range flags {
		listed, err := namedFile(root, strings.TrimPrefix(flag, fileFlag+"="))
		if err != nil {
			return nil, err
		}
		files = append(files, listed...)
	}
	if len(files) == 0 {
		return nil, errors.New("--file names no file git tracks or would add")
	}
	return files, nil
}

// literalPathspec makes git read a path as written, not as a glob, so a
// file named app/[id].jsx names only itself.
const literalPathspec = ":(literal)"

// namedFile lists path when git tracks it or would add it.
func namedFile(root, path string) ([]string, error) {
	if path == "" || path == fileFlag {
		return nil, errors.New("use " + fileFlag + "=<path>")
	}
	rel, err := fileInRepo(root, path)
	if err != nil {
		return nil, err
	}
	listed, err := gitx.Repo(root).FilesUnder(literalPathspec + rel)
	if err != nil {
		return nil, fmt.Errorf("--file %s: %w", path, err)
	}
	return listed, nil
}

// fileInRepo returns path as a slash path relative to root, or an error
// when it is not an existing file inside the repo.
func fileInRepo(root, path string) (string, error) {
	abs := path
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, path)
	}
	if !isFile(abs) {
		return "", errors.New("--file " + path + " is not a file")
	}
	rel, err := filepath.Rel(realPath(root), realPath(abs))
	if err != nil || escapesRoot(rel) {
		return "", errors.New("--file " + path + " is outside the repo")
	}
	return filepath.ToSlash(rel), nil
}

// isFile reports whether path names an existing file that is no folder.
func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// dirScope pulls `--dir=<folder>` out of argv and lists the non-ignored repo
// files under it. A run without the flag returns nil files.
func dirScope(root string, argv []string) (files, rest []string, err error) {
	dir, rest, err := oneValueFlag(argv, dirFlag, "<folder>")
	if err != nil || dir == "" {
		return nil, rest, err
	}
	files, err = folderFiles(root, dir)
	return files, rest, err
}

// folderFiles lists the files under dir, relative to root. A missing, outside or
// empty folder is an error, since an empty scope would judge the whole repo.
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
