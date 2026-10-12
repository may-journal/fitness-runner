package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// trailerKey is the commit trailer that exempts requirement numbers from
// pairing, with the reason after them.
const trailerKey = "Unpaired-Requirement"

var (
	// trailerValue is one requirement number or a comma-separated list, then
	// a reason, so a reason may start with a four-digit word.
	trailerValue = regexp.MustCompile(`^(\d{4}(?:\s*,\s*\d{4})*)\s+(\S.*)$`)
	// wordy holds a letter, so a bare second number is no reason.
	wordy = regexp.MustCompile(`\pL`)
)

// exemption is one trailer: its line as written, and the numbers it exempts.
type exemption struct {
	line    string
	numbers []string
}

// exemptions returns the exempt requirement numbers: the trailers of the
// commit being written, or of a pull request's commits once its body lists
// them. Each malformed or unlisted trailer is an error.
func exemptions(root string) (map[string]bool, []string) {
	if msg, ok := checkkit.CtxMessage(); ok {
		found, errs := trailers(root, msg)
		return exemptSet(found), errs
	}
	base := os.Getenv("GITHUB_BASE_REF")
	if base == "" {
		return nil, nil
	}
	var found []exemption
	var errs []string
	for _, c := range gitx.BranchCommits(root, "HEAD", "origin/"+base) {
		more, bad := trailers(root, gitx.Message(root, c))
		found, errs = append(found, more...), append(errs, bad...)
	}
	listed, unlisted := splitListed(found, prBody())
	return exemptSet(listed), append(errs, unlisted...)
}

// trailers parses message's exemption trailers the way git reads them.
func trailers(root, message string) ([]exemption, []string) {
	cmd := exec.Command("git", "-C", root, "interpret-trailers", "--parse")
	cmd.Stdin = strings.NewReader(message)
	out, _ := cmd.Output()
	var found []exemption
	var errs []string
	for _, line := range strings.Split(string(out), "\n") {
		key, value, ok := strings.Cut(line, ":")
		if ok && strings.EqualFold(strings.TrimSpace(key), trailerKey) {
			e, err := parseExemption(line, strings.TrimSpace(value))
			found, errs = append(found, e...), append(errs, err...)
		}
	}
	return found, errs
}

// parseExemption reads one trailer's value: numbers, then a reason with at
// least one word.
func parseExemption(line, value string) ([]exemption, []string) {
	m := trailerValue.FindStringSubmatch(value)
	if m == nil || !wordy.MatchString(m[2]) {
		return nil, []string{fmt.Sprintf("%q needs requirement numbers, then the reason they change alone: `%s: NNNN, NNNN <reason>`", line, trailerKey)}
	}
	return []exemption{{line: trailerKey + ": " + value, numbers: strings.FieldsFunc(m[1], notDigit)}}, nil
}

// notDigit separates the numbers in a trailer.
func notDigit(r rune) bool {
	return r < '0' || r > '9'
}

// splitListed keeps the exemptions body lists and flags the rest, so no
// exemption reaches the base branch unseen.
func splitListed(found []exemption, body string) ([]exemption, []string) {
	var listed []exemption
	var errs []string
	flat := strings.Join(strings.Fields(body), " ")
	for _, e := range found {
		if strings.Contains(flat, strings.Join(strings.Fields(e.line), " ")) {
			listed = append(listed, e)
			continue
		}
		errs = append(errs, fmt.Sprintf("%q is not listed in the pull request body; list each exemption there so reviewers see it, then push or reopen the pull request to run CI on the new body", e.line))
	}
	return listed, errs
}

// exemptSet is every number the exemptions name.
func exemptSet(found []exemption) map[string]bool {
	set := map[string]bool{}
	for _, e := range found {
		for _, n := range e.numbers {
			set[n] = true
		}
	}
	return set
}

// prBody reads the pull request body from the GitHub Actions event payload;
// empty when there is none.
func prBody() string {
	data, err := os.ReadFile(os.Getenv("GITHUB_EVENT_PATH"))
	if err != nil {
		return ""
	}
	var ev struct {
		PullRequest struct {
			Body string `json:"body"`
		} `json:"pull_request"`
	}
	_ = json.Unmarshal(data, &ev)
	return ev.PullRequest.Body
}
