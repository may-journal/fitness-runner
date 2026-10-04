// Command fitness-release builds, publishes, and verifies release assets.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/may-journal/fitness-runner/go/internal/release"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("use build, smoke, verify-download, verify-tag, publish, or tag")
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
