package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"slices"
	"strings"
)

// npmRegistry is where every dev package of a passing bump resolves.
const npmRegistry = "https://registry.npmjs.org/"

// npmSide is package.json and package-lock.json at one commit.
type npmSide struct {
	manifest map[string]any
	// devOrder is the manifest's dev dependencies in file order.
	devOrder []string
	lock     map[string]any
}

// readNPMSide decodes the manifest and lock files of one commit.
func readNPMSide(manifest, lock []byte) (npmSide, error) {
	var side npmSide
	if err := json.Unmarshal(manifest, &side.manifest); err != nil {
		return side, fmt.Errorf("package.json: %w", err)
	}
	if err := json.Unmarshal(lock, &side.lock); err != nil {
		return side, fmt.Errorf("package-lock.json: %w", err)
	}
	var raw struct {
		Dev json.RawMessage `json:"devDependencies"`
	}
	_ = json.Unmarshal(manifest, &raw)
	side.devOrder = objectKeys(raw.Dev)
	return side, nil
}

// objectKeys lists a JSON object's keys in file order, or none for anything
// that is not an object.
func objectKeys(raw json.RawMessage) []string {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if tok, err := decoder.Token(); err != nil || tok != json.Delim('{') {
		return nil
	}
	var keys []string
	for decoder.More() {
		key, ok := nextKey(decoder)
		if !ok {
			break
		}
		keys = append(keys, key)
	}
	return keys
}

// nextKey reads one key of an object and skips its value.
func nextKey(decoder *json.Decoder) (string, bool) {
	tok, err := decoder.Token()
	key, isKey := tok.(string)
	var value json.RawMessage
	return key, err == nil && isKey && decoder.Decode(&value) == nil
}

// object is v as a JSON object, or an empty one when v is null or false,
// as jq's `// {}` reads it.
func object(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

// without is m without the keys drop, or m itself when it is not an object.
func without(m any, drop ...string) any {
	obj, ok := m.(map[string]any)
	if !ok {
		return m
	}
	out := maps.Clone(obj)
	for _, k := range drop {
		delete(out, k)
	}
	return out
}

// versionLine matches a version's major and, when given, minor number.
var versionLine = regexp.MustCompile(`^[^0-9]*([0-9]+)(\.([0-9]+))?`)

// line is the release line of version: its major, or 0.minor for a 0.x
// version, which semver lets break on a minor bump. It is "" for a value
// that names no version.
func line(version any) string {
	s, _ := version.(string)
	m := versionLine.FindStringSubmatch(s)
	switch {
	case m == nil:
		return ""
	case m[1] != "0":
		return m[1]
	case m[2] == "":
		return "0.null"
	}
	return "0." + m[3]
}

// text is v as jq interpolates it into a string.
func text(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	raw, _ := json.Marshal(v)
	return string(raw)
}

// when is problem as a one-item list when bad holds, or none.
func when(bad bool, problem string) []string {
	if bad {
		return []string{problem}
	}
	return nil
}

// bump is a problem when from and to are not on one release line.
func bump(name string, from, to any) []string {
	return when(line(from) == "" || line(from) != line(to),
		fmt.Sprintf("%s moves from %s to %s, not a patch or minor bump", name, text(from), text(to)))
}

// bumpProblems lists why base to head is more than a patch or minor bump of
// dev packages.
func bumpProblems(base, head npmSide) []string {
	return append(manifestProblems(base, head), lockProblems(base.lock, head.lock)...)
}

// manifestProblems lists what package.json changes beyond dev bumps.
func manifestProblems(base, head npmSide) []string {
	bd, hd := object(base.manifest["devDependencies"]), object(head.manifest["devDependencies"])
	problems := when(!reflect.DeepEqual(without(base.manifest, "devDependencies"), without(head.manifest, "devDependencies")), "package.json changes more than devDependencies")
	problems = append(problems, when(!slices.Equal(slices.Sorted(maps.Keys(bd)), slices.Sorted(maps.Keys(hd))), "package.json adds or removes a dev dependency")...)
	for _, name := range base.devOrder {
		if to, ok := hd[name]; ok && !reflect.DeepEqual(bd[name], to) {
			problems = append(problems, bump(name, bd[name], to)...)
		}
	}
	return problems
}

// lockProblems lists what package-lock.json changes beyond dev bumps.
func lockProblems(base, head map[string]any) []string {
	problems := when(!reflect.DeepEqual(lockHeader(base), lockHeader(head)), "package-lock.json changes more than its packages")
	bp, hp := object(base["packages"]), object(head["packages"])
	problems = append(problems, when(!reflect.DeepEqual(without(bp[""], "devDependencies"), without(hp[""], "devDependencies")), "package-lock.json changes the root package")...)
	names := maps.Clone(bp)
	maps.Copy(names, hp)
	for _, name := range slices.Sorted(maps.Keys(names)) {
		if name != "" && !reflect.DeepEqual(bp[name], hp[name]) {
			problems = append(problems, packageProblems(name, bp[name], hp[name])...)
		}
	}
	return problems
}

// lockHeader is a lock file without its package lists.
func lockHeader(lock map[string]any) any {
	if _, ok := lock["packages"]; ok {
		return without(lock, "packages", "dependencies")
	}
	return lock
}

// packageProblems says why a locked package's change is more than a dev
// patch or minor bump. A nil side lacks the package.
func packageProblems(name string, from, to any) []string {
	switch {
	case !devOnly(from, to):
		return []string{name + " is not a dev-only package"}
	case outsideRegistry(to):
		return []string{name + " resolves outside the npm registry"}
	case !bothPresent(from, to):
		return nil
	case reflect.DeepEqual(object(from)["version"], object(to)["version"]):
		return []string{name + " changes without a new version"}
	}
	return bump(name, object(from)["version"], object(to)["version"])
}

// devOnly reports whether every present side is a dev package.
func devOnly(sides ...any) bool {
	for _, side := range sides {
		if side != nil && object(side)["dev"] != true {
			return false
		}
	}
	return true
}

// outsideRegistry reports whether the head side resolves outside npm's
// registry. Only a missing resolved field defaults to the registry.
func outsideRegistry(to any) bool {
	if to == nil {
		return false
	}
	resolved, ok := object(to)["resolved"].(string)
	return ok && !strings.HasPrefix(resolved, npmRegistry)
}

// bothPresent reports whether the package is on both sides.
func bothPresent(from, to any) bool {
	return from != nil && to != nil
}
