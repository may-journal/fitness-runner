package check

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
	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/langs"
	"github.com/may-journal/fitness-runner/go/internal/par"
	"github.com/may-journal/fitness-runner/go/internal/sharedconf"
	"github.com/may-journal/fitness-runner/go/internal/spell"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// maxIssuesPerFile mirrors cspell's default maxNumberOfProblems.
const maxIssuesPerFile = 100

var check = checkkit.Check{
	Describe: checkkit.Describe{Name: "cspell", Scope: checkkit.ScopeFiles, Blob: true},
	Run:      run,
}

func Main() {
	checkkit.Main(check)
}

func run(root string, args []string) (checkkit.Result, error) {
	if res, handled, err := bodycheck.RunDoc(root, args, func(r, content string) []string {
		cfg := loadConfig(r)
		return checkkit.Lines(errorLine, checkContent("(description)", content, newChecker(cfg), cfg.FlagWords))
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
	return checkkit.Found(len(files), errorLine, checkFiles(root, files, newChecker(cfg), cfg)...)
}

// targets lists the files to spell-check: the changed files, or every
// tracked file when the run is unscoped, minus lockfiles, build wrapper
// scripts, and those .gitattributes marks linguist-generated.
func targets(root string) []string {
	files, scanned := stagedPaths(root)
	if !scanned {
		files = walkfs.FilesByExt(root, "")
	}
	// A lockfile's or wrapper's words are package names, hashes and tool
	// names no one can fix.
	toolWritten := lang.IsToolWritten(langs.All)
	var kept []string
	for _, f := range files {
		if !toolWritten(f) {
			kept = append(kept, f)
		}
	}
	return walkfs.WithoutGenerated(root, kept)
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

// spelling is one kind of word cspell reports: the rule a finding names,
// the word that starts its message, and the fix.
type spelling struct {
	rule, kind, fix string
}

var (
	// unknown is a word no dictionary and no cspell.json words list holds.
	unknown = spelling{"unknown-word", "Unknown",
		"correct the spelling, or add the word to the words list in cspell.json"}
	// forbidden is a word or phrase cspell.json's flagWords lists.
	forbidden = spelling{"forbidden-word", "Forbidden",
		"reword without it; cspell.json's flagWords forbids it"}
)

// unreadable is the rule of a file the check could not read.
const unreadable = "unreadable-file"

// errorLine renders a finding as the cspell CLI would print it:
// "<path>:<line>:<col> - Unknown word (<word>)", or "<path>: <message>
// (run: <fix>)" for a file it could not read.
func errorLine(f checkkit.Finding) string {
	if f.Line == 0 {
		return fmt.Sprintf("%s: %s (run: %s)", f.File, f.Message, f.Fix)
	}
	return fmt.Sprintf("%s:%d:%d - %s", f.File, f.Line, f.Column, f.Message)
}

// checkContent scans one in-memory document (never reading from disk) with the
// same per-line/word pass checkFiles applies per file: its unknown words,
// then each flagWords match.
func checkContent(name, content string, checker *spell.Checker, flags []string) []checkkit.Finding {
	found := findings(name, unknown, checker.CheckText(content))
	return append(found, findings(name, forbidden, spell.FindPhrases(content, flags))...)
}

// checkFiles scans each file as the cspell CLI run from the root would.
// The cspell.json files that list the flagWords are never checked for them.
func checkFiles(root string, files []string, checker *spell.Checker, cfg config) []checkkit.Finding {
	perFile := par.Map(len(files), 0, func(i int) []checkkit.Finding {
		flags := cfg.FlagWords
		if path.Base(files[i]) == "cspell.json" {
			flags = nil
		}
		return fileIssues(root, files[i], checker, flags)
	})
	var found []checkkit.Finding
	for _, fe := range perFile {
		found = append(found, fe...)
	}
	return found
}

// fileIssues scans one file and returns its findings: a read failure yields
// the run-cspell hint, binary files yield nothing (counted as checked, never
// spelled).
func fileIssues(root, rel string, checker *spell.Checker, flags []string) []checkkit.Finding {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return []checkkit.Finding{{File: rel, Rule: unreadable, Message: "cspell reported issues", Fix: "npx cspell <files>"}}
	}
	if bytes.IndexByte(data[:min(len(data), 8192)], 0) >= 0 {
		return nil
	}
	return checkContent(rel, string(data), checker, flags)
}

// findings turns one kind of issue for one file into findings, capped at
// maxIssuesPerFile like cspell's default maxNumberOfProblems.
func findings(rel string, kind spelling, issues []spell.Issue) []checkkit.Finding {
	if len(issues) > maxIssuesPerFile {
		issues = issues[:maxIssuesPerFile]
	}
	var found []checkkit.Finding
	for _, is := range issues {
		found = append(found, checkkit.Finding{
			File: rel, Line: is.Line, Column: is.Col, Rule: kind.rule,
			Message: fmt.Sprintf("%s word (%s)", kind.kind, is.Word), Fix: kind.fix,
		})
	}
	return found
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
