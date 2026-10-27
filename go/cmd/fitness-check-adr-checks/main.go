package check

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/repocheck"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "adr-checks", Scope: checkkit.ScopeFiles},
		Run:      run,
	})
}

// numbered matches an ADR file kept apart from its check, named
// NNNN-kebab-title.md, and takes its number.
var numbered = regexp.MustCompile(`^(\d{4})-.+\.md$`)

// sections are the parts of an ADR that state its decision, in the order
// they appear.
var sections = []string{"Context", "Decision", "Consequences"}

// adr is one decision record: the README of a fitness/adr-NNNN folder.
type adr struct {
	folder, readme, text string
	retired              bool
}

func run(root string, _ []string) (checkkit.Result, error) {
	if !hasFolder(root) {
		return checkkit.Pass(0), nil
	}
	decisions, err := records(root)
	if err != nil {
		return checkkit.Pass(0), err
	}
	apart := strays(root)
	judged := len(decisions) + len(apart)
	errs := append(apart, shapeErrors(decisions)...)
	errs = append(errs, changeErrors(root, decisions, checkkit.DiffFiles())...)
	if len(errs) > 0 {
		return checkkit.Fail(judged, errs...), nil
	}
	return checkkit.Pass(judged), nil
}

// hasFolder reports whether the repo keeps a folder of its own checks.
func hasFolder(root string) bool {
	info, err := os.Stat(filepath.Join(root, repocheck.Folder))
	return err == nil && info.IsDir()
}

// records reads the README of every fitness/adr-NNNN folder. A folder
// without one reads as empty text, which holds no decision.
func records(root string) ([]adr, error) {
	entries, err := os.ReadDir(filepath.Join(root, repocheck.Folder))
	if err != nil {
		return nil, checkkit.Internal(err)
	}
	var out []adr
	for _, e := range entries {
		if !e.IsDir() || !adrFolder(e.Name()) {
			continue
		}
		folder := repocheck.Folder + "/" + e.Name()
		readme := folder + "/" + repocheck.Readme
		data, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(readme)))
		out = append(out, adr{folder: folder, readme: readme, text: string(data), retired: repocheck.Retired(string(data))})
	}
	return out, nil
}

// adrFolder reports whether a check folder is named adr-NNNN.
func adrFolder(name string) bool {
	n, ok := strings.CutPrefix(name, repocheck.ADRPrefix)
	return ok && len(n) == 4 && strings.Trim(n, "0123456789") == ""
}

// strays fails each numbered ADR kept in a folder named adr, apart from its
// check: its one home is the README of its fitness/adr-NNNN folder.
func strays(root string) []string {
	var errs []string
	for _, f := range walkfs.FilesByExt(root, ".md") {
		m := numbered.FindStringSubmatch(path.Base(f))
		if m == nil || path.Base(path.Dir(f)) != "adr" || strings.HasPrefix(f, repocheck.Folder+"/") {
			continue
		}
		errs = append(errs, fmt.Sprintf("%s: ADR %s lives apart from its check; move it into %s/%s%s/%s, beside the check that keeps it, and delete this file", f, m[1], repocheck.Folder, repocheck.ADRPrefix, m[1], repocheck.Readme))
	}
	return errs
}

// shapeErrors fails each ADR folder whose README lacks a section of the
// decision, so every adr-NNNN README is the ADR itself.
func shapeErrors(decisions []adr) []string {
	var errs []string
	for _, a := range decisions {
		if missing := missingSections(a.text); len(missing) > 0 {
			errs = append(errs, fmt.Sprintf("%s: is no ADR, as it lacks %s; write the decision this check keeps under ## Context, ## Decision and ## Consequences", a.readme, strings.Join(missing, " and ")))
		}
	}
	return errs
}

// missingSections lists each ADR section heading text lacks.
func missingSections(text string) []string {
	var missing []string
	for _, s := range sections {
		if _, ok := section(text, s); !ok {
			missing = append(missing, "## "+s)
		}
	}
	return missing
}

// section is the body of text's `## name` section, up to the next heading
// of its level or above, and whether text has one.
func section(text, name string) (string, bool) {
	var body []string
	in, found := false, false
	for _, line := range strings.Split(text, "\n") {
		if title, ok := heading(line); ok {
			in = strings.EqualFold(title, name)
			found = found || in
		} else if in {
			body = append(body, line)
		}
	}
	return strings.TrimSpace(strings.Join(body, "\n")), found
}

// heading reads a line that opens a section: a `##` heading's title, or ""
// for the `#` title, which opens none.
func heading(line string) (string, bool) {
	if title, ok := strings.CutPrefix(line, "## "); ok {
		return strings.TrimSpace(title), true
	}
	return "", strings.HasPrefix(line, "# ")
}
