// Command fitness-check-adr-structure validates that every numbered ADR under
// docs/architecture/adr follows the house template: an `# NNNN — Title` H1
// whose id matches the filename, and the sections Context, Decision, and
// Consequences in that order, with an optional Status section first. It self-
// gates — a repo with no ADRs passes with zero files. The template.md and any
// unnumbered file are not scanned, so the template itself never trips the id
// rule it documents.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
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

var (
	// h1Re matches the ADR H1 `# NNNN — Title` with an em-dash separator,
	// capturing the four-digit id.
	h1Re = regexp.MustCompile(`(?m)^# (\d{4}) — .+$`)
	// sectionRe matches every `## Heading` line, capturing the heading text.
	sectionRe = regexp.MustCompile(`(?m)^## (.+?)\s*$`)
)

// allowed is the ADR section vocabulary; Status is optional and everything
// else is required, so the accepted orderings are exactly two.
var allowed = map[string]bool{"Status": true, "Context": true, "Decision": true, "Consequences": true}

// validateADR returns one ADR's structure errors: a malformed or mismatched
// H1, and any section-set or ordering violation.
func validateADR(file, content string) []string {
	var errs []string
	errs = append(errs, headingErrors(file, content)...)
	errs = append(errs, sectionErrors(file, content)...)
	return errs
}

// headingErrors validates the H1 exists and its id matches the filename id.
func headingErrors(file, content string) []string {
	m := h1Re.FindStringSubmatch(content)
	if m == nil {
		return []string{file + ": ADR needs an H1 `# NNNN — Title` (four-digit id, em-dash separator)"}
	}
	if id := filepath.Base(file)[:4]; m[1] != id {
		return []string{fmt.Sprintf("%s: H1 id %s does not match filename id %s", file, m[1], id)}
	}
	return nil
}

// sectionErrors flags unexpected sections and any deviation from the two
// accepted section orderings.
func sectionErrors(file, content string) []string {
	got := sectionHeadings(content)
	var errs []string
	if bad := unknownSections(got); len(bad) > 0 {
		errs = append(errs, fmt.Sprintf(
			"%s: unexpected section(s) %s — an ADR uses Status (optional), Context, Decision, Consequences",
			file, strings.Join(bad, ", ")))
	}
	if msg := sequenceError(got); msg != "" {
		errs = append(errs, file+": "+msg)
	}
	return errs
}

// sectionHeadings returns the `## ` heading texts in document order.
func sectionHeadings(content string) []string {
	var out []string
	for _, m := range sectionRe.FindAllStringSubmatch(content, -1) {
		out = append(out, strings.TrimSpace(m[1]))
	}
	return out
}

// unknownSections returns the headings outside the ADR vocabulary.
func unknownSections(got []string) []string {
	var bad []string
	for _, h := range got {
		if !allowed[h] {
			bad = append(bad, h)
		}
	}
	return bad
}

// sequenceError projects the headings to the known vocabulary and reports a
// violation unless they match one of the two accepted orderings — which also
// catches a missing section, a duplicate, or Status placed after Context.
func sequenceError(got []string) string {
	joined := strings.Join(filterKnown(got), ", ")
	if joined == "Context, Decision, Consequences" ||
		joined == "Status, Context, Decision, Consequences" {
		return ""
	}
	return "sections must be Context, Decision, Consequences in order (optional Status first); found: " + joined
}

// filterKnown keeps the ADR-vocabulary headings in document order.
func filterKnown(got []string) []string {
	var out []string
	for _, h := range got {
		if allowed[h] {
			out = append(out, h)
		}
	}
	return out
}
