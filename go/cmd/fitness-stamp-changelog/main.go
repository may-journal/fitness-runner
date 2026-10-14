// Command fitness-stamp-changelog is the pre-commit stamping tool: when a
// new changelog/*.md entry file is staged, or else CHANGELOG.md, it
// restamps the first "### yyyy.mm.dd.HHMM" heading to the current local
// time and re-stages the file, so entries are written with any placeholder
// stamp and the hook makes them current. A branch whose earlier commit added
// its entry keeps that entry's time. The npm-era version bumping is gone with the npm packages: the
// changelog timestamp is the version now.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/changelog"
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
		fmt.Fprintln(os.Stderr, "fitness-stamp-changelog: restamped the changelog entry heading")
	}
}

// stamp restamps and re-stages when CHANGELOG.md is staged; reports whether
// it wrote anything.
func stamp(root string, now time.Time) (bool, error) {
	changed := false
	for _, name := range stampTargets(root) {
		wrote, err := stampFile(root, name, formatTimestamp(now))
		if err != nil {
			return changed, err
		}
		changed = changed || wrote
	}
	return changed, nil
}

// stampTargets are the files whose newest heading this commit stamps: each
// new entry file, else a staged CHANGELOG.md, and none when the branch
// already has its entry of either kind.
func stampTargets(root string) []string {
	if len(changelog.BranchEntryFiles(root)) > 0 || len(changelog.BranchEntries(root)) > 0 {
		return nil
	}
	if files := changelog.StagedEntryFiles(root); len(files) > 0 {
		return files
	}
	if changelogStaged(root) {
		return []string{changelogName}
	}
	return nil
}

// stampFile restamps name's first heading to ts and re-stages it; reports
// whether it wrote anything.
func stampFile(root, name, ts string) (bool, error) {
	path := filepath.Join(root, filepath.FromSlash(name))
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	next := restamp(string(raw), ts)
	if next == string(raw) {
		return false, nil
	}
	return true, writeAndRestage(root, path, name, next)
}

// writeAndRestage writes the restamped file and re-stages it so the
// in-flight commit picks up the new heading.
func writeAndRestage(root, path, name, next string) error {
	if err := os.WriteFile(path, []byte(next), 0o644); err != nil {
		return err
	}
	if out, err := exec.Command("git", "-C", root, "add", name).CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %v: %s", err, out)
	}
	return nil
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
