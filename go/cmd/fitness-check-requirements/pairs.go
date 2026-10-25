package check

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
	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/langs"
)

// numberedDoc matches a requirement doc file: 0014-kebab-title.md.
var numberedDoc = regexp.MustCompile(`^(\d{4})-.+\.md$`)

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
func newPairs(docPaths, testFiles []string, tests []reqTest, changed []string) pairs {
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
func (p pairs) addTestFiles(files []string, tests []reqTest) {
	for _, f := range files {
		for _, n := range lang.TestFileNumbers(langs.All, f) {
			p.addTest(n, f)
		}
	}
	for _, t := range tests {
		if t.ID != "" {
			p.addTest(t.ID[:4], t.File)
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

// judge flags each changed requirement test whose docs all stayed
// unchanged, unless fixed reports it failed this check at the base, and each
// changed doc whose tests all stayed unchanged. A reformatted doc needs no
// test change, yet still counts as changed for its tests.
func (p pairs) judge(changed []string, reformatted map[string]bool, fixed func(string) bool) []string {
	isChanged := map[string]bool{}
	for _, f := range changed {
		isChanged[f] = true
	}
	var errs []string
	for _, f := range changed {
		errs = append(errs, p.unpairedTest(f, isChanged, fixed)...)
		if n := docNumber(f); n != "" && !reformatted[f] {
			errs = append(errs, unpaired(f, []string{n}, p.tests, "requirement test", isChanged)...)
		}
	}
	return errs
}

// unpairedTest flags file when it proves requirements whose docs all stayed
// unchanged, unless it failed this check at the base.
func (p pairs) unpairedTest(file string, changed map[string]bool, fixed func(string) bool) []string {
	missing := unpaired(file, p.numbers[file], p.docs, "requirement doc", changed)
	if len(missing) == 0 || fixed(file) {
		return nil
	}
	return missing
}

// unpaired names file and its missing half when none of its partners
// changed with it. A number with no partner is ownership's concern.
func unpaired(file string, numbers []string, partners map[string][]string, half string, changed map[string]bool) []string {
	missing := unchangedPartners(numbers, partners, changed)
	if len(missing) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("%s changed without its %s %s; change both together", file, half, strings.Join(missing, " or "))}
}

// unchangedPartners lists the partner files of numbers when none of them
// changed; nil otherwise.
func unchangedPartners(numbers []string, partners map[string][]string, changed map[string]bool) []string {
	var all []string
	for _, n := range numbers {
		all = append(all, partners[n]...)
	}
	if slices.ContainsFunc(all, func(f string) bool { return changed[f] }) {
		return nil
	}
	return all
}

// pairErrors judges a scoped run's changed files: staged at commit time, or
// a pull request's diff against the current merge base with its base branch.
// A full scan, and a local merge, cherry-pick, or revert, judge no pairs.
func pairErrors(root string, docPaths, testFiles []string, tests []reqTest) []string {
	changed := commitChanges(root)
	if changed == nil || replaying(root) {
		return nil
	}
	base := baseRev(root)
	errs := newPairs(docPaths, testFiles, tests, changed).judge(changed, reformats(root, base, changed), fixedAtBase(root, base))
	sort.Strings(errs)
	return errs
}

// fixedAtBase reports whether a changed test file failed this check at the
// base commit, so the change fixes it. The base is judged once, and only
// when a test file needs it.
func fixedAtBase(root, base string) func(string) bool {
	var failed map[string]bool
	return func(file string) bool {
		if failed == nil {
			failed = failedAtBase(root, base)
		}
		return failed[file]
	}
}

// replayHeads are the refs git keeps while a commit replays another one.
var replayHeads = []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD"}

// replaying reports whether a local commit replays changes judged where they
// were first made, including an amend of a merge commit, whose second parent
// brought them. A pull request run still judges its whole diff.
func replaying(root string) bool {
	if os.Getenv("GITHUB_BASE_REF") != "" {
		return false
	}
	refs := replayHeads
	if os.Getenv("FITNESS_COMMIT_AMEND") == "1" {
		refs = append(slices.Clone(refs), "HEAD^2")
	}
	for _, ref := range refs {
		if exec.Command("git", "-C", root, "rev-parse", "-q", "--verify", ref).Run() == nil {
			return true
		}
	}
	return false
}

// commitChanges is what the commit or pull request changes: its diff, nil for
// a full scan or a `--dir` folder, or for an amend everything staged since
// the amended commit's parent.
func commitChanges(root string) []string {
	if os.Getenv("FITNESS_COMMIT_AMEND") != "1" {
		return checkkit.DiffFiles()
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
