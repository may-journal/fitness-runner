package check

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
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

// numbered matches an ADR file name, NNNN-kebab-title.md, and takes its
// number.
var numbered = regexp.MustCompile(`^(\d{4})-.+\.md$`)

// superseded matches a status line that retires the decision, such as
// `Status: Superseded by ADR 0012` or a `## Status` section opening with
// Superseded.
var superseded = regexp.MustCompile(`(?im)(^[\s>*_-]*status[\s*_]*:[\s*_]*superseded\b)|(^#+\s*status\s*$\s*^[\s>*_-]*superseded\b)`)

// link matches a Markdown link's target.
var link = regexp.MustCompile(`\]\(([^)\s]+)`)

// adr is one numbered decision record.
type adr struct {
	path, number string
	retired      bool
}

func run(root string, _ []string) (checkkit.Result, error) {
	decisions, err := records(root)
	if err != nil || len(decisions) == 0 || !hasFolder(root) {
		return checkkit.Pass(0), err
	}
	if errs := judge(root, decisions); len(errs) > 0 {
		return checkkit.Fail(len(decisions), errs...), nil
	}
	return checkkit.Pass(len(decisions)), nil
}

// judge lists every pairing problem: an ADR without its check or links, a
// check without its ADR, and a change to one half alone.
func judge(root string, decisions []adr) []string {
	folders := checkFolders(root)
	var errs []string
	for _, a := range decisions {
		errs = append(errs, pairErrors(root, a, folders)...)
	}
	errs = append(errs, orphans(decisions, folders)...)
	return append(errs, changeErrors(decisions, checkkit.DiffFiles())...)
}

// records reads every tracked numbered ADR in a folder named adr.
func records(root string) ([]adr, error) {
	var out []adr
	for _, f := range walkfs.FilesByExt(root, ".md") {
		m := numbered.FindStringSubmatch(path.Base(f))
		if m == nil || path.Base(path.Dir(f)) != "adr" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			return nil, checkkit.Internal(err)
		}
		out = append(out, adr{path: f, number: m[1], retired: superseded.Match(data)})
	}
	return out, nil
}

// hasFolder reports whether the repo keeps a folder of its own checks.
func hasFolder(root string) bool {
	info, err := os.Stat(filepath.Join(root, repocheck.Folder))
	return err == nil && info.IsDir()
}

// checkFolders maps each ADR number with a fitness/adr-NNNN folder to the
// folder's path.
func checkFolders(root string) map[string]string {
	entries, _ := os.ReadDir(filepath.Join(root, repocheck.Folder))
	folders := map[string]string{}
	for _, e := range entries {
		if n, ok := adrNumber(e.Name()); ok && e.IsDir() {
			folders[n] = repocheck.Folder + "/" + e.Name()
		}
	}
	return folders
}

// adrNumber reads the number of a check folder named adr-NNNN.
func adrNumber(name string) (string, bool) {
	n, ok := strings.CutPrefix(name, "adr-")
	return n, ok && len(n) == 4 && strings.Trim(n, "0123456789") == ""
}

// pairErrors judges one ADR: it needs its check, unless retired, and the
// two link each other.
func pairErrors(root string, a adr, folders map[string]string) []string {
	if a.retired {
		return nil
	}
	folder, ok := folders[a.number]
	if !ok {
		return []string{fmt.Sprintf("%s: ADR %s has no check; add %s/adr-%s/ with a README.md that links this ADR and a check that proves it, or mark the ADR Status: superseded", a.path, a.number, repocheck.Folder, a.number)}
	}
	readme := folder + "/" + repocheck.Readme
	var errs []string
	if !links(root, a.path, folder) {
		errs = append(errs, fmt.Sprintf("%s: link its check %s, so a reader finds what keeps the decision", a.path, readme))
	}
	if !links(root, readme, a.path) {
		errs = append(errs, fmt.Sprintf("%s: link %s, so the check names the decision it keeps", readme, a.path))
	}
	return errs
}

// links reports whether the Markdown file from holds a relative link to
// target, or to a file inside it.
func links(root, from, target string) bool {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(from)))
	if err != nil {
		return false
	}
	for _, m := range link.FindAllStringSubmatch(string(data), -1) {
		dest, _, _ := strings.Cut(m[1], "#")
		resolved := path.Join(path.Dir(from), dest)
		if resolved == target || strings.HasPrefix(resolved, target+"/") {
			return true
		}
	}
	return false
}

// orphans fails each ADR check folder whose number no ADR has.
func orphans(decisions []adr, folders map[string]string) []string {
	known := map[string]bool{}
	for _, a := range decisions {
		known[a.number] = true
	}
	var errs []string
	for n, folder := range folders {
		if !known[n] {
			errs = append(errs, fmt.Sprintf("%s: no ADR %s exists; delete the check or give it its ADR's number", folder, n))
		}
	}
	sort.Strings(errs)
	return errs
}
