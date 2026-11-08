package check

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/may-journal/fitness-runner/go/internal/adr"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "adr-as-issue", Scope: checkkit.ScopeFiles},
		Run:      run,
	})
}

// toIssue is where every ADR file belongs instead.
const toIssue = "open an `ADR` issue instead; `fitness adr-migrate` opens one per file and points each link at it"

func run(root string, _ []string) (checkkit.Result, error) {
	allow, problems, err := adr.LoadAllow(root)
	if err != nil {
		return checkkit.Fail(0, err.Error()), nil
	}
	files, err := walkfs.AllFiles(root)
	if err != nil {
		return checkkit.Fail(0, "cannot list the repo's files: "+err.Error()), nil
	}
	errs := append(problems, judgeAll(root, files, allow)...)
	if len(errs) == 0 {
		return checkkit.Pass(len(files)), nil
	}
	return checkkit.Fail(len(files), errs...), nil
}

// judgeAll names each ADR file no allow glob covers, then each allow glob
// that covers no ADR file.
func judgeAll(root string, files []string, allow adr.Allow) []string {
	var errs []string
	used := map[string]bool{}
	for _, f := range files {
		if rule, ok := ruleOf(root, f); ok {
			errs = append(errs, judge(f, rule, allow, used)...)
		}
	}
	for _, glob := range allow.Globs {
		if !used[glob] {
			errs = append(errs, fmt.Sprintf("adrAsIssue.allow path %q matches no ADR file; remove it", glob))
		}
	}
	return errs
}

// ruleOf reads rel and says which rule makes it an ADR file, if any.
func ruleOf(root, rel string) (string, bool) {
	if !adr.MayBe(rel) {
		return "", false
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return "", false
	}
	return adr.Rule(rel, string(content))
}

// judge is the error for ADR file rel, or none when an allow glob covers
// it, which it marks used.
func judge(rel, rule string, allow adr.Allow, used map[string]bool) []string {
	if glob := allow.Covering(rel); glob != "" {
		used[glob] = true
		return nil
	}
	return []string{rel + ": " + rule + "; " + toIssue}
}
