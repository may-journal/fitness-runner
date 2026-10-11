package gitx

import "regexp"

var planRe = regexp.MustCompile(`(?i)plan #\s*(\d+)`)

// PlanNumbers extracts distinct Plan trailers in first-seen order.
func PlanNumbers(text string) []string {
	seen := map[string]bool{}
	var nums []string
	for _, m := range planRe.FindAllStringSubmatch(text, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			nums = append(nums, m[1])
		}
	}
	return nums
}
