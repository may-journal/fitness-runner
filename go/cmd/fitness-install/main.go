// Command fitness-install fetches verified Fitness binaries and runs them.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/may-journal/fitness-runner/go/internal/distribution"
	"github.com/may-journal/fitness-runner/go/internal/report"
	"github.com/may-journal/fitness-runner/go/pkg/gitx"
)

// Release publishing embeds the exact version and platform bundle hashes.
var version = "0.20261004.1140"
var bundleHashes string

func main() { os.Exit(run(os.Args[1:])) }

// run installs and runs Fitness. Every git, ssh and gh it starts, such as
// `gh auth token`, fails instead of asking for a password.
func run(args []string) int {
	gitx.DisablePrompts(".")
	options, err := commandOptions(args)
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if err != nil {
		return failure(err)
	}
	options, err = distribution.ActionOptions(options)
	if err != nil {
		return failure(err)
	}
	installer, err := distribution.New(version, bundleHashes, os.Stderr)
	if err != nil {
		return failure(err)
	}
	return execute(installer, options)
}

func execute(installer *distribution.Installer, options distribution.Options) int {
	bin, err := installer.Install(context.Background(), options.Version)
	if err != nil {
		return failure(err)
	}
	checks, err := installChecks(installer, bin, options)
	if err != nil && options.NeedsChecks() {
		return failure(err)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "fitness: warning: %v; only the checks the engine runs itself run\n", err)
	}
	return runInstalled(bin, checks, options)
}

// installChecks installs each check the repo pins when asked, and returns
// their folder. The installed engine names the pins and its own checks.
func installChecks(installer *distribution.Installer, bin string, options distribution.Options) (string, error) {
	if !options.InstallsChecks() || options.SkipsChecks() {
		return "", nil
	}
	out, err := engineOutput(bin, "check-versions")
	if err != nil {
		return "", err
	}
	pins, err := distribution.ParsePins(out)
	if err != nil {
		return "", err
	}
	builtins, err := engineOutput(bin, "check", "--list")
	if err != nil {
		return "", err
	}
	installer.EngineChecks = strings.Fields(builtins)
	return installer.InstallChecks(context.Background(), pins)
}

// engineOutput runs the installed engine with args and returns what it
// printed on stdout.
func engineOutput(bin string, args ...string) (string, error) {
	cmd := exec.Command(filepath.Join(bin, "fitness"), args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("fitness %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

func runInstalled(bin, checks string, options distribution.Options) int {
	if options.InstallHook {
		return installHook(options)
	}
	if options.SetsUpAgents() {
		return setUpAgents(bin, options)
	}
	return runChecks(bin, checks, options)
}

func runChecks(bin, checks string, options distribution.Options) int {
	if options.GitHubAction {
		if err := distribution.WriteActionFiles(bin, checks); err != nil {
			return failure(err)
		}
	}
	if options.InstallOnly {
		fmt.Println(bin)
		if checks != "" {
			fmt.Println(checks)
		}
		report.Outcome("fitness-install", "Verified Fitness binaries installed successfully.")
		return 0
	}
	runner := filepath.Join(bin, "fitness")
	return failure(syscall.Exec(runner, append([]string{runner}, options.Args...), runnerEnvironment(bin, checks)))
}

func setUpAgents(bin string, options distribution.Options) int {
	if len(options.LinkSkills) > 0 {
		if err := distribution.LinkSkills(bin, options.LinkSkills, os.Stdout); err != nil {
			return failure(err)
		}
		report.Outcome("fitness-install", "Linked the Fitness skills successfully.")
	}
	if len(options.ClaudeSettings) > 0 {
		if err := distribution.WriteClaudeSettings(bin, options.ClaudeSettings, os.Stdout); err != nil {
			return failure(err)
		}
		report.Outcome("fitness-install", "Added the Fitness hooks to Claude Code successfully.")
	}
	return 0
}

func failure(err error) int {
	if report.Enabled() {
		report.Error("fitness-install", err)
	} else {
		fmt.Fprintln(os.Stderr, "fitness:", err)
	}
	return 1
}

func commandOptions(args []string) (distribution.Options, error) {
	if filepath.Base(os.Args[0]) == "pre-commit" {
		return distribution.ReadHookOptions()
	}
	return distribution.ParseOptions(args, version, os.Stderr)
}

func installHook(options distribution.Options) int {
	if err := distribution.InstallHook(options); err != nil {
		return failure(err)
	}
	fmt.Println("fitness: installed the Go pre-commit hook")
	report.Outcome("fitness-install", "Installed the Go pre-commit hook successfully.")
	return 0
}
