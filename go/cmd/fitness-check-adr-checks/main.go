package check

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/adr"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/repocheck"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "adr-checks", Scope: checkkit.ScopeFiles},
		Run:      run,
	})
}

// record is one decision's check folder: its README and the ADR issue the
// README links.
type record struct {
	folder, readme, issue string
	retired               bool
}

func run(root string, _ []string) (checkkit.Result, error) {
	if !hasFolder(root) {
		return checkkit.Pass(0), nil
	}
	decisions, err := records(root)
	if err != nil {
		return checkkit.Pass(0), err
	}
	allow, _, _ := adr.LoadAllow(root) // adr-as-issue reports a bad config
	errs := append(linkErrors(decisions, allow), changeErrors(root, decisions, checkkit.DiffFiles())...)
	if len(errs) > 0 {
		return checkkit.Fail(len(decisions), errs...), nil
	}
	return checkkit.Pass(len(decisions)), nil
}

// hasFolder reports whether the repo keeps a folder of its own checks.
func hasFolder(root string) bool {
	info, err := os.Stat(filepath.Join(root, repocheck.Folder))
	return err == nil && info.IsDir()
}

// records reads the README of every fitness/adr-NNNN folder. A folder
// without one reads as empty text, which links no issue.
func records(root string) ([]record, error) {
	entries, err := os.ReadDir(filepath.Join(root, repocheck.Folder))
	if err != nil {
		return nil, checkkit.Internal(err)
	}
	var out []record
	for _, e := range entries {
		if !e.IsDir() || !adrFolder(e.Name()) {
			continue
		}
		folder := repocheck.Folder + "/" + e.Name()
		readme := folder + "/" + repocheck.Readme
		data, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(readme)))
		out = append(out, record{folder: folder, readme: readme, issue: adr.FirstIssueURL(string(data)), retired: repocheck.Retired(string(data))})
	}
	return out, nil
}

// adrFolder reports whether a check folder is named adr-NNNN.
func adrFolder(name string) bool {
	n, ok := strings.CutPrefix(name, repocheck.ADRPrefix)
	return ok && len(n) == 4 && strings.Trim(n, "0123456789") == ""
}

// linkErrors fails each ADR folder whose README links no ADR issue, so each
// check names its decision. An adrAsIssue.allow glob lets a README wait.
func linkErrors(decisions []record, allow adr.Allow) []string {
	var errs []string
	for _, r := range decisions {
		if r.issue == "" && allow.Covering(r.readme) == "" {
			errs = append(errs, fmt.Sprintf("%s: links no ADR issue; put the full URL of the `ADR` issue this check proves here, and run `fitness adr-migrate` to move a decision still in the file", r.readme))
		}
	}
	return errs
}
