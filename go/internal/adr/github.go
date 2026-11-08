package adr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

// Label is the label every `ADR` issue carries.
const Label = "ADR"

// GitHub opens and reads a repo's `ADR` issues through the gh CLI.
type GitHub struct {
	// Repo is the owner/name the issues live in.
	Repo string
}

// gh runs the gh CLI with args, feeding it stdin, and returns its output.
func gh(stdin string, args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	cmd.Stdin = strings.NewReader(stdin)
	o := checkkit.Exec(cmd)
	if o.Err != nil {
		return "", fmt.Errorf("gh %s: %v: %s", args[0], o.Err, strings.TrimSpace(o.Stderr))
	}
	return strings.TrimSpace(o.Stdout), nil
}

// CurrentRepo is the owner/name of the repo gh sees in the working folder.
func CurrentRepo() (string, error) {
	return gh("", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
}

// Issue is one `ADR` issue: its URL, body and state, OPEN or CLOSED.
type Issue struct{ URL, Body, State string }

// Migrated maps each source file an `ADR` issue's marker names to the
// issue, open or closed.
func (g GitHub) Migrated() (map[string]Issue, error) {
	out, err := gh("", "issue", "list", "--repo", g.Repo, "--label", Label, "--state", "all", "--limit", "1000",
		"--json", "url,body,state", "--jq", ".[] | {url, body, state}")
	if err != nil {
		return nil, err
	}
	found := map[string]Issue{}
	dec := json.NewDecoder(bytes.NewReader([]byte(out)))
	for dec.More() {
		var issue Issue
		if err := dec.Decode(&issue); err != nil {
			return nil, fmt.Errorf("gh issue list printed no issue: %w", err)
		}
		if src := SourceOf(issue.Body); src != "" {
			found[src] = issue
		}
	}
	return found, nil
}

// EnsureLabel creates the `ADR` label, or updates it when it exists.
func (g GitHub) EnsureLabel() error {
	_, err := gh("", "label", "create", Label, "--repo", g.Repo, "--color", "1d76db",
		"--description", "An architecture decision", "--force")
	return err
}

// Open opens an `ADR` issue and returns its URL.
func (g GitHub) Open(title, body string) (string, error) {
	return gh(body, "issue", "create", "--repo", g.Repo, "--title", title, "--label", Label, "--body-file", "-")
}

// Edit replaces the body of the issue at url.
func (g GitHub) Edit(url, body string) error {
	_, err := gh(body, "issue", "edit", url, "--body-file", "-")
	return err
}

// Close closes the issue at url as not planned, with comment.
func (g GitHub) Close(url, comment string) error {
	_, err := gh("", "issue", "close", url, "--reason", "not planned", "--comment", comment)
	return err
}
