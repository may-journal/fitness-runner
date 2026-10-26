package check

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/frontmatter"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "no-fitness-front-matter", Scope: checkkit.ScopeFiles},
		Run:      run,
	})
}

// message is what each finding says about a file that holds a fitness key.
const message = "front matter holds fitnessFunctions or relatedConfigurations, which no check reads; run `fitness strip-front-matter`"

// rule names what the check enforces: no fitness keys in front matter.
const rule = "no-fitness-front-matter"

// fix says how to pass: let the fixer delete the keys.
const fix = "run `fitness strip-front-matter` from the repo root and commit the result"

// run judges every in-scope .md file under root, test fixtures left out.
func run(root string, _ []string) (checkkit.Result, error) {
	var found []checkkit.Finding
	judged := 0
	for _, file := range walkfs.InScope(walkfs.FilesByExt(root, ".md")) {
		if fixture(file) {
			continue
		}
		judged++
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			return checkkit.Result{}, err
		}
		if frontmatter.HasFitnessKeys(string(raw)) {
			found = append(found, checkkit.Finding{File: file, Rule: rule, Fix: fix, Message: message})
		}
	}
	return checkkit.Found(judged, errorLine, found...), nil
}

// fixture reports a slash-separated path under a testdata folder or a
// tests/examples folder, where fixture repos keep blocks as test data.
func fixture(rel string) bool {
	dir := "/" + rel
	return strings.Contains(dir, "/testdata/") || strings.Contains(dir, "/tests/examples/")
}

// errorLine renders a finding as "<path>: <message>".
func errorLine(f checkkit.Finding) string {
	return f.File + ": " + f.Message
}
