package distribution

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// supersededFor is how long a prune keeps a superseded release unused, so
// one a pinned repo ran within a day stays.
const supersededFor = 24 * time.Hour

// pinsSuffix ends the file beside a check set that records each linked
// check's pin.
const pinsSuffix = ".pins"

// requestedMarker marks an engine bundle that fitness-install was asked for
// by its exact version, which no newer bundle supersedes.
const requestedMarker = "requested"

// checkKey matches the cache folder of one check release and platform,
// capturing its program, major, minor and patch, and platform:
// fitness-check-<name>-v<major>.<minor>.<patch>-<platform>.tar.
var checkKey = regexp.MustCompile(`^(fitness-check-.+)-v([0-9]+)\.([0-9]+\.[0-9]+)-(.+)\.tar$`)

// bundleVersion matches an engine bundle folder, capturing its version and
// platform.
var bundleVersion = regexp.MustCompile(`^fitness-([0-9]+\.[0-9]+\.[0-9]+)-(.+)-[0-9a-f]{64}$`)

// markRequested marks the engine bundle key as asked for by exact version.
func markRequested(key string) {
	_ = os.WriteFile(filepath.Join(key, requestedMarker), nil, 0600)
}

// pruneSuperseded deletes each release, unused for a day, that a newer cached
// one supersedes: same check and major, or same engine platform.
func pruneSuperseded(root string, keep []string) []string {
	checks := filepath.Join(root, checksFolder)
	sets := folders(filepath.Join(root, checkSetsFolder), nil)
	p := pruner{cutoff: time.Now().Add(-supersededFor)}
	p.keep = slices.Concat(keep, linkedBy(keep), exactlyPinned(checks, sets), requested(root))
	p.keep = append(p.keep, p.liveLinks(sets)...)
	doomed := slices.DeleteFunc(superseded(folders(checks, checkKey), releaseVersion), func(release string) bool {
		return !p.unused(release)
	})
	p.pruneDangling(sets, doomed)
	p.pruneAll(doomed)
	p.pruneAll(superseded(folders(root, bundleKey), engineVersion))
	return p.removed
}

// linkedBy lists what each of paths links, when it is a check set.
func linkedBy(paths []string) []string {
	var links []string
	for _, path := range paths {
		links = append(links, linked(path)...)
	}
	return links
}

// liveLinks lists what each of sets used since cutoff links, so a set a
// run used lately keeps every program.
func (p *pruner) liveLinks(sets []string) []string {
	var links []string
	for _, set := range sets {
		if !p.stale(set) {
			links = append(links, linked(set)...)
		}
	}
	return links
}

// pruneDangling deletes each unused set that links one of doomed, so no set
// is left linking a deleted release.
func (p *pruner) pruneDangling(sets, doomed []string) {
	for _, set := range sets {
		if slices.ContainsFunc(linked(set), func(link string) bool { return within(link, doomed) }) {
			p.removeSet(set)
		}
	}
}

// within reports whether path is one of folders or inside one.
func within(path string, folders []string) bool {
	return slices.ContainsFunc(folders, func(folder string) bool {
		return path == folder || strings.HasPrefix(path, folder+string(filepath.Separator))
	})
}

// requested lists each engine bundle in root fitness-install was asked for
// by its exact version.
func requested(root string) []string {
	return slices.DeleteFunc(folders(root, bundleKey), func(key string) bool {
		return !exists(filepath.Join(key, requestedMarker))
	})
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
