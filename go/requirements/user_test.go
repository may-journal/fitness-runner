// Package requirements proves docs/requirements as a user meets them: each test
// runs the installer at FITNESS_INSTALL in a copied example repo.
package requirements

import (
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/may-journal/fitness-runner/go/internal/acceptance"
)

// cases is an acceptance's table: each case is one situation the acceptance
// covers, named by its Given.
type cases acceptance.Cases

// run runs each case in parallel as its own t.Run of the acceptance's test.
func (cs cases) run(t *testing.T) {
	t.Helper()
	acceptance.Cases(cs).Run(t)
}

// runAlone runs the cases like run, but before the parallel acceptances start.
func (cs cases) runAlone(t *testing.T) {
	t.Helper()
	acceptance.Cases(cs).RunAlone(t)
}

// The suite's helpers live in the acceptance package, which each language
// package's suite shares.
var (
	example   = acceptance.Example
	commit    = acceptance.Commit
	write     = acceptance.Write
	git       = acceptance.Git
	mustDo    = acceptance.MustDo
	fitness   = acceptance.Fitness
	piped     = acceptance.Piped
	execute   = acceptance.Execute
	started   = acceptance.Started
	userEnv   = acceptance.UserEnv
	localAPI  = acceptance.LocalAPI
	installer = acceptance.Installer
	ansi      = acceptance.Ansi
)

// placeRunner hard-links the installed runner at dst, so no parallel test's fork
// inherits an open write handle. Where linking fails, it writes, closes and
// renames a temporary file into place.
func placeRunner(t *testing.T, installed, dst string) {
	t.Helper()
	if os.Link(installed, dst) == nil {
		return
	}
	raw, err := os.ReadFile(installed)
	mustDo(t, err)
	tmp := dst + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755)
	mustDo(t, err)
	_, err = f.Write(raw)
	mustDo(t, errors.Join(err, f.Close()))
	mustDo(t, os.Rename(tmp, dst))
}

// read returns text as a person reads it, table borders dropped and wrapped
// lines joined into one.
func read(text string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(text, "│", " ")), " ")
}

// sees asserts the user saw every want and got exit code code.
func sees(t *testing.T, out string, gotCode, code int, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(read(out), read(w)) {
			t.Errorf("output is missing %q:\n%s", w, out)
		}
	}
	if gotCode != code {
		t.Errorf("exit code = %d, want %d:\n%s", gotCode, code, out)
	}
}

// resultRow matches a results-table line: name, status, files, time.
var resultRow = regexp.MustCompile(`^│ (\S+)\s+│ (passed|failed|info|skipped)\s+│`)

// rowCells returns the check cell of each results-table row, top to bottom.
func rowCells(out string) []string {
	var cells []string
	for _, line := range strings.Split(out, "\n") {
		if m := resultRow.FindStringSubmatch(line); m != nil {
			cells = append(cells, m[1])
		}
	}
	return cells
}

// row returns the results-table line for check, or "". The table shortens
// long names with "…", as a reader sees them.
func row(out, check string) string {
	for _, line := range strings.Split(out, "\n") {
		if m := resultRow.FindStringSubmatch(line); m != nil && names(m[1], check) {
			return line
		}
	}
	return ""
}

// names reports whether a table cell shows check, whole or shortened.
func names(cell, check string) bool {
	short, cut := strings.CutSuffix(cell, "…")
	return cell == check || cut && strings.HasPrefix(check, short)
}

// passedWithNoFiles asserts check's row shows a pass that judged 0 files.
func passedWithNoFiles(t *testing.T, out, check string) {
	t.Helper()
	if !regexp.MustCompile(`│ passed\s+│ 0\s+│`).MatchString(row(out, check)) {
		t.Errorf("%s must pass with 0 files, got row %q", check, row(out, check))
	}
}

// passedWith asserts check's row shows a pass that judged files.
func passedWith(t *testing.T, out, check, files string) {
	t.Helper()
	if !regexp.MustCompile(`│ passed\s+│ ` + files + `\s+│`).MatchString(row(out, check)) {
		t.Errorf("%s must pass with %s files, got row %q", check, files, row(out, check))
	}
}
