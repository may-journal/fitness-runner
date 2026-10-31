// Command fitness-install fetches verified Fitness binaries and runs them.
//
//fitness:container tools
//fitness:component tool It ships beside the bundle as its own binary, outside any run.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/may-journal/fitness-runner/go/internal/distribution"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
	"github.com/may-journal/fitness-runner/go/internal/report"
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
	return runInstalled(bin, options)
}

func runInstalled(bin string, options distribution.Options) int {
	if options.InstallHook {
		return installHook(options)
	}
	if len(options.LinkSkills) > 0 {
		return linkSkills(bin, options)
	}
	return runChecks(bin, options)
}

func runChecks(bin string, options distribution.Options) int {
	if options.GitHubAction {
		if err := distribution.WriteActionFiles(bin); err != nil {
			return failure(err)
		}
	}
	if options.InstallOnly {
		fmt.Println(bin)
		report.Outcome("fitness-install", "Verified Fitness binaries installed successfully.")
		return 0
	}
	runner := filepath.Join(bin, "fitness")
	return failure(syscall.Exec(runner, append([]string{runner}, options.Args...), runnerEnvironment(bin)))
}

func linkSkills(bin string, options distribution.Options) int {
	if err := distribution.LinkSkills(bin, options.LinkSkills, os.Stdout); err != nil {
		return failure(err)
	}
	report.Outcome("fitness-install", "Linked the Fitness skills successfully.")
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
