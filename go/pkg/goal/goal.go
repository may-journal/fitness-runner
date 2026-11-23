// Package goal holds the coverage goal, as go/docs/coverage.md describes.
package goal

import "strconv"

// Percent is the coverage goal, as a percentage.
const Percent = 100

// Met reports whether covered of total meets the goal. Nothing to cover
// meets it.
func Met[N ~int | ~int64](covered, total N) bool {
	return covered*100 >= total*Percent
}

// Text is the goal as findings print it, such as "100%".
func Text() string {
	return strconv.Itoa(Percent) + "%"
}
