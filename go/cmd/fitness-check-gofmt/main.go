package check

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/golang"
	"github.com/may-journal/fitness-runner/go/internal/lang/golang/toolchain"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// batch caps the files passed to one gofmt run, keeping argv well under the
// OS limit.
const batch = 200

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "gofmt", Scope: checkkit.ScopeFiles, Comments: checkkit.CommentsJudged, TimeoutMs: 60000},
		Run:      run,
	})
}

func run(root string, _ []string) (checkkit.Result, error) {
	files := goFiles(root)
	if len(files) == 0 {
		return checkkit.Pass(0), nil
	}
	bin, err := exec.LookPath("gofmt")
	if err != nil {
		return checkkit.Result{}, toolchain.Missing()
	}
	var found []checkkit.Finding
	for start := 0; start < len(files); start += batch {
		end := min(start+batch, len(files))
		batchFound, err := unformatted(root, bin, files[start:end])
		if err != nil {
			return checkkit.Result{}, err
		}
		found = append(found, batchFound...)
	}
	return checkkit.Found(len(files), errorLine, found...), nil
}

// rule names what gofmt enforces: Go files stay as gofmt writes them.
const rule = "gofmt-formatted"

// message says what is wrong with a file gofmt would change.
const message = "not gofmt-formatted"

// fixCommand is the command that formats a file, given its path.
const fixCommand = "gofmt -w "

// syntaxRule names a file gofmt cannot parse, so it cannot format it.
const syntaxRule = "go-syntax"

// syntaxError matches the line gofmt prints on stderr for a file it cannot
// parse: path:line:column: message.
var syntaxError = regexp.MustCompile(`^(.+\.go):(\d+):(\d+): (.+)$`)

// errorLine renders a finding as `path: not gofmt-formatted (run: gofmt -w
// path)`, or a syntax error as `path:line:column: message`.
func errorLine(f checkkit.Finding) string {
	if f.Rule == syntaxRule {
		return fmt.Sprintf("%s:%d:%d: %s", f.File, f.Line, f.Column, f.Message)
	}
	return f.File + ": " + f.Message + " (run: " + f.Fix + ")"
}

// goFiles returns the repo's own in-scope .go files, leaving out fixtures.
func goFiles(root string) []string {
	var files []string
	for _, f := range walkfs.InScope(walkfs.FilesByExt(root, golang.Ext)) {
		if !toolchain.Fixture(f) {
			files = append(files, f)
		}
	}
	return files
}

// unformatted runs gofmt -l over files and turns each listed path or syntax
// error into a finding. A run
// that could not start, was killed, or failed with neither is an internal
// error, as gofmt judged nothing.
func unformatted(root, bin string, files []string) ([]checkkit.Finding, error) {
	cmd := exec.Command(bin, append([]string{"-l"}, files...)...)
	cmd.Dir = root
	cmd.Env = toolchain.CleanEnv(os.Environ())
	o := checkkit.Exec(cmd)
	if err := o.EnvErr("gofmt"); err != nil {
		return nil, err
	}
	found := append(listed(o.Stdout), syntaxErrors(o.Stderr)...)
	if o.Code != 0 && len(found) == 0 {
		return nil, checkkit.NoVerdict("gofmt -l", o.Code, o.Stderr+"\n"+o.Stdout)
	}
	return found, nil
}

// listed turns each path gofmt -l printed into an unformatted finding.
func listed(stdout string) []checkkit.Finding {
	var found []checkkit.Finding
	for _, l := range toolchain.Lines(stdout) {
		found = append(found, checkkit.Finding{File: l, Rule: rule, Message: message, Fix: fixCommand + l})
	}
	return found
}

// syntaxErrors turns each parse error gofmt printed on stderr into a
// finding at its file, line and column.
func syntaxErrors(stderr string) []checkkit.Finding {
	var found []checkkit.Finding
	for _, l := range toolchain.Lines(stderr) {
		if m := syntaxError.FindStringSubmatch(l); m != nil {
			line, _ := strconv.Atoi(m[2])
			col, _ := strconv.Atoi(m[3])
			found = append(found, checkkit.Finding{File: m[1], Line: line, Column: col, Rule: syntaxRule, Message: m[4]})
		}
	}
	return found
}
