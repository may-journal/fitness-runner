package check

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// change is one file whose lines changed beyond whitespace.
type change struct {
	path    string
	deleted bool
}

// scope is what a run diffs: the arguments that give `git diff` the change,
// and, for a pull request, the merge base it starts from.
type scope struct {
	args []string
	base string
}

// diffScope is the change a run judges: a pull request's diff from its merge
// base with the base branch, an amend's staged files since the amended
// commit's parent, or else the staged files.
func diffScope(root string) (scope, error) {
	if branch := os.Getenv("GITHUB_BASE_REF"); branch != "" {
		base, err := gitOut(root, "merge-base", "HEAD", "origin/"+branch)
		return scope{args: []string{base, "HEAD"}, base: base}, err
	}
	if gitx.Amending() {
		return scope{args: []string{"--cached", gitx.AmendParent(root)}}, nil
	}
	return scope{args: []string{"--cached"}}, nil
}

// readChanges lists the files the diff changes beyond whitespace. Git drops
// a reformatted file when told to ignore whitespace and blank lines, and a
// pure rename, with status R100, changes no line, so both drop out.
func readChanges(root string, args []string) ([]change, error) {
	out, err := gitOut(root, append([]string{"diff", "-z", "--name-status", "-M", "-w", "--ignore-blank-lines"}, args...)...)
	if err != nil {
		return nil, err
	}
	return parseStatus(out), nil
}

// parseStatus reads `git diff -z --name-status -M`: a status, then one path,
// or two for a rename or copy, whose new path is the one that counts.
func parseStatus(out string) []change {
	fields := strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")
	var changes []change
	for i := 0; i+1 < len(fields); i += 2 {
		status, c := fields[i], change{path: fields[i+1], deleted: fields[i] == "D"}
		if strings.ContainsAny(status[:1], "RC") && i+2 < len(fields) {
			i++
			c.path = fields[i+1]
		}
		if status != "R100" {
			changes = append(changes, c)
		}
	}
	return changes
}

// gitOut runs git in root and returns its trimmed output, or an internal
// error naming what git said.
func gitOut(root string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	out, err := cmd.Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return "", checkkit.Internal(errors.New("git " + args[0] + ": " + strings.TrimSpace(string(exit.Stderr))))
	}
	if err != nil {
		return "", checkkit.Internal(err)
	}
	return strings.TrimSpace(string(out)), nil
}
