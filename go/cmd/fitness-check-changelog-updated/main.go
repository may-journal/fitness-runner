// Command fitness-check-changelog-updated checks that each staged commit
// writes a changelog entry about its change. A branch keeps one entry file,
// and its heading shows the commit time. CHANGELOG.md is history that only
// the release PR writes. The README holds the full rules.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/changelog"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

const (
	minOverlap   = 3
	minWordLen   = 3
	suggestWords = 10

	// stampGrace is how far behind the clock the newest heading may be.
	stampGrace  = 5 * time.Minute
	stampLayout = "2006.01.02.1504"

	changelogMD = "CHANGELOG.md"
	packageJSON = "package.json"

	msgChangelogMissing = "CHANGELOG.md missing; add it, and record your staged changes in a changelog/<name>.md entry file"
	msgStageEntry       = "Stage a changelog/<name>.md entry file that mentions your staged changes"
	msgSectionEdit      = "CHANGELOG.md is staged; only the release PR writes it, so record this change in a changelog/<name>.md entry file instead"
	msgOverlapHead      = " additions should mention at least "
	msgOverlapTail      = " words from your staged changes (found "
	msgSuggestPrefix    = "e.g. use words like: "
	msgChangelogTime    = " new section heading must use current date and time (yyyy.mm.dd.HHMM), not a guessed time"
	msgAmendHead        = "this branch already has its entry "
	msgAmendTail        = "; add your bullets to it instead of a new entry: "

	// labelEntryFile names an entry file in messages.
	labelEntryFile = "Changelog entry file"
)

var (
	headingRe   = regexp.MustCompile(`### (\d{4}\.\d{2}\.\d{2}\.\d{4})`)
	versionTSRe = regexp.MustCompile(`(\d{4}\.\d{2}\.\d{2}\.\d{4})$`)
)

// now returns the current wall-clock time; a variable so tests can pin it.
var now = time.Now

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
// mid-revert. Such a commit replays historical CHANGELOG additions rather than
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

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "changelog-updated"},
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
	if msg, ok := checkHistory(root, staged); !ok {
		return checkkit.Fail(1, msg), nil
	}
	if msg, ok := checkOneEntry(root); !ok {
		return checkkit.Fail(1, msg), nil
	}
	return judge(root, parseStagedDiff(stagedDiff(root)), now()), nil
}

// checkHistory requires CHANGELOG.md on disk, and keeps a commit from
// changing the one HEAD holds: it is history that only the release PR
// writes. A commit may add the file to a repo that lacks it.
func checkHistory(root string, staged []string) (msg string, ok bool) {
	if _, err := os.Stat(filepath.Join(root, changelogMD)); err != nil {
		return msgChangelogMissing, false
	}
	if slices.Contains(staged, changelogMD) && inHead(root, changelogMD) {
		return msgSectionEdit, false
	}
	return "", true
}

// inHead reports whether HEAD holds path.
func inHead(root, path string) bool {
	return exec.Command("git", "-C", root, "cat-file", "-e", "HEAD:"+path).Run() == nil
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

// diffFile is one file's accumulated added-line content from the staged diff,
// in first-appearance order.
type diffFile struct {
	path  string
	added string
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

// accumulateAdded folds one added line's text into path's entry in files —
// created on first sight, space-joined after — using index (path → position
// in files) to keep first-appearance order.
func accumulateAdded(files []diffFile, index map[string]int, path, text string) []diffFile {
	i, seen := index[path]
	switch {
	case !seen:
		index[path] = len(files)
		files = append(files, diffFile{path: path, added: text})
	case files[i].added != "":
		files[i].added += " " + text
	default:
		files[i].added = text
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

// judge applies the check's gates in TS order: changelog additions present,
// rest-of-diff words present, timestamp agreement, then word overlap.
func judge(root string, files []diffFile, at time.Time) checkkit.Result {
	changelogAdded, restParts := splitDiff(files)
	label := labelEntryFile
	changelogWords := extractWords(changelogAdded)
	if len(changelogWords) == 0 {
		return checkkit.Fail(1, msgStageEntry)
	}
	restWords := extractWords(strings.Join(restParts, " "))
	if len(restWords) == 0 {
		return checkkit.Pass(1)
	}
	if msg, ok := checkChangelogTime(root, label, changelogAdded, at); !ok {
		return checkkit.Fail(1, msg)
	}
	return judgeOverlap(label, changelogWords, restWords)
}

// splitDiff separates the entry files' added content from every other
// file's, keeping both in diff order.
func splitDiff(files []diffFile) (changelogAdded string, restParts []string) {
	var entryParts []string
	for _, f := range files {
		if changelog.IsEntryPath(f.path) {
			entryParts = append(entryParts, f.added)
		} else {
			restParts = append(restParts, f.added)
		}
	}
	return strings.Join(entryParts, " "), restParts
}

// checkChangelogTime requires the first "### yyyy.mm.dd.HHMM" heading in the
// added changelog content, the newest section, to carry the expected stamp.
// Later added headings are history a rewrite re-adds, so they keep their
// times. ok is true when there are no headings or the newest agrees.
func checkChangelogTime(root, label, added string, at time.Time) (msg string, ok bool) {
	m := headingRe.FindStringSubmatch(added)
	if m == nil {
		return "", true
	}
	if version, found := versionTimestamp(root); found {
		return timeMismatch(label, m[1] == version, version)
	}
	return timeMismatch(label, withinStampGrace(m[1], at), at.Format(stampLayout))
}

// timeMismatch turns a heading verdict into checkChangelogTime's result,
// naming the expected stamp on failure.
func timeMismatch(label string, ok bool, expected string) (string, bool) {
	if ok {
		return "", true
	}
	return label + msgChangelogTime + " (expected ### " + expected + ")", false
}

// withinStampGrace reports whether stamp names a minute from stampGrace
// before at up to at's own minute. The pre-commit hook stamps the heading
// before the suite runs, so the clock may pass a minute boundary in between.
func withinStampGrace(stamp string, at time.Time) bool {
	t, err := time.ParseInLocation(stampLayout, stamp, at.Location())
	if err != nil {
		return false
	}
	minute := at.Truncate(time.Minute)
	return !t.After(minute) && !t.Before(minute.Add(-stampGrace))
}

// versionTimestamp returns the end-anchored yyyy.mm.dd.HHMM suffix of the
// root package.json version, which overrides the clock when present.
func versionTimestamp(root string) (string, bool) {
	raw, err := os.ReadFile(filepath.Join(root, packageJSON))
	if err != nil {
		return "", false
	}
	var pkg struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(raw, &pkg) != nil {
		return "", false
	}
	if m := versionTSRe.FindStringSubmatch(pkg.Version); m != nil {
		return m[1], true
	}
	return "", false
}

// judgeOverlap passes when the changelog additions share at least minOverlap
// distinct words with the rest of the staged diff; otherwise it reports the
// shortfall (first five overlapping words) and suggests the first
// suggestWords rest-diff words.
func judgeOverlap(label string, changelogWords, restWords []string) checkkit.Result {
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
		fmt.Sprintf("%s%s%d%s%d: %s)", label,
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
