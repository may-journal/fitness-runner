package release

import (
	"regexp"
	"strings"
)

var versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// Numeric components are compared by length first to avoid integer overflow.
func compareVersions(a, b string) int {
	left, right := strings.Split(a, "."), strings.Split(b, ".")
	for i := range left {
		if len(left[i]) != len(right[i]) {
			return len(left[i]) - len(right[i])
		}
		if order := strings.Compare(left[i], right[i]); order != 0 {
			return order
		}
	}
	return 0
}
