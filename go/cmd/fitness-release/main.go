// Command fitness-release builds, publishes, and verifies release assets.
// cspell:ignore releasemeta
//
//fitness:container tools
//fitness:component tool It builds and publishes the bundles, outside any run.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/minutes"
	"github.com/may-journal/fitness-runner/go/internal/release"
	"github.com/may-journal/fitness-runner/go/internal/releasemeta"
	"github.com/may-journal/fitness-runner/go/internal/report"
	"github.com/may-journal/fitness-runner/go/internal/usage"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == generatedCommand {
		checkGenerated(os.Args[2:])
	}
	os.Exit(command(os.Args[1:]))
}

func command(args []string) int {
	operation := "fitness-release"
	if len(args) > 0 {
		operation += " " + args[0]
	}
	if err := run(args); err != nil {
		reportFailure(operation, err)
		return 1
	}
	report.Outcome(operation, "Completed successfully.")
	return 0
}

func reportFailure(operation string, err error) {
	if report.Enabled() {
		report.Error(operation, err)
	} else {
		fmt.Fprintln(os.Stderr, err)
	}
}

// subcommand is one release command and the flags it reads. Every command
// reads --root; a flag it does not list draws a "not handled" warning.
type subcommand struct {
	flags []string
	run   func(config release.Config, root string) error
}

var commands = map[string]subcommand{
	"bundle-hashes":   {[]string{"root", "host"}, printBundleHashes},
	"assemble":        {[]string{"root", "host"}, func(c release.Config, _ string) error { return c.Assemble() }},
	"promote":         {[]string{"root"}, func(c release.Config, _ string) error { return c.Promote(context.Background()) }},
	"release-pr":      {[]string{"root"}, func(_ release.Config, root string) error { return updateReleasePR(root) }},
	"release-notes":   {[]string{"root"}, func(_ release.Config, root string) error { return publishNotes(root) }},
	"record":          {[]string{"root"}, func(c release.Config, _ string) error { return c.Record(context.Background(), os.Stdout) }},
	"smoke":           {[]string{"root"}, func(c release.Config, _ string) error { return c.Smoke(context.Background(), false) }},
	"verify-download": {[]string{"root"}, func(c release.Config, _ string) error { return c.Smoke(context.Background(), true) }},
	"verify-tag":      {[]string{"root"}, func(c release.Config, _ string) error { return c.VerifyTag(os.Getenv("GITHUB_REF_NAME")) }},
	"version":         {[]string{"root"}, func(c release.Config, _ string) error { return printVersion(c, "") }},
	"tag":             {[]string{"root"}, func(c release.Config, _ string) error { return printVersion(c, "v") }},
	"checks":          {[]string{"root"}, func(c release.Config, _ string) error { return c.WriteChecks() }},
	"minutes":         {[]string{"root", "days", "org", "repos"}, func(release.Config, string) error { return printMinutes() }},
}

// window is what minutes measures: the last days days of runs in repos, or
// in org's private repos when repos is empty.
var window struct {
	days  int
	org   string
	repos string
}

// printMinutes prints the jobs, billed minutes and median job time of each
// workflow in the window.
func printMinutes() error {
	var repos []string
	for _, r := range strings.Split(window.repos, ",") {
		if r = strings.TrimSpace(r); r != "" {
			repos = append(repos, r)
		}
	}
	return minutes.Print(context.Background(), os.Stdout, window.org, repos, window.days)
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("use bundle-hashes, assemble, smoke, verify-download, verify-tag, promote, release-pr, release-notes, record, minutes, version, tag, or checks")
	}
	flags := flag.NewFlagSet("fitness-release", flag.ContinueOnError)
	root := flags.String("root", ".", "repository root")
	host := flags.Bool("host", false, "bundle-hashes and assemble: expect build outputs for this platform only")
	flags.IntVar(&window.days, "days", 4, "minutes: how many days of runs to total")
	flags.StringVar(&window.org, "org", "", "minutes: total the org's private repos")
	flags.StringVar(&window.repos, "repos", "", "minutes: comma-separated owner/repo names to total instead")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	command, err := lookup(args[0], flags)
	if err != nil {
		return err
	}
	config, err := newConfig(*root, *host, command)
	if err != nil {
		return err
	}
	return command.run(config, *root)
}

// lookup finds name and warns about each flag it was given but does not read.
func lookup(name string, flags *flag.FlagSet) (subcommand, error) {
	command, ok := commands[name]
	if !ok {
		return command, fmt.Errorf("unknown release command %q", name)
	}
	usage.WarnUnhandled(os.Stderr, "fitness-release "+name, flags, command.flags)
	return command, nil
}

// newConfig reads the release config, narrowed to this platform with host
// when the command reads --host.
func newConfig(root string, host bool, command subcommand) (release.Config, error) {
	config, err := release.New(root)
	if err != nil || !host || !slices.Contains(command.flags, "host") {
		return config, err
	}
	return config.Host()
}

func printBundleHashes(config release.Config, _ string) error {
	hashes, err := config.BundleHashes()
	if err == nil {
		fmt.Println(hashes)
	}
	return err
}

func printVersion(config release.Config, prefix string) error {
	version, err := config.Version()
	if err == nil {
		fmt.Println(prefix + version)
	}
	return err
}

func updateReleasePR(root string) error {
	var value struct {
		Number int `json:"number"`
	}
	if err := json.Unmarshal([]byte(os.Getenv("RELEASE_PR")), &value); err != nil {
		return err
	}
	client := releasemeta.Client{Root: root, Repo: os.Getenv("GITHUB_REPOSITORY")}
	return client.UpdateReleasePR(context.Background(), value.Number)
}

// publishNotes writes the description of the release GITHUB_REF_NAME tags
// from the entry files added since the previous release tag.
func publishNotes(root string) error {
	client := releasemeta.Client{Root: root, Repo: os.Getenv("GITHUB_REPOSITORY")}
	return client.PublishNotes(context.Background(), os.Getenv("GITHUB_REF_NAME"))
}
