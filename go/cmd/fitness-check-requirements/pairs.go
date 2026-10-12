package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

var (
	// numberedTest matches a test file named for its requirements:
	// 0014_test.go, or 0003_0004_test.go for two.
	numberedTest = regexp.MustCompile(`^(\d{4}(?:_\d{4})*)_test\.go$`)
	// numberedDoc matches a requirement doc file: 0014-kebab-title.md.
	numberedDoc = regexp.MustCompile(`^(\d{4})-.+\.md$`)
)

// pairs maps each requirement number to its doc and test files, and each
// requirement test file to the numbers it proves.
type pairs struct {
	docs    map[string][]string
	tests   map[string][]string
	numbers map[string][]string
}

// newPairs pairs every doc and test file by number. A test file proves the
// numbers in its name and those of the tests it holds; a changed file that
// is gone still pairs by its name.
func newPairs(docPaths, testFiles []string, tests []goTest, changed []string) pairs {
	p := pairs{docs: map[string][]string{}, tests: map[string][]string{}, numbers: map[string][]string{}}
	for _, f := range append(slices.Clone(docPaths), changed...) {
		if n := docNumber(f); n != "" {
			p.docs[n] = addOnce(p.docs[n], f)
		}
	}
	p.addTestFiles(append(slices.Clone(testFiles), changed...), tests)
	return p
}

// addTestFiles pairs each file by the numbers in its name, then each test
// named for an acceptance by its number.
func (p pairs) addTestFiles(files []string, tests []goTest) {
	for _, f := range files {
		for _, n := range nameNumbers(f) {
			p.addTest(n, f)
		}
	}
	for _, t := range tests {
		if t.id != "" {
			p.addTest(t.id[:4], t.file)
		}
	}
}

// addTest records that file proves requirement n.
func (p pairs) addTest(n, file string) {
	p.tests[n] = addOnce(p.tests[n], file)
	p.numbers[file] = addOnce(p.numbers[file], n)
}

// addOnce appends s unless list holds it already.
func addOnce(list []string, s string) []string {
	if slices.Contains(list, s) {
		return list
	}
	return append(list, s)
}

// docNumber returns the requirement number of a doc path, or "".
func docNumber(f string) string {
	if !strings.HasPrefix(f, docsDir) {
		return ""
	}
	if m := numberedDoc.FindStringSubmatch(path.Base(f)); m != nil {
		return m[1]
	}
	return ""
}

// nameNumbers returns the requirement numbers a test file's name carries.
func nameNumbers(f string) []string {
	m := numberedTest.FindStringSubmatch(path.Base(f))
	if m == nil {
		return nil
	}
	return strings.Split(m[1], "_")
}

// errors flags each changed requirement test whose docs all stayed
// unchanged, and each changed doc whose tests all stayed unchanged, unless a
// trailer exempts its number.
func (p pairs) judge(changed []string, exempt map[string]bool) []string {
	isChanged := map[string]bool{}
	for _, f := range changed {
		isChanged[f] = true
	}
	var errs []string
	for _, f := range changed {
		errs = append(errs, unpaired(f, p.numbers[f], p.docs, "requirement doc", isChanged, exempt)...)
		if n := docNumber(f); n != "" {
			errs = append(errs, unpaired(f, []string{n}, p.tests, "requirement test", isChanged, exempt)...)
		}
	}
	return errs
}

// unpaired names file and its missing half when none of its partners
// changed with it. A number with no partner is ownership's concern.
func unpaired(file string, numbers []string, partners map[string][]string, half string, changed, exempt map[string]bool) []string {
	missing := unchangedPartners(numbers, partners, changed, exempt)
	if len(missing) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("%s changed without its %s %s; change both together, or add the commit trailer `%s: %s <reason>`",
		file, half, strings.Join(missing, " or "), trailerKey, strings.Join(numbers, " "))}
}

// unchangedPartners lists the partner files of numbers when none of them
// changed and no number is exempt; nil otherwise.
func unchangedPartners(numbers []string, partners map[string][]string, changed, exempt map[string]bool) []string {
	var all []string
	for _, n := range numbers {
		if exempt[n] {
			return nil
		}
		all = append(all, partners[n]...)
	}
	if slices.ContainsFunc(all, func(f string) bool { return changed[f] }) {
		return nil
	}
	return all
}

// scopedPairs judges pairs in a suite run, except in the pre-commit hook,
// which leaves them to commit-msg so an exemption trailer is readable.
func scopedPairs(root string, docPaths, testFiles []string, tests []goTest) []string {
	if os.Getenv("FITNESS_INTERNAL_HOOK") == "pre-commit" {
		return nil
	}
	return pairErrors(root, docPaths, testFiles, tests, checkkit.ChangedFiles())
}

// pairErrors judges a scoped run's changed files: staged at commit time, or
// a pull request's diff against the current merge base with its base branch.
// A full scan, and a local merge, cherry-pick, or revert, judge no pairs.
func pairErrors(root string, docPaths, testFiles []string, tests []goTest, changed []string) []string {
	if changed == nil || replaying(root) {
		return nil
	}
	exempt, errs := exemptions(root)
	errs = append(errs, newPairs(docPaths, testFiles, tests, changed).judge(changed, exempt)...)
	sort.Strings(errs)
	return errs
}

// replayHeads are the refs git keeps while a commit replays another one.
var replayHeads = []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD"}

// replaying reports whether a local commit replays changes judged where they
// were first made. A pull request run still judges its whole diff.
func replaying(root string) bool {
	if os.Getenv("GITHUB_BASE_REF") != "" {
		return false
	}
	for _, ref := range replayHeads {
		if exec.Command("git", "-C", root, "rev-parse", "-q", "--verify", ref).Run() == nil {
			return true
		}
	}
	return false
}

// commitChanges is what the commit being written changes: the staged files,
// or for an amend everything staged since the amended commit's parent.
func commitChanges(root string) []string {
	if os.Getenv("FITNESS_COMMIT_AMEND") != "1" {
		return checkkit.ChangedFiles()
	}
	parent := "HEAD^"
	if exec.Command("git", "-C", root, "rev-parse", "-q", "--verify", parent).Run() != nil {
		parent = emptyTree(root)
	}
	return gitx.StagedSince(root, parent)
}

// emptyTree returns the ID of the empty tree in root's object format.
func emptyTree(root string) string {
	cmd := exec.Command("git", "-C", root, "hash-object", "-t", "tree", "--stdin")
	cmd.Stdin = strings.NewReader("")
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}
