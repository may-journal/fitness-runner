// Package actions gives a GitHub Actions step typed access to its files and
// to the command-line tools it calls.
//
// A Step writes only the outputs it declares, so the action that runs it can
// declare each one by name. The CLI helpers run gh, git and other programs
// as a shell step does. They capture stdout without its trailing newlines,
// as `$(...)` does, and pass stderr through to the log.
package actions

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/may-journal/fitness-runner/go/internal/report"
)

// Step is one workflow step and the outputs it may write.
type Step struct {
	// Outputs are the step output keys the step declares.
	Outputs []string
}

// Output appends key=value to the step output file, GITHUB_OUTPUT. It
// refuses a key the step does not declare.
func (s Step) Output(key, value string) error {
	if !slices.Contains(s.Outputs, key) {
		return fmt.Errorf("step output %q is not declared", key)
	}
	return appendLine("GITHUB_OUTPUT", key+"="+value)
}

// Env appends key=value to GITHUB_ENV, so later steps of the job see it.
func Env(key, value string) error {
	return appendLine("GITHUB_ENV", key+"="+value)
}

// Path appends dir to GITHUB_PATH, so later steps of the job find its programs.
func Path(dir string) error {
	return appendLine("GITHUB_PATH", dir)
}

// Summary appends a line to the job summary, GITHUB_STEP_SUMMARY.
func Summary(line string) error {
	return appendLine("GITHUB_STEP_SUMMARY", line)
}

// appendLine appends line and a newline to the file the variable name names.
func appendLine(name, line string) error {
	path := os.Getenv(name)
	if path == "" {
		return fmt.Errorf("%s is not set", name)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, err = f.WriteString(line + "\n")
	return errors.Join(err, f.Close())
}

// Notice prints a notice annotation.
func Notice(message string) { annotate("notice", message) }

// Warning prints a warning annotation.
func Warning(message string) { annotate("warning", message) }

// Error prints an error annotation. It does not end the step; return Failed
// for that.
func Error(message string) { annotate("error", message) }

// annotate prints the workflow command kind with message, escaped so a
// multi-line message stays one annotation.
func annotate(kind, message string) {
	fmt.Printf("::%s::%s\n", kind, report.EscapeData(message))
}

// Failed is the error a step returns once it has annotated its own failure,
// so the program exits 1 without printing it again.
var Failed = errors.New("the step failed")

// Skip prints message as a notice and returns nil: the step passes without
// doing its work.
func Skip(message string) error {
	Notice(message)
	return nil
}

// Fail prints message as an error annotation and returns Failed.
func Fail(message string) error {
	Error(message)
	return Failed
}
