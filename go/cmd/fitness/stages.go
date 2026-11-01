package main

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/report"
)

// changedFilesEnv is the env variable that lists a scoped run's files.
const changedFilesEnv = "FITNESS_CHANGED_FILES"

// pullRequestChanges is the files a pull request changes when its whole
// suite runs with a diff, so it takes two stages, else nil.
func pullRequestChanges(s setup, argv, env []string) []string {
	if !wholeSuitePullRequest(s, argv) {
		return nil
	}
	for _, kv := range env {
		if list, ok := strings.CutPrefix(kv, changedFilesEnv+"="); ok {
			return strings.Split(list, "\n")
		}
	}
	return nil
}

// wholeSuitePullRequest reports whether the run is a pull request's whole
// suite: no named check, --all or --dir.
func wholeSuitePullRequest(s setup, argv []string) bool {
	spec, _ := parseArgv(argv)
	return os.Getenv("GITHUB_BASE_REF") != "" && spec == "" && !hasAllFlag(argv) && len(s.scope) == 0
}

// runStages judges the changed files, then scans every file, so a pull
// request fails while the repo does. scopedPullRequests skips the scan.
func runStages(s setup, checks []resolved, env, changed []string) int {
	scoped := fmt.Sprintf("checking the %d files this pull request changes", len(changed))
	if scopedPullRequests(s.root) {
		announce("Stage 1 of 1: " + scoped)
		code := runStage(s.root, s.perfPath, checks, nil, env, s.cleanup, true).code
		if code == 0 {
			announce("Stage 2 skipped: .fitnessrc.json sets scopedPullRequests, so only a push to main scans every file")
		}
		return code
	}
	announce("Stage 1 of 2: " + scoped)
	if code := runStage(s.root, s.perfPath, checks, nil, env, s.cleanup, false).code; code != 0 {
		announce("Stage 2 skipped: stage 1 failed, so fix its findings first")
		return code
	}
	announce("Stage 2 of 2: the changes pass, so every tracked file is scanned")
	full := runStage(s.root, s.perfPath, checks, nil, withoutVar(env, changedFilesEnv), s.cleanup, true)
	if full.code != 0 {
		announce("Stage 2 failed", whereFound(full, changed)...)
	}
	return full.code
}

// scopedPullRequests reports whether the repo config opts out of stage 2.
// An unreadable config reads as unset, since selection already failed on it.
func scopedPullRequests(root string) bool {
	cfg, err := loadConfig(root)
	return err == nil && cfg != nil && cfg.ScopedPullRequests
}

// announce prints a stage's headline and any detail lines on stderr. In
// GitHub Actions the job summary shows the headline as a heading and the
// lines under it.
func announce(headline string, lines ...string) {
	fmt.Fprintln(os.Stderr, strings.Join(append([]string{headline}, lines...), "\n"))
	if !isGitHubActions() {
		return
	}
	md := "## " + report.EscapeMarkdown(headline) + "\n\n"
	for _, line := range lines {
		if item, ok := strings.CutPrefix(line, "- "); ok {
			md += "- " + report.EscapeMarkdown(item) + "\n"
		} else {
			md += "\n" + report.EscapeMarkdown(line) + "\n\n"
		}
	}
	writeStepSummary(md)
}

// whereFound lists where stage 2 failed: unchanged files, then changed
// files, then the checks whose findings name no file.
func whereFound(r stageResult, changed []string) []string {
	p := places{outside: map[string][]string{}, inside: map[string][]string{}, nowhere: map[string][]string{}}
	for i, row := range r.rows {
		if !row.Ok {
			for _, f := range orUndiagnosed(row.Name, r.found[i]) {
				p.add(row.Name, report.FindingFile(f), changed)
			}
		}
	}
	lines := foundList("Files this pull request does not change:", p.outside)
	lines = append(lines, foundList("Files this pull request changes, on lines stage 1 did not judge:", p.inside)...)
	return append(lines, foundList("Checks whose findings name no file:", p.nowhere)...)
}

// places maps each file where stage 2 failed to its failing checks, in
// whereFound's three groups; the last group is keyed by check.
type places struct {
	outside, inside, nowhere map[string][]string
}

// add files one finding of check in file, "" for none, into its group.
func (p places) add(check, file string, changed []string) {
	switch {
	case file == "":
		p.nowhere[check] = nil
	case slices.Contains(changed, file):
		p.inside[file] = appendOnce(p.inside[file], check)
	default:
		p.outside[file] = appendOnce(p.outside[file], check)
	}
}

// foundList renders one group of whereFound under its title: each place,
// with the checks that failed there when it is a file. An empty group
// renders nothing.
func foundList(title string, found map[string][]string) []string {
	if len(found) == 0 {
		return nil
	}
	lines := []string{title}
	for _, place := range slices.Sorted(maps.Keys(found)) {
		line := "- " + place
		if checks := found[place]; len(checks) > 0 {
			line += " (" + strings.Join(checks, ", ") + ")"
		}
		lines = append(lines, line)
	}
	return lines
}

// appendOnce appends name to names unless it is already there.
func appendOnce(names []string, name string) []string {
	if slices.Contains(names, name) {
		return names
	}
	return append(names, name)
}
