package distribution

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// The cache folders that hold check programs, one folder per release, and
// the check sets that link them.
const (
	checksFolder    = "checks"
	checkSetsFolder = "check-sets"
)

// unusedFor is how long a cached engine bundle, check release or check set
// goes unused before a prune deletes it.
const unusedFor = 30 * 24 * time.Hour

// pruneEvery is how often the cache is pruned, so hooks stay fast.
const pruneEvery = 24 * time.Hour

// prunedName records when the cache was last pruned.
const prunedName = "cache-pruned.json"

// bundleKey matches the cache folder of one engine release and platform:
// fitness-<version>-<platform>-<sha256>.
var bundleKey = regexp.MustCompile(`^fitness-.+-[0-9a-f]{64}$`)

// used marks path as used now, so a prune keeps it, and reports whether it
// exists. Each run marks what it uses before it reads it.
func used(path string) bool {
	now := time.Now()
	return os.Chtimes(path, now, now) == nil
}

// cachedProgram marks a check program's release as used and reports whether
// the cache holds the program.
func cachedProgram(program string) bool {
	return used(filepath.Dir(program)) && exists(program)
}

// Prune deletes what cache left unused for 30 days, once a day, and returns
// it. It keeps what inUse holds and follows no link out of cache.
func Prune(cache string, inUse []string) []string {
	if _, ok := readFresh[bool](cache, prunedName, pruneEvery); ok || !exists(cache) {
		return nil
	}
	recordFresh(cache, prunedName, true)
	root, err := filepath.EvalSymlinks(cache)
	if err != nil {
		return nil
	}
	p := pruner{cutoff: time.Now().Add(-unusedFor), keep: realPaths(inUse)}
	p.pruneSets(filepath.Join(root, checkSetsFolder))
	p.prune(filepath.Join(root, checksFolder), nil)
	p.prune(root, bundleKey)
	return p.removed
}

// pruner deletes folders unused since cutoff that hold no kept path.
type pruner struct {
	cutoff  time.Time
	keep    []string
	removed []string
}

// pruneSets deletes each unused check set, and keeps the releases each
// other set links, so a set in use never loses a program.
func (p *pruner) pruneSets(dir string) {
	for _, set := range folders(dir, nil) {
		if !p.remove(set) {
			p.keep = append(p.keep, linked(set)...)
		}
	}
}

// prune deletes each unused folder in dir whose name matches names, or any
// name for nil.
func (p *pruner) prune(dir string, names *regexp.Regexp) {
	for _, path := range folders(dir, names) {
		p.remove(path)
	}
}

// remove deletes path when it is unused, and reports whether it did.
func (p *pruner) remove(path string) bool {
	if !p.unused(path) || os.RemoveAll(path) != nil {
		return false
	}
	p.removed = append(p.removed, path)
	return true
}

// unused reports whether path was last used before cutoff and holds no kept
// path. A folder whose time cannot be read counts as used.
func (p *pruner) unused(path string) bool {
	info, err := os.Lstat(path)
	if err != nil || !info.ModTime().Before(p.cutoff) {
		return false
	}
	return !slices.ContainsFunc(p.keep, func(kept string) bool {
		return kept == path || strings.HasPrefix(kept, path+string(filepath.Separator))
	})
}

// folders lists the folders directly in dir whose names match names, or
// every one for nil. Links are left out, so nothing outside dir is listed.
func folders(dir string, names *regexp.Regexp) []string {
	entries, _ := os.ReadDir(dir)
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() && (names == nil || names.MatchString(entry.Name())) {
			paths = append(paths, filepath.Join(dir, entry.Name()))
		}
	}
	return paths
}

// linked lists the real path of each program check set links.
func linked(set string) []string {
	entries, _ := os.ReadDir(set)
	var paths []string
	for _, entry := range entries {
		paths = append(paths, filepath.Join(set, entry.Name()))
	}
	return realPaths(paths)
}

// realPaths resolves each path's links, keeping one that cannot be resolved
// as it is.
func realPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if real, err := filepath.EvalSymlinks(path); err == nil {
			path = real
		}
		out = append(out, filepath.Clean(path))
	}
	return out
}
