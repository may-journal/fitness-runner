package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// The hook folders a repo can ship, newest first. The org sync ships
// .fitness-hooks; `fitness init` wrote .githooks before it existed.
const (
	syncedHooksDir    = ".fitness-hooks"
	installedHooksDir = ".githooks"
)

// shippedHooksDir names the hook folder HEAD tracks, newest first, or "" when
// it tracks neither or git fails.
func shippedHooksDir(root string) string {
	for _, dir := range []string{syncedHooksDir, installedHooksDir} {
		tracked, err := gitx.Repo(root).TracksDir(dir)
		if err != nil {
			return ""
		}
		if tracked {
			return dir
		}
	}
	return ""
}

// repairHooksPath points a stale, set core.hooksPath at the folder HEAD ships,
// reporting on stderr to keep hook output unchanged.
func repairHooksPath() {
	root, _ := os.Getwd()
	current := hooksPath(root)
	if current == "" {
		return
	}
	shipped := shippedHooksDir(root)
	if shipped == "" {
		return
	}
	if note := repairedPath(root, current, shipped); note != "" {
		fmt.Fprintln(os.Stderr, "fitness hook: "+note)
	}
}

// repairedPath sets core.hooksPath to shipped when current names another
// folder inside the repo, and says what it did. A path outside the repo is
// kept and named; the shipped path itself needs nothing and says nothing.
func repairedPath(root, current, shipped string) string {
	rel, inside := insideRepo(root, current)
	if !inside {
		return "core.hooksPath " + current + " is outside the repo, so it is left as is"
	}
	if rel == shipped {
		return ""
	}
	if err := setHooksPath(root, shipped); err != nil {
		return err.Error()
	}
	return "core.hooksPath was " + current + ", now " + shipped
}

// setHooksPath points git at dir in root under the git deadline, failing
// with what git said.
func setHooksPath(root, dir string) error {
	return gitx.Repo(root).SetConfig("core.hooksPath", dir)
}

// hooksPath reads core.hooksPath as git resolves it, under the git deadline.
// It is "" when unset, or when the read fails or stalls, so a stalled git
// leaves the path alone.
func hooksPath(root string) string {
	path, _ := gitx.Repo(root).Config("core.hooksPath")
	return path
}

// insideRepo returns path relative to root and whether it lies inside root.
func insideRepo(root, path string) (string, bool) {
	rel, err := filepath.Rel(root, resolvePath(root, path))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

// resolvePath resolves a core.hooksPath as git does: a leading ~/ is the home
// folder, and a relative path starts at the working-tree root.
func resolvePath(root, path string) string {
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, rest)
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	return path
}
