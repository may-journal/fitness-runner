// Command fitness-release builds, publishes, and verifies release assets.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/may-journal/fitness-runner/go/internal/release"
	"github.com/may-journal/fitness-runner/go/internal/report"
)

func main() { os.Exit(command(os.Args[1:])) }

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

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("use build, smoke, verify-download, verify-tag, publish, promote, ensure-tag, pin-release, or tag")
	}
	flags := flag.NewFlagSet("fitness-release", flag.ContinueOnError)
	root := flags.String("root", ".", "repository root")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	config, err := release.New(*root)
	if err != nil {
		return err
	}
	return dispatch(args[0], config)
}

func dispatch(command string, config release.Config) error {
	handlers := map[string]func() error{
		"ensure-tag":      func() error { return config.EnsureTag(context.Background()) },
		"pin-release":     func() error { return config.PinRelease(context.Background()) },
		"promote":         func() error { return config.Promote(context.Background()) },
		"build":           func() error { return config.Build(context.Background()) },
		"smoke":           func() error { return config.Smoke(context.Background(), false) },
		"verify-download": func() error { return config.Smoke(context.Background(), true) },
		"verify-tag":      func() error { return config.VerifyTag(os.Getenv("GITHUB_REF_NAME")) },
		"publish":         func() error { return config.Publish(context.Background()) },
		"tag":             func() error { version, err := config.Version(); fmt.Println("go/v" + version); return err },
	}
	handler, ok := handlers[command]
	if !ok {
		return fmt.Errorf("unknown release command %q", command)
	}
	return handler()
}
