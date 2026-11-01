package check

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/comments"
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

// addTestFiles pairs each file by the numbers in its name, or a file in a
// fitness/req-NNNN check by that number, then each test named for an
// acceptance by its number.
func (p pairs) addTestFiles(files []string, tests []reqTest) {
	for _, f := range files {
		for _, n := range fileNumbers(f) {
			p.addTest(n, f)
		}
	}
	for _, t := range tests {
		if t.ID != "" {
			p.addTest(t.ID[:4], t.File)
		}
	}
}

// fileNumbers lists the requirement numbers a file proves by its path: a
// requirement check's number, or those its test file name carries.
func fileNumbers(f string) []string {
	if n := reqCheckNumber(f); n != "" {
		return []string{n}
	}
	return lang.TestFileNumbers(langs.All, f)
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
// unchanged, unless fixed excuses it. It flags each changed doc whose tests
// all stayed unchanged. A reformatted doc needs no test change, yet still
// counts as changed for its tests, as an excused test does for its docs.
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
// unchanged, unless fixed excuses it.
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
	changed := gitx.CommitChanges(root)
	if changed == nil || gitx.Replaying(root) {
		return nil
	}
	base := gitx.ScopeBase(root)
	errs := newPairs(docPaths, testFiles, tests, changed).judge(changed, reformats(root, base, changed), excused(root, base))
	sort.Strings(errs)
	return errs
}

// excused reports whether a changed test file needs no doc change. Its code
// may be the same with comments set aside, or it may fix a failure at base.
func excused(root, base string) func(string) bool {
	fixed := fixedAtBase(root, base)
	return func(file string) bool {
		return comments.CommentOnly(root, base, file, file) || fixed(file)
	}
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
