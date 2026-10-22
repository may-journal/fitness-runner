// Command fitness-check-changelog-updated checks that each staged commit
// writes a changelog entry about its change. A branch keeps one entry file.
// CHANGELOG.md is no part of it. The README holds the full rules.
package check

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/changelog"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

const (
	minOverlap   = 3
	minWordLen   = 3
	suggestWords = 10

	msgStageEntry    = "Stage a changelog/<yyyy-mm-dd-HHMM>-<change>.md entry file that mentions your staged changes"
	msgOverlapHead   = " additions should mention at least "
	msgOverlapTail   = " words from your staged changes (found "
	msgSuggestPrefix = "e.g. use words like: "
	msgAmendHead     = "this branch already has its entry "
	msgAmendTail     = "; add your bullets to it instead of a new entry: "

	// labelEntryFile names an entry file in messages.
	labelEntryFile = "Changelog entry file"
)

// stagedDiff returns raw `git diff --cached` output for root; a variable so
// tests can inject fixed diffs. Failures degrade to whatever stdout produced,
// mirroring the TS execSyncResult behavior.
var stagedDiff = func(root string) string {
	cmd := exec.Command("git", "diff", "--cached")
	cmd.Dir = root
	out, _ := cmd.Output()
	return string(out)
}

// replayInProgress reports whether Git is mid-merge, mid-cherry-pick, or
// mid-revert. Such a commit replays historical entry additions rather than
// authoring a new entry, so the freshness gates must not apply. A variable so
// tests can pin it.
var replayInProgress = func(root string) bool {
	for _, ref := range []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD"} {
		if gitStateFileExists(root, ref) {
			return true
		}
	}
	return false
}

// gitStateFileExists reports whether Git's <ref> state file exists, resolving
// its path with `git rev-parse --git-path` so it holds when the .git entry is
// a file, not a directory.
func gitStateFileExists(root, ref string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-path", ref)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	path := strings.TrimSpace(string(out))
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	_, statErr := os.Stat(path)
	return statErr == nil
}

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "changelog-updated", Scope: checkkit.ScopeGit},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	if replayInProgress(root) {
		return checkkit.Pass(0), nil
	}
	staged := resolveStagedFiles(root)
	if len(staged) == 0 {
		return checkkit.Pass(0), nil
	}
	if errs, err := entryErrors(root); err != nil || len(errs) > 0 {
		return checkkit.Fail(1, errs...), err
	}
	return judge(parseStagedDiff(stagedDiff(root))), nil
}

// entryErrors fails a commit that edits an entry file a release holds, then
// one that adds a second entry file to its branch, then one that adds an
// entry file not named for its heading's date.
func entryErrors(root string) ([]string, error) {
	if released, err := changelog.StagedReleasedEdits(root); err != nil || len(released) > 0 {
		return released, err
	}
	if msg, ok := checkOneEntry(root); !ok {
		return []string{msg}, nil
	}
	return changelog.StagedNameErrors(root), nil
}

// checkOneEntry keeps a branch to one entry file: once an earlier commit on
// the branch added one, a later commit amends it and adds none. The message
// names the branch's first entry file and every one the commit adds.
func checkOneEntry(root string) (msg string, ok bool) {
	own := changelog.BranchEntryFiles(root)
	if len(own) == 0 {
		return "", true
	}
	extra := changelog.StagedEntryFiles(root)
	if len(extra) == 0 {
		return "", true
	}
	return fmt.Sprintf("%s%s%s%s", msgAmendHead, own[0], msgAmendTail, strings.Join(extra, ", ")), false
}

// resolveStagedFiles prefers the runner-provided FITNESS_STAGED_FILES (even
// when present but empty); standalone runs fall back to asking git.
func resolveStagedFiles(root string) []string {
	if _, present := os.LookupEnv("FITNESS_STAGED_FILES"); present {
		return checkkit.StagedFiles()
	}
	return gitx.StagedFiles(root)
}

// diffFile is one file's added lines from the staged diff, space-joined to read,
// in first-appearance order.
type diffFile struct {
	path  string
	added []string
}

// parseStagedDiff walks raw `git diff --cached` output preserving the TS
// parser's quirks exactly: "+++ " headers set the current file (trimmed, one
// leading "b/" stripped), and added lines — "+" but not "++" — accumulate
// space-joined per file.
func parseStagedDiff(diff string) []diffFile {
	var files []diffFile
	index := map[string]int{}
	current := ""
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+++ ") {
			current = strings.TrimPrefix(strings.TrimSpace(line[4:]), "b/")
			continue
		}
		if isAddedLine(current, line) {
			files = accumulateAdded(files, index, current, line[1:])
		}
	}
	return files
}

// isAddedLine reports whether line is added content attributable to a known
// current file: a "+" prefix but not "++" (which would be a "+++" header).
func isAddedLine(current, line string) bool {
	return current != "" && strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "++")
}

// accumulateAdded appends one added line's text to path's entry in files —
// created on first sight — using index (path → position in files) to keep
// first-appearance order. Empty lines before the first text are dropped, so
// the parts space-join to what the TS parser's concatenation built, without
// copying the whole file per line.
func accumulateAdded(files []diffFile, index map[string]int, path, text string) []diffFile {
	i, seen := index[path]
	if !seen {
		i = len(files)
		index[path] = i
		files = append(files, diffFile{path: path})
	}
	if len(files[i].added) > 0 || text != "" {
		files[i].added = append(files[i].added, text)
	}
	return files
}

// extractWords lowercases text, treats every non-alphanumeric rune as a
// separator, and returns the distinct words of at least minWordLen characters
// in first-occurrence order.
func extractWords(text string) []string {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
	seen := map[string]bool{}
	var words []string
	for _, w := range fields {
		if len(w) < minWordLen || seen[w] {
			continue
		}
		seen[w] = true
		words = append(words, w)
	}
	return words
}

// judge applies the check's gates in order: changelog additions present,
// rest-of-diff words present, then word overlap.
func judge(files []diffFile) checkkit.Result {
	changelogAdded, restParts := splitDiff(files)
	changelogWords := extractWords(changelogAdded)
	if len(changelogWords) == 0 {
		return checkkit.Fail(1, msgStageEntry)
	}
	restWords := extractWords(strings.Join(restParts, " "))
	if len(restWords) == 0 {
		return checkkit.Pass(1)
	}
	return judgeOverlap(changelogWords, restWords)
}

// splitDiff separates the entry files' added content from every other
// file's, keeping both in diff order.
func splitDiff(files []diffFile) (changelogAdded string, restParts []string) {
	var entryParts []string
	for _, f := range files {
		if changelog.IsEntryPath(f.path) {
			entryParts = append(entryParts, strings.Join(f.added, " "))
		} else {
			restParts = append(restParts, strings.Join(f.added, " "))
		}
	}
	return strings.Join(entryParts, " "), restParts
}

// judgeOverlap passes when the changelog additions share at least minOverlap
// distinct words with the rest of the staged diff; otherwise it reports the
// shortfall (first five overlapping words) and suggests the first
// suggestWords rest-diff words.
func judgeOverlap(changelogWords, restWords []string) checkkit.Result {
	overlap := overlapWords(changelogWords, restWords)
	if len(overlap) >= minOverlap {
		return checkkit.Pass(1)
	}
	shown := overlap
	if len(shown) > 5 {
		shown = shown[:5]
	}
	suggested := restWords
	if len(suggested) > suggestWords {
		suggested = suggested[:suggestWords]
	}
	return checkkit.Fail(1,
		fmt.Sprintf("%s%s%d%s%d: %s)", labelEntryFile,
			msgOverlapHead, minOverlap, msgOverlapTail, len(overlap), strings.Join(shown, ", ")),
		msgSuggestPrefix+strings.Join(suggested, ", "))
}

// overlapWords returns the changelog words also present in the rest of the
// staged diff, in changelog first-occurrence order.
func overlapWords(changelogWords, restWords []string) []string {
	restSet := make(map[string]bool, len(restWords))
	for _, w := range restWords {
		restSet[w] = true
	}
	var overlap []string
	for _, w := range changelogWords {
		if restSet[w] {
			overlap = append(overlap, w)
		}
	}
	return overlap
}
