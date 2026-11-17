package distribution

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
)

// pointersName records the Claude Code settings files and skills folders an
// install pointed at a cache bundle, which each prune reads again.
const pointersName = "cache-pointers.json"

// maxSettingsBytes bounds how much of a settings file a prune reads.
const maxSettingsBytes = 1 << 20

// rememberPointers records that each of paths points into the cache bundle
// bin, so a prune keeps the bundle while one still does.
func rememberPointers(bin string, paths []string) {
	key, ok := bundleOf(bin)
	if !ok {
		return
	}
	cache := filepath.Dir(key)
	known := readPointers(cache)
	for _, path := range realPaths(paths) {
		if !slices.Contains(known, path) {
			known = append(known, path)
		}
	}
	writePointers(cache, known)
}

// readPointers lists the paths cache records as pointing into its bundles.
func readPointers(cache string) []string {
	var known []string
	data, _ := os.ReadFile(filepath.Join(cache, pointersName))
	_ = json.Unmarshal(data, &known)
	return known
}

// writePointers records known in cache. A cache it may not change keeps its
// record.
func writePointers(cache string, known []string) {
	if data, err := json.Marshal(known); err == nil {
		_ = writeAtomicMode(filepath.Join(cache, pointersName), data, 0600)
	}
}

// pointedInto lists the bundles recorded settings files and skills folders
// still point into, and forgets each that points into none.
func pointedInto(root string, bundles []string) []string {
	known := readPointers(root)
	var live, pointed []string
	for _, pointer := range known {
		if into := pointsInto(pointer, bundles); len(into) > 0 {
			live, pointed = append(live, pointer), append(pointed, into...)
		}
	}
	if len(live) < len(known) {
		writePointers(root, live)
	}
	return pointed
}

// pointsInto lists the bundles a skills folder's links lead into, or a
// settings file names.
func pointsInto(pointer string, bundles []string) []string {
	info, err := os.Stat(pointer)
	if err != nil {
		return nil
	}
	if info.IsDir() {
		return linkedBundles(pointer, bundles)
	}
	return namedBundles(pointer, bundles)
}

// linkedBundles lists the bundles the links in folder lead into.
func linkedBundles(folder string, bundles []string) []string {
	entries, _ := os.ReadDir(folder)
	var into []string
	for _, entry := range entries {
		target, err := filepath.EvalSymlinks(filepath.Join(folder, entry.Name()))
		for _, bundle := range bundles {
			if err == nil && within(target, []string{bundle}) {
				into = append(into, bundle)
			}
		}
	}
	return into
}

// namedBundles lists the bundles whose folder name file holds.
func namedBundles(file string, bundles []string) []string {
	f, err := os.Open(file)
	if err != nil {
		return nil
	}
	defer f.Close()
	data, _ := io.ReadAll(io.LimitReader(f, maxSettingsBytes))
	return slices.DeleteFunc(slices.Clone(bundles), func(bundle string) bool {
		return !bytes.Contains(data, []byte(filepath.Base(bundle)))
	})
}
