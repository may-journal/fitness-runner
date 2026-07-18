// Command fitness-stamp-changelog is the pre-commit stamping tool: when
// CHANGELOG.md is staged, it restamps the first "### yyyy.mm.dd.HHMM"
// heading to the current local time and re-stages the file, so entries are
// written with any placeholder stamp and the hook makes them current. The
// npm-era version bumping is gone with the npm packages: the changelog
// timestamp is the version now.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const changelogName = "CHANGELOG.md"

var headingRe = regexp.MustCompile(`(?m)^(### )\d{4}\.\d{2}\.\d{2}\.\d{4}`)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	changed, err := stamp(root, time.Now())
	if err != nil {
		fmt.Fprintln(os.Stderr, "fitness-stamp-changelog:", err)
		os.Exit(1)
	}
	if changed {
		fmt.Fprintln(os.Stderr, "fitness-stamp-changelog: restamped CHANGELOG.md heading")
	}
}

// stamp restamps and re-stages when CHANGELOG.md is staged; reports whether
// it wrote anything.
func stamp(root string, now time.Time) (bool, error) {
	if !changelogStaged(root) {
		return false, nil
	}
	path := filepath.Join(root, changelogName)
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	next := restamp(string(raw), formatTimestamp(now))
	if next == string(raw) {
		return false, nil
	}
	if err := os.WriteFile(path, []byte(next), 0o644); err != nil {
		return false, err
	}
	if out, err := exec.Command("git", "-C", root, "add", changelogName).CombinedOutput(); err != nil {
		return false, fmt.Errorf("git add: %v: %s", err, out)
	}
	return true, nil
}

// changelogStaged reports whether CHANGELOG.md is in the staged file list.
func changelogStaged(root string) bool {
	out, err := exec.Command("git", "-C", root, "diff", "--cached", "--name-only").Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == changelogName {
			return true
		}
	}
	return false
}

// restamp replaces the FIRST heading timestamp only, like the node script's
// non-global regex replace.
func restamp(content, ts string) string {
	done := false
	return headingRe.ReplaceAllStringFunc(content, func(m string) string {
		if done {
			return m
		}
		done = true
		return headingRe.ReplaceAllString(m, "${1}"+ts)
	})
}

// formatTimestamp renders local time as yyyy.mm.dd.HHMM.
func formatTimestamp(t time.Time) string {
	return t.Format("2006.01.02.1504")
}
