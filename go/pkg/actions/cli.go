package actions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// CLIError is a program that exited non-zero, with what it printed to stderr
// or, for a combined call, to both streams.
type CLIError struct {
	Program string
	Code    int
	Output  string
}

func (e *CLIError) Error() string {
	if e.Output == "" {
		return fmt.Sprintf("%s exited %d", e.Program, e.Code)
	}
	return fmt.Sprintf("%s exited %d: %s", e.Program, e.Code, e.Output)
}

// Call is one program run with extra environment.
type Call struct {
	// Env holds KEY=value pairs added to the step's environment.
	Env []string
	// Dir is the folder it runs in; empty is the step's.
	Dir string
}

// Capture runs name with args and returns its stdout, trailing newlines
// dropped. Stderr passes through to the log.
func (c Call) Capture(name string, args ...string) (string, error) {
	var out bytes.Buffer
	err := c.run(name, args, &out, os.Stderr)
	return trim(out.String()), err
}

// Quiet runs name with args and returns its stdout and stderr apart, trailing
// newlines dropped, printing neither.
func (c Call) Quiet(name string, args ...string) (stdout, stderr string, err error) {
	var out, errs bytes.Buffer
	err = c.run(name, args, &out, &errs)
	if cli := (*CLIError)(nil); errors.As(err, &cli) {
		cli.Output = trim(errs.String())
	}
	return trim(out.String()), trim(errs.String()), err
}

// Combined runs name with args and returns stdout and stderr together, as
// `2>&1` captures them, trailing newlines dropped.
func (c Call) Combined(name string, args ...string) (string, error) {
	var out bytes.Buffer
	err := c.run(name, args, &out, &out)
	if cli := (*CLIError)(nil); errors.As(err, &cli) {
		cli.Output = trim(out.String())
	}
	return trim(out.String()), err
}

// Run runs name with args, both streams passed through to the log.
func (c Call) Run(name string, args ...string) error {
	return c.run(name, args, os.Stdout, os.Stderr)
}

// run starts name with args, writing its streams to stdout and stderr.
func (c Call) run(name string, args []string, stdout, stderr io.Writer) error {
	cmd := exec.Command(name, args...)
	cmd.Dir, cmd.Stdout, cmd.Stderr = c.Dir, stdout, stderr
	if len(c.Env) > 0 {
		cmd.Env = append(os.Environ(), c.Env...)
	}
	err := cmd.Run()
	if exit := (*exec.ExitError)(nil); errors.As(err, &exit) {
		return &CLIError{Program: name, Code: exit.ExitCode()}
	}
	return err
}

// trim drops trailing newlines, as `$(...)` does.
func trim(s string) string {
	return strings.TrimRight(s, "\n")
}

// Gh runs gh with args and returns its stdout.
func Gh(args ...string) (string, error) {
	return Call{}.Capture("gh", args...)
}

// GhJSON runs gh with args and decodes its stdout into v.
func GhJSON(v any, args ...string) error {
	out, err := Gh(args...)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(out), v)
}

// GhPages runs a `gh api --paginate` call, whose stdout is one JSON array per
// page, and returns every page's items in order.
func GhPages[T any](args ...string) ([]T, error) {
	out, err := Gh(args...)
	if err != nil {
		return nil, err
	}
	return Pages[T](out)
}

// Pages decodes text holding one JSON array after another into their items.
func Pages[T any](text string) ([]T, error) {
	var all []T
	decoder := json.NewDecoder(strings.NewReader(text))
	for {
		var page []T
		err := decoder.Decode(&page)
		if errors.Is(err, io.EOF) {
			return all, nil
		}
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
	}
}

// Git runs git with args and returns its stdout.
func Git(args ...string) (string, error) {
	return Call{}.Capture("git", args...)
}

// Run runs name with args, both streams passed through to the log.
func Run(name string, args ...string) error {
	return Call{}.Run(name, args...)
}

// MakeExecutable sets the run bits on path, which an artifact download drops.
func MakeExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	return os.Chmod(path, info.Mode()|0o111)
}

// RunTool makes path executable, then runs it with args, as a step runs a
// program it downloaded.
func RunTool(path string, args ...string) error {
	if err := MakeExecutable(path); err != nil {
		return err
	}
	return Run(path, args...)
}
