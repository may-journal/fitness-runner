// Command fitness-check-changelog-updated gates staged commits on the
// changelog: CHANGELOG.md additions must share at least three distinct words
// with the rest of the staged diff, and every new "### yyyy.mm.dd.HHMM"
// section heading must carry the expected stamp (the root package.json
// version suffix, else the current local wall clock) — the Go port of the
// changelog-updated check. One deviation from the TypeScript original: the
// suggestion line lists the first ten rest-diff words deterministically where
// TS sampled ten at random.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

const (
	minOverlap   = 3
	minWordLen   = 3
	suggestWords = 10

	changelogMD = "CHANGELOG.md"
	packageJSON = "package.json"

	msgChangelogMissing = "CHANGELOG.md missing; add it and mention your staged changes"
	msgStageChangelog   = "Stage CHANGELOG.md and add an entry that mentions your staged changes"
	msgOverlapHead      = "CHANGELOG.md additions should mention at least "
	msgOverlapTail      = " words from your staged changes (found "
	msgSuggestPrefix    = "e.g. use words like: "
	msgChangelogTime    = "CHANGELOG.md new section heading must use current date and time (yyyy.mm.dd.HHMM), not a guessed time"
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

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "changelog-updated"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	if len(resolveStagedFiles(root)) == 0 {
		return checkkit.Pass(0), nil
	}
	if _, err := os.Stat(filepath.Join(root, changelogMD)); err != nil {
		return checkkit.Fail(1, msgChangelogMissing), nil
	}
	return judge(root, parseStagedDiff(stagedDiff(root)), now()), nil
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
	changelogWords := extractWords(changelogAdded)
	if len(changelogWords) == 0 {
		return checkkit.Fail(1, msgStageChangelog)
	}
	restWords := extractWords(strings.Join(restParts, " "))
	if len(restWords) == 0 {
		return checkkit.Pass(1)
	}
	if msg, ok := checkChangelogTime(root, changelogAdded, at); !ok {
		return checkkit.Fail(1, msg)
	}
	return judgeOverlap(changelogWords, restWords)
}

// splitDiff separates the changelog's added content from every other file's,
// keeping the rest in diff order.
func splitDiff(files []diffFile) (changelogAdded string, restParts []string) {
	for _, f := range files {
		if f.path == changelogMD {
			changelogAdded = f.added
		} else {
			restParts = append(restParts, f.added)
		}
	}
	return changelogAdded, restParts
}

// checkChangelogTime requires every "### yyyy.mm.dd.HHMM" heading in the
// added changelog content to match the expected stamp; ok true when there are
// no headings or all agree.
func checkChangelogTime(root, added string, at time.Time) (msg string, ok bool) {
	matches := headingRe.FindAllStringSubmatch(added, -1)
	if len(matches) == 0 {
		return "", true
	}
	expected := expectedTimestamp(root, at)
	for _, m := range matches {
		if m[1] != expected {
			return msgChangelogTime + " (expected ### " + expected + ")", false
		}
	}
	return "", true
}

// expectedTimestamp prefers the end-anchored yyyy.mm.dd.HHMM suffix of the
// root package.json version; anything short of that falls back to the current
// local wall clock.
func expectedTimestamp(root string, at time.Time) string {
	if raw, err := os.ReadFile(filepath.Join(root, packageJSON)); err == nil {
		var pkg struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(raw, &pkg) == nil {
			if m := versionTSRe.FindStringSubmatch(pkg.Version); m != nil {
				return m[1]
			}
		}
	}
	return at.Format("2006.01.02.1504")
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
		fmt.Sprintf("%s%d%s%d: %s)",
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
