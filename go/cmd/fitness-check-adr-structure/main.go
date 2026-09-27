// Command fitness-check-adr-structure validates that every numbered ADR under
// docs/architecture/adr follows the house template: an `# NNNN — Title` H1
// whose id matches the filename, and exactly the sections Context, Decision,
// and Consequences. The section rules reuse the shared mdtemplate mechanism
// that also backs plan-structure and pr-structure, mirroring
// docs/architecture/adr/template.md. It self-gates — a repo with no ADRs
// passes with zero files — and skips template.md and any unnumbered file, so
// the template itself never trips the id rule it documents.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mdtemplate"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "adr-structure"},
		Run:      run,
	})
}

const adrDir = "docs/architecture/adr"

// adrFileRe matches a numbered ADR filename like 0003-some-title.md. The
// template and any index file are unnumbered, so they are never scanned.
var adrFileRe = regexp.MustCompile(`^\d{4}-[a-z0-9-]+\.md$`)

// adrSpec is the ADR template shape, mirroring template.md: no pitch, and
// exactly Context, Decision, Consequences with no other sections.
var adrSpec = mdtemplate.Spec{
	Noun:    "ADR",
	NoPitch: true,
	Sections: []mdtemplate.Section{
		{Heading: "Context", Requires: mdtemplate.Prose},
		{Heading: "Decision", Requires: mdtemplate.Prose},
		{Heading: "Consequences", Requires: mdtemplate.Prose},
	},
}

func run(root string, _ []string) (checkkit.Result, error) {
	errs, n, err := scanADRs(root)
	if err != nil {
		return checkkit.Result{}, err
	}
	if len(errs) > 0 {
		return checkkit.Fail(n, errs...), nil
	}
	return checkkit.Pass(n), nil
}

// scanADRs reads every numbered ADR under root and collects its structure
// errors, returning the count of ADRs actually checked.
func scanADRs(root string) ([]string, int, error) {
	var errs []string
	n := 0
	for _, f := range walkfs.FilesByExt(root, ".md") {
		if !isADR(f) {
			continue
		}
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			return nil, 0, err
		}
		n++
		errs = append(errs, validateADR(f, string(content))...)
	}
	return errs, n, nil
}

// isADR reports whether file is a numbered ADR living in the ADR directory.
func isADR(file string) bool {
	return filepath.Dir(file) == adrDir && adrFileRe.MatchString(filepath.Base(file))
}

// h1Re matches the ADR H1 `# NNNN — Title` with an em-dash separator,
// capturing the four-digit id.
var h1Re = regexp.MustCompile(`(?m)^# (\d{4}) — .+$`)

// validateADR returns one ADR's structure errors, each prefixed with the
// file: the ADR-specific H1/id rule, then the shared section rules.
func validateADR(file, content string) []string {
	errs := headingErrors(file, content)
	errs = append(errs, mdtemplate.Validate(content, adrSpec)...)
	for i, e := range errs {
		errs[i] = file + ": " + e
	}
	return errs
}

// headingErrors validates the H1 exists and its id matches the filename id;
// the messages are unprefixed — validateADR adds the file.
func headingErrors(file, content string) []string {
	m := h1Re.FindStringSubmatch(content)
	if m == nil {
		return []string{"ADR needs an H1 `# NNNN — Title` (four-digit id, em-dash separator)"}
	}
	if id := filepath.Base(file)[:4]; m[1] != id {
		return []string{fmt.Sprintf("H1 id %s does not match filename id %s", m[1], id)}
	}
	return nil
}
