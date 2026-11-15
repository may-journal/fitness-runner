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
	"regexp"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/may-journal/fitness-runner/go/internal/distribution"
	"github.com/may-journal/fitness-runner/go/internal/report"
	"github.com/may-journal/fitness-runner/go/pkg/gitx"
)

// Release publishing embeds the exact version and platform bundle hashes. A
// plain `go install` embeds neither, so installerVersion reads the module
// version Go recorded instead.
var version string
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
	installer, err := distribution.New(installerVersion(), bundleHashes, os.Stderr)
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
	return runInstalled(bin, checks, options)
}

// installChecks installs each check the repo pins when asked, reading the
// pins from the installed engine, and returns their folder.
func installChecks(installer *distribution.Installer, bin string, options distribution.Options) (string, error) {
	if !options.InstallsChecks() || options.SkipsChecks() {
		return "", nil
	}
	versions := exec.Command(filepath.Join(bin, "fitness"), "check-versions")
	versions.Stderr = os.Stderr
	out, err := versions.Output()
	if err != nil {
		return "", fmt.Errorf("fitness check-versions: %w", err)
	}
	pins, err := distribution.ParsePins(string(out))
	if err != nil {
		return "", err
	}
	return installer.InstallChecks(context.Background(), pins)
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
	return distribution.ParseOptions(args, installerVersion(), os.Stderr)
}

func installHook(options distribution.Options) int {
	if err := distribution.InstallHook(options); err != nil {
		return failure(err)
	}
	fmt.Println("fitness: installed the Go pre-commit hook")
	report.Outcome("fitness-install", "Installed the Go pre-commit hook successfully.")
	return 0
}

// installerVersion is the version release publishing stamped, else the
// released module version a `go install` recorded, such as v1.69.1. A build
// from a checkout or a commit gets latest.
func installerVersion() string {
	if version != "" {
		return version
	}
	// A downloaded module carries its checksum; a checkout's version stamp
	// from git tags does not, so it reads as latest.
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Sum != "" && releaseVersion.MatchString(info.Main.Version) {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return "latest"
}

// releaseVersion matches a released module version such as v1.69.1, and not
// (devel) or a pseudo-version.
var releaseVersion = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)
