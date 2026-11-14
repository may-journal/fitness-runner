package repocheck

import (
	"fmt"
	"regexp"
	"strconv"
)

// release matches a release version, major.minor.patch, with an optional
// leading v.
var release = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)$`)

// NeedsNewer says why the running fitness is too old for a check's minFitness,
// or "" when new enough or unversioned.
func NeedsNewer(check, minimum, running string) string {
	if minimum == "" || running == "" {
		return ""
	}
	want, ok := parts(minimum)
	if !ok {
		return fmt.Sprintf("check %s declares minFitness %q, which is no major.minor.patch version; fix its --describe", check, minimum)
	}
	if !olderThan(running, want) {
		return ""
	}
	return fmt.Sprintf("check %s needs fitness %s or later, and this is fitness %s; upgrade fitness, such as by rerunning fitness-install with --version=%s", check, minimum, running, minimum)
}

// parts reads a version's three numbers.
func parts(v string) ([3]int, bool) {
	m := release.FindStringSubmatch(v)
	var out [3]int
	if m == nil {
		return out, false
	}
	for i := range out {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// olderThan reports whether the running version comes before want; one
// that is no release version never does.
func olderThan(running string, want [3]int) bool {
	have, ok := parts(running)
	return ok && older(have, want)
}

// older reports whether have comes before want.
func older(have, want [3]int) bool {
	for i := range have {
		if have[i] != want[i] {
			return have[i] < want[i]
		}
	}
	return false
}
