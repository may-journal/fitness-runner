// Package goal holds the coverage goal every coverage check judges by,
// whichever language it measures: every counted line, branch or statement
// covered. A check compares exact counts, so a total that rounds to the goal
// still falls short.
//
//fitness:topic coverage
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
