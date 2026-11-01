package check

import (
	"errors"
	"os/exec"
	"regexp"
	"strings"
)

// history is what past commits say about requirement IDs.
type history struct {
	// removed holds every ID a commit took out entirely.
	removed map[string]bool
}

// historyLine matches an added or removed title or acceptance line in a diff.
var historyLine = regexp.MustCompile(`^([+-])(?:# (\d{4}) |- (\d{4}\.\d+)\s*$)`)

// readHistory replays every commit that touched docsDir, oldest first, and
// notes why git failed so the gap reports as info.
func readHistory(root string) (h history, note string) {
	h = history{removed: map[string]bool{}}
	out, err := exec.Command("git", "-C", root, "log", "--reverse", "--no-color", "--no-ext-diff",
		"-U0", "--format=%x00", "-p", "--", docsDir).Output()
	if err != nil {
		if why := gitFailure(err); !noHistory.MatchString(why) {
			return h, "requirement IDs were not checked against git history: " + why
		}
		return h, ""
	}
	live := map[string]int{}
	for _, commit := range strings.Split(string(out), "\x00") {
		h.apply(live, commitNet(commit))
	}
	return h, ""
}

// noHistory matches git's complaint outside a repo or before the first commit,
// where there is no history to read.
var noHistory = regexp.MustCompile(`not a git repository|does not have any commits`)

// gitFailure names why a git command failed: its error text, else the Go
// error.
func gitFailure(err error) string {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if msg := strings.Join(strings.Fields(string(exitErr.Stderr)), " "); msg != "" {
			return msg
		}
	}
	return err.Error()
}

// commitNet counts each ID's added minus removed lines in one commit, so an
// edited or moved line nets to zero.
func commitNet(diff string) map[string]int {
	net := map[string]int{}
	for _, line := range strings.Split(diff, "\n") {
		if m := historyLine.FindStringSubmatch(line); m != nil {
			net[m[2]+m[3]] += map[string]int{"+": 1, "-": -1}[m[1]]
		}
	}
	return net
}

// apply folds one commit's net changes into the live counts, recording IDs
// whose last copy disappears.
func (h history) apply(live, net map[string]int) {
	for id, n := range net {
		before := live[id]
		live[id] += n
		if before > 0 && live[id] <= 0 {
			h.removed[id] = true
		}
	}
}
