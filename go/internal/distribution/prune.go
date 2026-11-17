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

// supersededFor is how long a prune keeps a superseded release unused, so
// one a pinned repo ran within a day stays.
const supersededFor = 24 * time.Hour

// pinsSuffix ends the file beside a check set that records each linked
// check's pin.
const pinsSuffix = ".pins"

// checkKey matches the cache folder of one check release and platform,
// capturing its program, major, minor and patch, and platform:
// fitness-check-<name>-v<major>.<minor>.<patch>-<platform>.tar.
var checkKey = regexp.MustCompile(`^(fitness-check-.+)-v([0-9]+)\.([0-9]+\.[0-9]+)-(.+)\.tar$`)

// bundleVersion matches an engine bundle folder, capturing its version and
// platform.
var bundleVersion = regexp.MustCompile(`^fitness-([0-9]+\.[0-9]+\.[0-9]+)-(.+)-[0-9a-f]{64}$`)

// pruneEvery is how often the cache is pruned, so hooks stay fast.
const pruneEvery = 24 * time.Hour

// prunedName records when the cache was last pruned.
const prunedName = "cache-pruned.json"

// bundleKey matches the cache folder of one engine release and platform:
// fitness-<version>-<platform>-<sha256>.
var bundleKey = regexp.MustCompile(`^fitness-.+-[0-9a-f]{64}$`)

// stagedSet matches a check set folder a killed install left half made.
var stagedSet = regexp.MustCompile(`^check-set-[0-9]+$`)

// linkedMarker marks a bundle folder that Claude Code settings or linked
// skills point into, so a prune never deletes it.
const linkedMarker = "linked"

// used marks path as used now, so a prune keeps it. Each run marks what it
// uses before it reads it; a cache it may not change stays readable.
func used(path string) {
	now := time.Now()
	_ = os.Chtimes(path, now, now)
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

// keepLinked marks the cache bundle bin as one that settings or skills
// outside the cache point at.
func keepLinked(bin string) {
	if key, ok := bundleOf(bin); ok {
		_ = os.WriteFile(filepath.Join(key, linkedMarker), nil, 0600)
	}
}

// Prune deletes what cache left unused for 30 days, and superseded releases,
// once a day while online, and returns it. It keeps what inUse holds and
// follows no link out of cache.
func Prune(cache string, inUse []string) []string {
	if !dueForPrune(cache) {
		return nil
	}
	recordFresh(cache, prunedName, true)
	root, err := filepath.EvalSymlinks(cache)
	if err != nil {
		return nil
	}
	keep := realPaths(inUse)
	p := pruner{cutoff: time.Now().Add(-unusedFor), keep: keep}
	p.pruneSets(filepath.Join(root, checkSetsFolder))
	p.prune(filepath.Join(root, checksFolder), nil)
	p.prune(root, bundleKey)
	p.prune(root, stagedSet)
	return append(p.removed, pruneSuperseded(root, keep)...)
}

// pruneSuperseded deletes each release, unused for a day, that a newer cached
// one supersedes: same check and major, or same engine platform.
// It keeps what keep holds or links, and each release a check set pins exactly.
func pruneSuperseded(root string, keep []string) []string {
	checks := filepath.Join(root, checksFolder)
	kept := slices.Clone(keep)
	for _, path := range keep {
		kept = append(kept, linked(path)...)
	}
	kept = append(kept, exactlyPinned(checks, folders(filepath.Join(root, checkSetsFolder), nil))...)
	p := pruner{cutoff: time.Now().Add(-supersededFor), keep: kept}
	p.pruneAll(superseded(folders(checks, checkKey), releaseVersion))
	p.pruneAll(superseded(folders(root, bundleKey), engineVersion))
	return p.removed
}

// releaseVersion is the group of a check release folder, its check, major and
// platform, its version, and whether it holds the program.
func releaseVersion(path string) (group, version string, cached bool) {
	m := checkKey.FindStringSubmatch(filepath.Base(path))
	if m == nil {
		return "", "", false
	}
	return m[1] + " " + m[2] + " " + m[4], m[2] + "." + m[3], exists(filepath.Join(path, m[1]))
}

// engineVersion is the group of an engine bundle folder, its platform, its
// version, and whether it holds an unpacked bundle.
func engineVersion(path string) (group, version string, cached bool) {
	m := bundleVersion.FindStringSubmatch(filepath.Base(path))
	if m == nil {
		return "", "", false
	}
	bundles, _ := filepath.Glob(filepath.Join(path, "bundle-*"))
	return m[2], m[1], len(bundles) > 0
}

// superseded lists each path whose group holds a newer cached version.
func superseded(paths []string, version func(string) (string, string, bool)) []string {
	newest := newestCached(paths, version)
	return slices.DeleteFunc(slices.Clone(paths), func(path string) bool {
		group, v, _ := version(path)
		return group == "" || !newer(newest[group], v)
	})
}

// newestCached maps each group of paths to its newest cached version.
func newestCached(paths []string, version func(string) (string, string, bool)) map[string]string {
	newest := map[string]string{}
	for _, path := range paths {
		if group, v, cached := version(path); group != "" && cached && newer(v, newest[group]) {
			newest[group] = v
		}
	}
	return newest
}

// exactlyPinned lists each release folder in checks that one of sets
// records an exact pin of, such as v1.2.3, on any platform.
func exactlyPinned(checks string, sets []string) []string {
	prefixes := exactPrefixes(sets)
	return slices.DeleteFunc(folders(checks, checkKey), func(release string) bool {
		return !slices.ContainsFunc(prefixes, func(prefix string) bool {
			return strings.HasPrefix(filepath.Base(release), prefix)
		})
	})
}

// exactPrefixes lists the folder name start of each release that one of
// sets pins exactly.
func exactPrefixes(sets []string) []string {
	var prefixes []string
	for _, set := range sets {
		for _, line := range setPins(set) {
			if name, pin, _ := strings.Cut(line, " "); exactPin.MatchString(pin) {
				prefixes = append(prefixes, checkPrefix+name+"-"+pin+"-")
			}
		}
	}
	return prefixes
}

// dueForPrune reports whether a day passed since the last prune and GitHub
// just answered the index read. Offline, a deleted check could not return.
func dueForPrune(cache string) bool {
	_, pruned := readFresh[bool](cache, prunedName, pruneEvery)
	index, online := readFresh[checkIndex](cache, checkIndexName, latestFresh)
	_, unanswered := readFresh[bool](cache, unansweredName, unansweredFresh)
	return !pruned && online && index.Format == 1 && !unanswered
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
		if p.remove(set) {
			_ = os.Remove(set + pinsSuffix)
		} else {
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

// remove deletes path when unused, and reports whether it did. It renames
// path aside and checks its time again, so a run marking it meanwhile keeps it.
func (p *pruner) remove(path string) bool {
	if !p.unused(path) {
		return false
	}
	aside := filepath.Join(filepath.Dir(path), ".pruning-"+filepath.Base(path))
	if os.Rename(path, aside) != nil {
		return false
	}
	if !p.stale(aside) {
		_ = os.Rename(aside, path)
		return false
	}
	_ = os.RemoveAll(aside)
	p.removed = append(p.removed, path)
	return true
}

// unused reports whether path is stale and holds no kept path or marker.
func (p *pruner) unused(path string) bool {
	if !p.stale(path) || exists(filepath.Join(path, linkedMarker)) {
		return false
	}
	return !slices.ContainsFunc(p.keep, func(kept string) bool {
		return kept == path || strings.HasPrefix(kept, path+string(filepath.Separator))
	})
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
