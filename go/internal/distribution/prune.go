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

// stagedSet matches a check set folder a killed install left half made.
var stagedSet = regexp.MustCompile(`^check-set-[0-9]+$`)

// pruneRetry is how long a prune that found GitHub silent waits before it
// probes again, so offline hooks stay fast.
const pruneRetry = time.Hour

// pruneOfflineName records a prune that found GitHub silent.
const pruneOfflineName = "cache-prune-offline.json"

// asidePrefix starts the name of a folder a prune set aside to delete.
const asidePrefix = ".pruning-"

// asideName matches a folder a prune set aside to delete.
var asideName = regexp.MustCompile(`^\.pruning-`)

// used marks path as used now, so a prune keeps it, taking path back when a
// prune set it aside meanwhile. A cache it may not change stays readable.
func used(path string) {
	now := time.Now()
	if os.Chtimes(path, now, now) == nil {
		return
	}
	if os.Rename(aside(path), path) == nil {
		_ = os.Chtimes(path, now, now)
	}
}

// aside is the name path takes while a prune deletes it.
func aside(path string) string {
	return filepath.Join(filepath.Dir(path), asidePrefix+filepath.Base(path))
}

// cachedProgram marks a check program's release as used and reports whether
// the cache holds the program.
func cachedProgram(program string) bool {
	used(filepath.Dir(program))
	return exists(program)
}

// bundleOf is the cache folder of the bundle bin, a bundle-N folder, and
// whether bin is in one.
func bundleOf(bin string) (string, bool) {
	key := filepath.Dir(bin)
	return key, bundleKey.MatchString(filepath.Base(key))
}

// MarkUsed marks the cache bundle holding program as used, so a prune keeps
// one a person runs by its path.
func MarkUsed(program string) {
	if key, ok := bundleOf(filepath.Dir(program)); ok {
		used(key)
	}
}

// Prune deletes unused and superseded releases from cache once a day while
// online, keeping inUse. online probes GitHub when no read answered lately.
func Prune(cache string, inUse []string, online func() bool) []string {
	if !dueForPrune(cache, online) {
		return nil
	}
	recordFresh(cache, prunedName, true)
	root, err := filepath.EvalSymlinks(cache)
	if err != nil {
		return nil
	}
	for _, dir := range []string{root, filepath.Join(root, checksFolder), filepath.Join(root, checkSetsFolder)} {
		sweepAside(dir)
	}
	keep := append(realPaths(inUse), pointedInto(root, folders(root, bundleKey))...)
	p := pruner{cutoff: time.Now().Add(-unusedFor), keep: keep}
	p.pruneSets(filepath.Join(root, checkSetsFolder))
	p.prune(filepath.Join(root, checksFolder), nil)
	p.prune(root, bundleKey)
	p.prune(root, stagedSet)
	return append(p.removed, pruneSuperseded(root, keep)...)
}

// dueForPrune reports whether a day passed since the last prune and GitHub
// answers. Offline, a deleted check could not return.
func dueForPrune(cache string, online func() bool) bool {
	if _, pruned := readFresh[bool](cache, prunedName, pruneEvery); pruned {
		return false
	}
	if answeredLately(cache) {
		return true
	}
	if _, waiting := readFresh[bool](cache, pruneOfflineName, pruneRetry); waiting {
		return false
	}
	if online() {
		return true
	}
	recordFresh(cache, pruneOfflineName, true)
	return false
}

// answeredLately reports whether GitHub answered an index read in the last
// 10 minutes, so no probe is needed.
func answeredLately(cache string) bool {
	_, answered := readFresh[checkIndex](cache, checkIndexName, latestFresh)
	_, unanswered := readFresh[bool](cache, unansweredName, unansweredFresh)
	return answered && !unanswered
}

// sweepAside settles each folder an earlier prune set aside in dir and left.
// One whose name is free comes back for this prune to judge; a run remade the
// rest, so they go.
func sweepAside(dir string) {
	for _, path := range folders(dir, asideName) {
		name := filepath.Join(dir, strings.TrimPrefix(filepath.Base(path), asidePrefix))
		if exists(name) || os.Rename(path, name) != nil {
			_ = os.RemoveAll(path)
		}
	}
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
		if !p.removeSet(set) {
			p.keep = append(p.keep, linked(set)...)
		}
	}
}

// prune deletes each unused folder in dir whose name matches names, or any
// name for nil.
func (p *pruner) prune(dir string, names *regexp.Regexp) {
	p.pruneAll(folders(dir, names))
}

// pruneAll deletes each unused folder of paths.
func (p *pruner) pruneAll(paths []string) {
	for _, path := range paths {
		p.remove(path)
	}
}

// removeSet deletes check set when unused, with its pins file, and reports
// whether it did.
func (p *pruner) removeSet(set string) bool {
	if !p.remove(set) {
		return false
	}
	_ = os.Remove(set + pinsSuffix)
	return true
}

// remove deletes path when unused, and reports whether it did. It sets path
// aside first; a run marking it meanwhile takes it back, which the second look
// sees, so that run keeps it.
func (p *pruner) remove(path string) bool {
	if !p.unused(path) {
		return false
	}
	set := aside(path)
	if os.Rename(path, set) != nil {
		return false
	}
	if !p.stale(set) {
		_ = os.Rename(set, path)
		return false
	}
	_ = os.RemoveAll(set)
	p.removed = append(p.removed, path)
	return true
}

// unused reports whether path is stale and holds no kept path.
func (p *pruner) unused(path string) bool {
	if !p.stale(path) {
		return false
	}
	return !slices.ContainsFunc(p.keep, func(kept string) bool { return within(kept, []string{path}) })
}

// stale reports whether path was last used before cutoff. A folder whose
// time cannot be read counts as used.
func (p *pruner) stale(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.ModTime().Before(p.cutoff)
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
