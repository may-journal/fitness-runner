// Command fitness-check-cspell is the Go port of the cspell check's soul: no
// unknown words in the checked files, with the project cspell.json words
// always winning over the embedded base dictionaries (internal/spell).
//
// Scoped mode (a changed-file list in the environment) checks the changed
// files that still exist on disk; with none left it passes without scanning.
// With no scope it checks every tracked file (internal/walkfs). Files the
// repo's .gitattributes marks linguist-generated, such as lock files, are
// left out. Binary files are counted but never spelled. No path is ever ignored: a resolved
// cspell.json that sets ignorePaths fails the check instead.
//
// cspell.json resolves through internal/sharedconf: the repo's own file
// wins, then an installed node_modules/@mayjournal/fitness-shared, then the
// copy embedded in this binary, materialized to the cache on demand — so a
// repo with no npm anywhere still gets the shared words.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/bodycheck"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/par"
	"github.com/may-journal/fitness-runner/go/internal/sharedconf"
	"github.com/may-journal/fitness-runner/go/internal/spell"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// maxIssuesPerFile mirrors cspell's default maxNumberOfProblems.
const maxIssuesPerFile = 100

var check = checkkit.Check{
	Describe: checkkit.Describe{Name: "cspell"},
	Run:      run,
}

func main() {
	checkkit.Main(check)
}

func run(root string, args []string) (checkkit.Result, error) {
	if res, handled, err := bodycheck.RunDoc(root, args, func(r, content string) []string {
		cfg := loadConfig(r)
		return checkContent("(description)", content, newChecker(cfg), cfg.FlagWords)
	}); handled || err != nil {
		return res, err
	}
	return walkFiles(root), nil
}

// walkFiles spell-checks the changed files, or every tracked file under
// root when the run is unscoped. A resolved cspell.json with ignorePaths
// fails before any scan.
func walkFiles(root string) checkkit.Result {
	cfg := loadConfig(root)
	if cfg.IgnorePaths != nil {
		return checkkit.Fail(0, fmt.Sprintf(
			"%s sets ignorePaths; cspell checks every tracked file, so remove ignorePaths and fix the findings instead",
			cfg.path))
	}
	files := targets(root)
	errs := checkFiles(root, files, newChecker(cfg), cfg)
	if len(errs) > 0 {
		return checkkit.Fail(len(files), errs...)
	}
	return checkkit.Pass(len(files))
}

// targets lists the files to spell-check: the changed files, or every
// tracked file when the run is unscoped, minus those .gitattributes marks
// linguist-generated, such as lock files.
func targets(root string) []string {
	files, scanned := stagedPaths(root)
	if !scanned {
		files = walkfs.FilesByExt(root, "")
	}
	return walkfs.WithoutGenerated(root, files)
}

// stagedPaths returns the changed files to check and whether a scope exists
// at all: only paths that no longer exist on disk are dropped.
func stagedPaths(root string) (files []string, staged bool) {
	stagedFiles := checkkit.ChangedFiles()
	if len(stagedFiles) == 0 {
		return nil, false
	}
	for _, p := range stagedFiles {
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(p))); err == nil && !info.IsDir() {
			files = append(files, p)
		}
	}
	return files, true
}

// newChecker builds the spell.Checker used by both the file walk and body
// mode: the embedded base dictionaries with the resolved cspell.json words and
// ignoreWords layered on so project terms always win.
func newChecker(cfg config) *spell.Checker {
	checker := spell.NewEmbeddedChecker()
	checker.AddWords(cfg.Words)
	checker.AddWords(cfg.IgnoreWords)
	return checker
}

// checkContent scans one in-memory document (never reading from disk) with the
// same per-line/word pass checkFiles applies per file, emitting issues in the
// identical "<name>:<line>:<col> - Unknown word (<word>)" format, then each
// flagWords match as "<name>:<line>:<col> - Forbidden word (<entry>)".
func checkContent(name, content string, checker *spell.Checker, flags []string) []string {
	errs := formatIssues(name, "Unknown", checker.CheckText(content))
	return append(errs, formatIssues(name, "Forbidden", spell.FindPhrases(content, flags))...)
}

// checkFiles scans each file and formats issues exactly like the cspell CLI
// run from the root: "<path>:<line>:<col> - Unknown word (<word>)". The
// cspell.json files that list the flagWords are never checked for them.
func checkFiles(root string, files []string, checker *spell.Checker, cfg config) []string {
	perFile := par.Map(len(files), 0, func(i int) []string {
		flags := cfg.FlagWords
		if path.Base(files[i]) == "cspell.json" {
			flags = nil
		}
		return fileIssues(root, files[i], checker, flags)
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
func fileIssues(root, rel string, checker *spell.Checker, flags []string) []string {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return []string{fmt.Sprintf("%s: cspell reported issues (run: npx cspell <files>)", rel)}
	}
	if bytes.IndexByte(data[:min(len(data), 8192)], 0) >= 0 {
		return nil
	}
	return checkContent(rel, string(data), checker, flags)
}

// formatIssues renders one kind of issue for one file, capped at
// maxIssuesPerFile like cspell's default maxNumberOfProblems.
func formatIssues(rel, kind string, issues []spell.Issue) []string {
	if len(issues) > maxIssuesPerFile {
		issues = issues[:maxIssuesPerFile]
	}
	var errs []string
	for _, is := range issues {
		errs = append(errs, fmt.Sprintf("%s:%d:%d - %s word (%s)", rel, is.Line, is.Col, kind, is.Word))
	}
	return errs
}

// config is the subset of cspell.json this check reads; path is the
// resolved file it came from. IgnorePaths is read only to reject it.
// FlagWords are words or phrases no checked text may contain.
type config struct {
	Words       []string        `json:"words"`
	FlagWords   []string        `json:"flagWords"`
	IgnoreWords []string        `json:"ignoreWords"`
	IgnorePaths json.RawMessage `json:"ignorePaths"`
	path        string
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
	cfg.path = displayPath(root, p)
	return cfg
}

// displayPath shows the resolved cspell.json relative to root when it lives
// inside the repo, else as its absolute path.
func displayPath(root, p string) string {
	if rel, err := filepath.Rel(root, p); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return p
}
