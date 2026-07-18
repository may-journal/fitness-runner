// Command fitness-check-markdown-front-matter validates that every markdown
// file opens with front matter naming fitnessFunctions or
// relatedConfigurations, and that each entry resolves — relative to the
// markdown file's directory — to an existing path inside the repo, or names
// an enabled check — the Go port of the markdown-front-matter check. The TS
// check received registeredCheckNames from the runner context; here the same
// list arrives as FITNESS_ENABLED_CHECKS (empty when run standalone).
package main

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/mdx"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "markdown-front-matter"},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	return runCheck(root, checkkit.EnabledChecks())
}

// runCheck walks every markdown file under root and validates its front
// matter; registered holds the check names an entry may use instead of a
// path (the runner's enabled set).
func runCheck(root string, registered []string) (checkkit.Result, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return checkkit.Result{}, err
	}
	errs, fileCount, err := walkfs.ScanFiles(root, []string{".md"}, func(file, content string) []string {
		mdDir := filepath.Dir(filepath.Join(absRoot, filepath.FromSlash(file)))
		return validateFile(file, content, absRoot, mdDir, registered)
	})
	if err != nil {
		return checkkit.Result{}, err
	}
	if len(errs) > 0 {
		return checkkit.Fail(fileCount, errs...), nil
	}
	return checkkit.Pass(fileCount), nil
}

var (
	arrayRe      = regexp.MustCompile(`(?:fitnessFunctions|relatedConfigurations):\s*\[([^\]]*)\]`)
	emptyArrayRe = regexp.MustCompile(`(fitnessFunctions|relatedConfigurations):\s*\[\s*\]`)
)

// validateFile returns one markdown file's error messages: missing or
// keyless front matter is a single error, and empty arrays short-circuit
// path validation, exactly like the TS validateFile.
func validateFile(file, content, root, mdDir string, registered []string) []string {
	inner, ok := mdx.FrontMatter(content)
	if !ok || !hasRequiredKeys(inner) {
		return []string{file + ": missing front matter with fitnessFunctions or relatedConfigurations"}
	}
	if empty := emptyArrayErrors(file, inner); len(empty) > 0 {
		return empty
	}
	var errs []string
	for _, entry := range frontMatterPaths(inner) {
		if msg := validatePath(file, entry, root, mdDir, registered); msg != "" {
			errs = append(errs, msg)
		}
	}
	return errs
}

// hasRequiredKeys reports whether the front matter body mentions
// fitnessFunctions or relatedConfigurations (substring test, like the TS
// hasRequiredFrontMatter).
func hasRequiredKeys(inner string) bool {
	return strings.Contains(inner, "fitnessFunctions") || strings.Contains(inner, "relatedConfigurations")
}

// emptyArrayErrors flags every empty fitnessFunctions or
// relatedConfigurations array in the front matter body.
func emptyArrayErrors(file, inner string) []string {
	var errs []string
	for _, m := range emptyArrayRe.FindAllStringSubmatch(inner, -1) {
		errs = append(errs, file+": "+m[1]+" must not be an empty array")
	}
	return errs
}

// frontMatterPaths parses the fitnessFunctions and relatedConfigurations
// array entries out of the front matter body, trimming whitespace and one
// pair of surrounding quotes — the port of getFrontMatterPaths.
func frontMatterPaths(inner string) []string {
	var paths []string
	for _, m := range arrayRe.FindAllStringSubmatch(inner, -1) {
		body := strings.TrimSpace(m[1])
		if body == "" {
			continue
		}
		for _, entry := range strings.Split(body, ",") {
			paths = append(paths, stripQuotes(strings.TrimSpace(entry)))
		}
	}
	return paths
}

// stripQuotes drops one leading and one trailing single or double quote,
// mirroring the TS replace(/^['"]|['"]$/g, ”).
func stripQuotes(s string) string {
	if len(s) > 0 && (s[0] == '\'' || s[0] == '"') {
		s = s[1:]
	}
	if len(s) > 0 && (s[len(s)-1] == '\'' || s[len(s)-1] == '"') {
		s = s[:len(s)-1]
	}
	return s
}

// validatePath returns the error message for one front matter entry, or ""
// when it is valid: external links and anchors are exempt, registered check
// names pass, and everything else must resolve from the markdown file's
// directory to an existing path that stays inside root.
func validatePath(file, entry, root, mdDir string, registered []string) string {
	if isExternalOrAnchor(entry) || slices.Contains(registered, entry) {
		return ""
	}
	target := resolveFrom(mdDir, entry)
	if !strings.HasPrefix(target, root) {
		return file + ": front matter path escapes repo: " + entry
	}
	if _, err := os.Stat(target); err != nil {
		return file + ": front matter path missing: " + entry
	}
	return ""
}

// isExternalOrAnchor reports whether the entry is an http(s) link, an
// in-page anchor, or a mailto link — all skipped from validation.
func isExternalOrAnchor(entry string) bool {
	return strings.HasPrefix(entry, "http") || strings.HasPrefix(entry, "#") || strings.HasPrefix(entry, "mailto:")
}

// resolveFrom mirrors Node's path.resolve(dir, p): an absolute p stands
// alone, anything else joins onto dir; either way the result is cleaned.
func resolveFrom(dir, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(dir, p)
}
