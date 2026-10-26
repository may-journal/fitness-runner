package check

import (
	"fmt"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/repocheck"
	"github.com/may-journal/fitness-runner/go/internal/requirement"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// reqPrefix starts the path of every file in a requirement's check folder,
// fitness/req-NNNN/.
const reqPrefix = repocheck.Folder + "/req-"

// reqCheckNumber is the requirement number of a file in a fitness/req-NNNN
// folder, or "".
func reqCheckNumber(file string) string {
	rest, ok := strings.CutPrefix(file, reqPrefix)
	n, _, nested := strings.Cut(rest, "/")
	if !ok || !nested || len(n) != 4 || strings.Trim(n, "0123456789") != "" {
		return ""
	}
	return n
}

// reqChecks lists the tracked files of every requirement check folder, and
// the requirement numbers they prove.
func reqChecks(root string) (files []string, proven map[string]bool) {
	proven = map[string]bool{}
	for _, f := range walkfs.FilesByExt(root, "") {
		if n := reqCheckNumber(f); n != "" {
			files = append(files, f)
			proven[n] = true
		}
	}
	return files, proven
}

// orphanChecks fails each requirement check folder whose number no doc
// defines.
func orphanChecks(docs []requirement.Document, proven map[string]bool) []string {
	defined := map[string]bool{}
	for _, d := range docs {
		defined[d.ID] = true
	}
	var errs []string
	for _, n := range sortedKeys(proven) {
		if !defined[n] {
			errs = append(errs, fmt.Sprintf("%s%s proves requirement %s, which no doc defines; write the doc or delete the check", reqPrefix, n, n))
		}
	}
	return errs
}
