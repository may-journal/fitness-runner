package check

import (
	"cmp"
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

// newPairs pairs every doc and test file by number, using the numbers in a test file's name and tests.
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

// addTestFiles pairs each file by the numbers in its name, a requirement check by its folder number, and each test by its acceptance.
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

// judge flags each changed test whose docs stayed unchanged, unless it failed at base, and each changed doc whose tests stayed unchanged.
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

// pairErrors judges a scoped run's changed files, and a full scan or a local merge, cherry-pick or revert judges no pairs.
func pairErrors(root string, docPaths, testFiles []string, tests []reqTest) []string {
	changed := gitx.CommitChanges(root)
	if changed == nil || gitx.Replaying(root) {
		return nil
	}
	base := gitx.ScopeBase(root)
	p := newPairs(docPaths, testFiles, tests, changed)
	changed = p.withoutCommentOnly(root, base, changed)
	errs := p.judge(changed, reformats(root, base, changed), fixedAtBase(root, base))
	sort.Strings(errs)
	return errs
}

// withoutCommentOnly drops each changed test or doc whose code, comments
// aside, is the same as at base. It neither needs a partner nor counts as
// one.
func (p pairs) withoutCommentOnly(root, base string, changed []string) []string {
	var renames map[string]string
	return slices.DeleteFunc(slices.Clone(changed), func(f string) bool {
		if len(p.numbers[f]) == 0 && docNumber(f) == "" {
			return false
		}
		if renames == nil {
			renames = gitx.Renames(root, base)
		}
		from := cmp.Or(renames[f], f)
		return comments.CommentOnly(root, base, from, f)
	})
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
