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
	checker := spell.NewChecker(spell.EmbeddedWords())
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
		if stagedSkip[path.Base(p)] {
			continue
		}
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(p))); err == nil && !info.IsDir() {
			files = append(files, p)
		}
	}
	return files, true
}

// checkFiles scans each file and formats issues exactly like the cspell CLI
// run from the root: "<path>:<line>:<col> - Unknown word (<word>)".
func checkFiles(root string, files []string, checker *spell.Checker) []string {
	var errs []string
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: cspell reported issues (run: npx cspell <files>)", rel))
			continue
		}
		if bytes.IndexByte(data[:min(len(data), 8192)], 0) >= 0 {
			continue // binary file: counted as checked, never spelled
		}
		issues := checker.CheckText(string(data))
		if len(issues) > maxIssuesPerFile {
			issues = issues[:maxIssuesPerFile]
		}
		for _, is := range issues {
			errs = append(errs, fmt.Sprintf("%s:%d:%d - Unknown word (%s)", rel, is.Line, is.Col, is.Word))
		}
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
		if err != nil {
			return nil
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil || rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if matcher.Matches(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".md") && !matcher.Matches(rel) {
			files = append(files, rel)
		}
		return nil
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

// withoutGitignored drops paths `git check-ignore` reports as ignored,
// batched over stdin; outside a git repo (or without git) it filters
// nothing.
func withoutGitignored(root string, files []string) []string {
	if len(files) == 0 {
		return files
	}
	cmd := exec.Command("git", "-C", root, "check-ignore", "--stdin", "-z")
	cmd.Stdin = strings.NewReader(strings.Join(files, "\x00") + "\x00")
	out, err := cmd.Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
			return files // not a git repo or git missing: no filtering
		}
	}
	ignored := make(map[string]bool)
	for _, p := range strings.Split(string(out), "\x00") {
		if p != "" {
			ignored[p] = true
		}
	}
	var kept []string
	for _, f := range files {
		if !ignored[f] {
			kept = append(kept, f)
		}
	}
	return kept
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
