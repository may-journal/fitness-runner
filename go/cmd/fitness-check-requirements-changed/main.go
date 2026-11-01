package check

import (
	"fmt"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "requirements-changed", Scope: checkkit.ScopeGit, ContextInlineArg: "--message"},
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
	return judge(root, changes)
}

// skips reports a run with nothing to judge: a full scan, a replay, or a
// repo that has no requirement docs yet.
func skips(root string) bool {
	return gitx.CommitChanges(root) == nil || gitx.Replaying(root) || len(requirementDocs(root)) == 0
}

// judge fails each changed product file whose test or requirement doc did
// not change with it.
// It also fails each deleted one whose doc and tests stay behind.
func judge(root string, changes []change) (checkkit.Result, error) {
	tests, err := scanTests(root)
	if err != nil {
		return checkkit.Result{}, err
	}
	product := productChanges(root, tests, changes)
	changed := changedSet(changes)
	var errs []string
	for _, c := range product {
		errs = append(errs, c.problem(root, tests, changed)...)
	}
	if len(errs) > 0 {
		return checkkit.Fail(len(product), errs...), nil
	}
	return checkkit.Pass(len(product)), nil
}

// problem is what is wrong with one product change.
// It is an edit with no test or doc beside it, or a deletion that leaves its
// doc and tests behind.
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
