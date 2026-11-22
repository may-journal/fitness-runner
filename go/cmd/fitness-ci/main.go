// Command fitness-ci runs the steps of this repo's workflows and shared
// actions, one subcommand per step job.
//
// A job builds it once into bin/, and each step calls `fitness-ci <name>`
// on one line, so the logic is typed and shared.
package main

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/pkg/actions"
)

// subcommand is one step job: what it does, the step outputs it writes, and
// the code that runs it.
type subcommand struct {
	summary string
	outputs []string
	run     func(step actions.Step, args []string) error
}

// commands are every step job, by the name a workflow calls.
var commands = map[string]subcommand{}

// register adds a subcommand; each file registers the ones it holds.
func register(name, summary string, outputs []string, run func(actions.Step, []string) error) {
	commands[name] = subcommand{summary, outputs, run}
}

func main() {
	os.Exit(command(os.Args[1:]))
}

// command runs the subcommand args name and returns the exit code.
func command(args []string) int {
	if len(args) == 0 {
		fmt.Print(usage())
		return 2
	}
	if slices.Contains([]string{"help", "-h", "--help"}, args[0]) {
		fmt.Print(usage())
		return 0
	}
	sub, ok := commands[args[0]]
	if !ok {
		fmt.Fprintf(os.Stderr, "fitness-ci: unknown subcommand %q\n%s", args[0], usage())
		return 2
	}
	return exitCode(args[0], sub.run(actions.Step{Outputs: sub.outputs}, args[1:]))
}

// exitCode is the code a subcommand that returned err exits with, printing
// err unless the step already said why it failed.
func exitCode(name string, err error) int {
	var cli *actions.CLIError
	switch {
	case err == nil:
		return 0
	case errors.Is(err, actions.Failed):
		return 1
	case errors.As(err, &cli):
		// A program the step ran already printed why it failed.
		return max(cli.Code, 1)
	}
	fmt.Fprintf(os.Stderr, "fitness-ci %s: %v\n", name, err)
	return 1
}

// usage lists every subcommand with the outputs it writes and what it does.
func usage() string {
	var b strings.Builder
	b.WriteString("Usage: fitness-ci <subcommand> [args]\n\n")
	for _, name := range slices.Sorted(maps.Keys(commands)) {
		b.WriteString(usageLine(name, commands[name]))
	}
	return b.String()
}

// usageLine is one subcommand's entry in the usage.
func usageLine(name string, sub subcommand) string {
	if len(sub.outputs) > 0 {
		name += " (outputs: " + strings.Join(sub.outputs, ", ") + ")"
	}
	return "  " + name + "\n      " + sub.summary + "\n"
}

// env reads the environment variable name.
func env(name string) string { return os.Getenv(name) }

// firstError runs each step in order and returns the first error.
func firstError(steps ...func() error) error {
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}
