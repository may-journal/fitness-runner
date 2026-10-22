package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// dirFlag names the one folder a run judges.
const dirFlag = "--dir"

// dirScope pulls `--dir=<folder>` out of argv and lists the repo files under
// it, tracked or new but not ignored, so neither check selection nor a single
// check's passthrough sees the flag. A run without the flag returns nil files.
func dirScope(root string, argv []string) (files, rest []string, err error) {
	flags, rest := splitDirFlags(argv)
	if len(flags) == 0 {
		return nil, rest, nil
	}
	dir := strings.TrimPrefix(flags[0], dirFlag+"=")
	if len(flags) > 1 || dir == "" || dir == dirFlag {
		return nil, nil, errors.New("use --dir=<folder> once")
	}
	files, err = folderFiles(root, dir)
	return files, rest, err
}

// splitDirFlags separates the `--dir` arguments from the rest of argv.
func splitDirFlags(argv []string) (flags, rest []string) {
	for _, a := range argv {
		if a == dirFlag || strings.HasPrefix(a, dirFlag+"=") {
			flags = append(flags, a)
		} else {
			rest = append(rest, a)
		}
	}
	return flags, rest
}

// folderFiles lists the files under dir, relative to root. A folder that is
// missing, outside the repo, or holds no files is an error: an empty scope
// would otherwise read as no scope and judge the whole repo.
func folderFiles(root, dir string) ([]string, error) {
	rel, err := folderInRepo(root, dir)
	if err != nil {
		return nil, err
	}
	files := gitx.FilesUnder(root, rel)
	if len(files) == 0 {
		return nil, errors.New("--dir " + dir + " holds no files to check")
	}
	return files, nil
}

// folderInRepo returns dir as a slash path relative to root, or an error
// when it is not an existing folder inside the repo.
func folderInRepo(root, dir string) (string, error) {
	abs := dir
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, dir)
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil || escapesRoot(rel) {
		return "", errors.New("--dir " + dir + " is outside the repo")
	}
	if !isFolder(abs) {
		return "", errors.New("--dir " + dir + " is not a folder")
	}
	return filepath.ToSlash(rel), nil
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
