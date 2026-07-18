// Command fitness-check-cspell is the Go port of the cspell check's soul: no
// unknown words in the checked files, with the project cspell.json words
// always winning over the embedded base dictionaries (internal/spell).
//
// Staged mode (FITNESS_STAGED_FILES non-empty) checks the staged files after
// dropping .gitignore/package-lock.json/tsconfig.json basenames and paths
// that no longer exist; with nothing left it passes without scanning. With no
// staged context it checks every **/*.md under the root, honoring the
// resolved cspell.json ignorePaths and, when useGitignore is set, the repo's
// gitignore via `git check-ignore`.
//
// cspell.json resolves through internal/sharedconf: the repo's own file
// wins, then an installed node_modules/@mayjournal/fitness-shared, then the
// copy embedded in this binary, materialized to the cache on demand — so a
// repo with no npm anywhere still gets the shared words and ignorePaths.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/par"
	"github.com/may-journal/fitness-runner/go/internal/sharedconf"
	"github.com/may-journal/fitness-runner/go/internal/spell"
)

// maxIssuesPerFile mirrors cspell's default maxNumberOfProblems.
const maxIssuesPerFile = 100

// stagedSkip lists basenames the shared cspell.json always ignores; staged
// paths matching them are dropped before any scanning.
var stagedSkip = map[string]bool{
	".gitignore":        true,
	"package-lock.json": true,
	"tsconfig.json":     true,
}

var check = checkkit.Check{
	Describe: checkkit.Describe{Name: "cspell"},
	Run:      run,
}

func main() {
	checkkit.Main(check)
}

func run(root string, _ []string) (checkkit.Result, error) {
	cfg := loadConfig(root)
	checker := spell.NewEmbeddedChecker()
	checker.AddWords(cfg.Words)
	checker.AddWords(cfg.IgnoreWords)
	files, scanned := stagedPaths(root)
	if !scanned {
		files = markdownFiles(root, cfg)
	} else if len(files) == 0 {
		return checkkit.Pass(0), nil
	}
	errs := checkFiles(root, files, checker)
	if len(errs) > 0 {
		return checkkit.Fail(len(files), errs...), nil
	}
	return checkkit.Pass(len(files)), nil
}

// stagedPaths returns the staged files to check and whether staged context
// exists at all: paths with an always-ignored basename or that no longer
// exist on disk are dropped.
func stagedPaths(root string) (files []string, staged bool) {
	stagedFiles := checkkit.StagedFiles()
	if len(stagedFiles) == 0 {
		return nil, false
	}
	for _, p := range stagedFiles {
		if checkableStaged(root, p) {
			files = append(files, p)
		}
	}
	return files, true
}

// checkableStaged reports whether a staged path should be scanned: not an
// always-ignored basename and still an existing non-directory under root.
func checkableStaged(root, p string) bool {
	if stagedSkip[path.Base(p)] {
		return false
	}
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(p)))
	return err == nil && !info.IsDir()
}

// checkFiles scans each file and formats issues exactly like the cspell CLI
// run from the root: "<path>:<line>:<col> - Unknown word (<word>)".
func checkFiles(root string, files []string, checker *spell.Checker) []string {
	perFile := par.Map(len(files), 0, func(i int) []string {
		return fileIssues(root, files[i], checker)
	})
	var errs []string
	for _, fe := range perFile {
		errs = append(errs, fe...)
	}
	return errs
}

// fileIssues scans one file and returns its formatted issue lines: a read
// failure yields the run-cspell hint, binary files yield nothing (counted as
// checked, never spelled).
func fileIssues(root, rel string, checker *spell.Checker) []string {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return []string{fmt.Sprintf("%s: cspell reported issues (run: npx cspell <files>)", rel)}
	}
	if bytes.IndexByte(data[:min(len(data), 8192)], 0) >= 0 {
		return nil
	}
	return formatIssues(rel, checker.CheckText(string(data)))
}

// formatIssues renders issues for one file, capped at maxIssuesPerFile like
// cspell's default maxNumberOfProblems.
func formatIssues(rel string, issues []spell.Issue) []string {
	if len(issues) > maxIssuesPerFile {
		issues = issues[:maxIssuesPerFile]
	}
	var errs []string
	for _, is := range issues {
		errs = append(errs, fmt.Sprintf("%s:%d:%d - Unknown word (%s)", rel, is.Line, is.Col, is.Word))
	}
	return errs
}

// markdownFiles walks root for **/*.md, pruning ignored directories, then
// filters through ignorePaths and (when configured) the repo's gitignore.
// Paths come back sorted case-insensitively like cspell's own file order.
func markdownFiles(root string, cfg config) []string {
	matcher := spell.NewIgnoreMatcher(append([]string{"node_modules", ".git"}, cfg.IgnorePaths...))
	var files []string
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		rel, ok := walkRel(root, p, err)
		if !ok {
			return nil
		}
		return collectMarkdown(rel, d, matcher, &files)
	})
	if cfg.UseGitignore {
		files = withoutGitignored(root, files)
	}
	sort.Slice(files, func(i, j int) bool {
		li, lj := strings.ToLower(files[i]), strings.ToLower(files[j])
		if li != lj {
			return li < lj
		}
		return files[i] < files[j]
	})
	return files
}

// walkRel converts a WalkDir callback's absolute path to a slash-separated
// root-relative one; ok is false for walk errors and the root itself, which
// the walk skips without failing.
func walkRel(root, p string, err error) (string, bool) {
	if err != nil {
		return "", false
	}
	rel, relErr := filepath.Rel(root, p)
	if relErr != nil || rel == "." {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

// collectMarkdown handles one WalkDir entry: ignored directories are pruned,
// and non-ignored .md files accumulate into *files.
func collectMarkdown(rel string, d os.DirEntry, matcher *spell.IgnoreMatcher, files *[]string) error {
	if d.IsDir() {
		if matcher.Matches(rel) {
			return filepath.SkipDir
		}
		return nil
	}
	if strings.HasSuffix(d.Name(), ".md") && !matcher.Matches(rel) {
		*files = append(*files, rel)
	}
	return nil
}

// withoutGitignored drops paths `git check-ignore` reports as ignored,
// batched over stdin; outside a git repo (or without git) it filters
// nothing.
func withoutGitignored(root string, files []string) []string {
	if len(files) == 0 {
		return files
	}
	ignored, ok := gitIgnoredSet(root, files)
	if !ok {
		return files // not a git repo or git missing: no filtering
	}
	var kept []string
	for _, f := range files {
		if !ignored[f] {
			kept = append(kept, f)
		}
	}
	return kept
}

// gitIgnoredSet asks `git check-ignore --stdin -z` which files are ignored;
// ok is false when git exits with anything but the check-ignore verdict
// codes 0 and 1 (not a repo, git missing), meaning no filtering applies.
func gitIgnoredSet(root string, files []string) (map[string]bool, bool) {
	cmd := exec.Command("git", "-C", root, "check-ignore", "--stdin", "-z")
	cmd.Stdin = strings.NewReader(strings.Join(files, "\x00") + "\x00")
	out, err := cmd.Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
			return nil, false
		}
	}
	return nulSeparatedSet(string(out)), true
}

// nulSeparatedSet builds the membership set of a NUL-separated list,
// dropping empty entries (the trailing terminator).
func nulSeparatedSet(out string) map[string]bool {
	set := make(map[string]bool)
	for _, p := range strings.Split(out, "\x00") {
		if p != "" {
			set[p] = true
		}
	}
	return set
}

// config is the subset of cspell.json this check honors.
type config struct {
	Words        []string `json:"words"`
	IgnoreWords  []string `json:"ignoreWords"`
	IgnorePaths  []string `json:"ignorePaths"`
	UseGitignore bool     `json:"useGitignore"`
}

// loadConfig reads the resolved cspell.json — repo-local, installed
// @mayjournal/fitness-shared, or the embedded copy materialized on demand
// (the sharedconf.Resolve contract); when even the fallback fails to
// materialize (or the file is unreadable) the embedded dictionaries stand
// alone.
func loadConfig(root string) config {
	var cfg config
	p := sharedconf.Resolve(root, "cspell.json")
	if p == "" {
		return cfg
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(raw, &cfg)
	return cfg
}
