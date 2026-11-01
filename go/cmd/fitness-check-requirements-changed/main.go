package check

import (
	"fmt"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/comments"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "requirements-changed", Scope: checkkit.ScopeGit, Comments: checkkit.CommentsStripped, ContextInlineArg: "--message"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	if skips(root) {
		return checkkit.Pass(0), nil
	}
	scope, err := diffScope(root)
	if err != nil {
		return checkkit.Result{}, err
	}
	changes, err := readChanges(root, scope.args)
	if err != nil {
		return checkkit.Result{}, err
	}
	tests, err := scanTests(root)
	if err != nil {
		return checkkit.Result{}, err
	}
	return judgeCode(root, scope.base, tests, changes), nil
}

// judgeCode judges changes once comment-only edits are set aside. With no
// product code changed, it passes before comparing any file.
func judgeCode(root, base string, tests testSet, changes []change) checkkit.Result {
	if len(productChanges(root, tests, changes)) == 0 {
		return checkkit.Pass(0)
	}
	return judge(root, tests, withoutCommentOnly(root, base, changes))
}

// skips reports a run with nothing to judge: a full scan, a replay, or a
// repo that has no requirement docs yet.
func skips(root string) bool {
	return gitx.CommitChanges(root) == nil || gitx.Replaying(root) || len(requirementDocs(root)) == 0
}

// judge fails each changed product file whose test or doc did not change, and each deleted one whose doc and tests stay.
func judge(root string, tests testSet, changes []change) checkkit.Result {
	product := productChanges(root, tests, changes)
	changed := changedSet(changes)
	var errs []string
	for _, c := range product {
		errs = append(errs, c.problem(root, tests, changed)...)
	}
	if len(errs) > 0 {
		return checkkit.Fail(len(product), errs...)
	}
	return checkkit.Pass(len(product))
}

// withoutCommentOnly drops each edit whose code, comments aside, matches base, since it changes no behavior and needs no test or doc.
func withoutCommentOnly(root, base string, changes []change) []change {
	return slices.DeleteFunc(slices.Clone(changes), func(c change) bool {
		return !c.deleted && comments.CommentOnly(root, base, c.from, c.path)
	})
}

// problem names what is wrong with one product change: an edit lacking a test or doc, or a deletion leaving them behind.
func (c change) problem(root string, tests testSet, changed map[string]bool) []string {
	if c.deleted {
		return leftBehind(root, c.path, tests, changed)
	}
	if what, which := missing(tests.anyTest(changed), anyDoc(changed)); what != "" {
		return []string{fmt.Sprintf("%s changed without %s; change %s with it", c.path, what, which)}
	}
	return nil
}

// missing names the half a change lacks, and the pronoun for it, or "" when
// it has both.
func missing(tested, documented bool) (what, which string) {
	switch {
	case !tested && !documented:
		return "a test and a requirement doc", "both"
	case !tested:
		return "a test", "one"
	case !documented:
		return "a requirement doc", "one"
	}
	return "", ""
}

// changedSet is the set of paths that changed, a rename by its new name.
func changedSet(changes []change) map[string]bool {
	set := map[string]bool{}
	for _, c := range changes {
		set[c.path] = true
	}
	return set
}

// anyDoc reports whether a requirement doc changed.
func anyDoc(changed map[string]bool) bool {
	for f := range changed {
		if docNumber(f) != "" {
			return true
		}
	}
	return false
}

// joinList names files as a reader lists them.
func joinList(files []string) string {
	return strings.Join(files, ", ")
}
